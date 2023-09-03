/*
Copyright © 2021 Ken'ichiro Oyama <k1lowxb@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/report"
	"github.com/k1LoW/octocov/version"
	"github.com/spf13/cobra"
)

// dumpCmd represents the dump command.
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "dump report",
	Long:  `dump report.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		cmd.PrintErrf("%s version %s\n", version.Name, version.Version)
		c := config.New()
		if err := c.Load(configPath); err != nil {
			return err
		}
		c.Build()
		if reportPath != "" {
			c.Coverage.Paths = []string{reportPath}
			c.CodeToTestRatio = nil
			c.TestExecutionTime = nil
		}

		r, err := report.New(c.Repository)
		if err != nil {
			return err
		}

		if err := c.CoverageConfigReady(); err != nil {
			cmd.PrintErrf("Skip measuring code coverage: %v\n", err)
		} else {
			if err := r.MeasureCoverage(c.Coverage.Paths); err != nil {
				cmd.PrintErrf("Skip measuring code coverage: %v\n", err)
			}
		}

		if err := c.CodeToTestRatioConfigReady(); err != nil {
			cmd.PrintErrf("Skip measuring code to test ratio: %v\n", err)
		} else {
			if err := r.MeasureCodeToTestRatio(c.Root(), c.CodeToTestRatio.Code, c.CodeToTestRatio.Test); err != nil {
				cmd.PrintErrf("Skip measuring code to test ratio: %v\n", err)
			}
		}

		if err := c.TestExecutionTimeConfigReady(); err != nil {
			cmd.PrintErrf("Skip measuring test execution time: %v\n", err)
		} else {
			stepNames := []string{}
			if len(c.TestExecutionTime.Steps) > 0 {
				stepNames = c.TestExecutionTime.Steps
			}
			if err := r.MeasureTestExecutionTime(ctx, stepNames); err != nil {
				cmd.PrintErrf("Skip measuring test execution time: %v\n", err)
			}
		}

		if err := r.CollectCustomMetrics(); err != nil {
			cmd.PrintErrf("Skip collecting custom metrics: %v\n", err)
		}

		if r.CountMeasured() == 0 {
			return errors.New("nothing could be measured")
		}

		if err := r.Validate(); err != nil {
			return fmt.Errorf("validation error: %w", err)
		}
		cmd.Println(r.String())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(dumpCmd)
	dumpCmd.Flags().StringVarP(&configPath, "config", "", "", "config file path")
	dumpCmd.Flags().StringVarP(&reportPath, "report", "r", "", "coverage report file path")
}
