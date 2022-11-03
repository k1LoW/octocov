/*
Copyright © 2022 Ken'ichiro Oyama <k1lowxb@gmail.com>

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
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/internal"
	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/k1LoW/octocov/report"
	"github.com/spf13/cobra"
)

const (
	badgeCoverage = "coverage"
	badgeRatio    = "ratio"
	badgeTime     = "time"
)

var outPath string

// badgeCmd represents the badge command
var badgeCmd = &cobra.Command{
	Use:       "badge",
	Short:     "generate badge",
	Long:      `generate badge.`,
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{badgeCoverage, badgeRatio, badgeTime},
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c := config.New()
		if err := c.Load(configPath); err != nil {
			return err
		}
		c.Build()

		r, err := report.New(c.Repository)
		if err != nil {
			return err
		}

		var out io.Writer
		if outPath != "" {
			file, err := os.OpenFile(outPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
			if err != nil {
				return err
			}
			defer func() {
				if err := file.Close(); err != nil {
					os.Exit(1)
				}
			}()
			out = file
		} else {
			out = os.Stdout
		}

		switch args[0] {
		case badgeCoverage:
			if err := c.CoverageConfigReady(); err != nil {
				return err
			}
			if err := r.MeasureCoverage(c.Coverage.Paths); err != nil {
				return err
			}
			cp := r.CoveragePercent()
			b := badge.New("coverage", fmt.Sprintf("%.1f%%", cp))
			b.MessageColor = c.CoverageColor(cp)
			if err := b.AddIcon(internal.Icon); err != nil {
				return err
			}
			if err := b.Render(out); err != nil {
				return err
			}
		case badgeRatio:
			if !c.Loaded() {
				cmd.PrintErrf("%s are not found\n", strings.Join(config.DefaultConfigFilePaths, " and "))
			}
			if err := c.CodeToTestRatioConfigReady(); err != nil {
				return err
			}
			if err := r.MeasureCodeToTestRatio(c.Root(), c.CodeToTestRatio.Code, c.CodeToTestRatio.Test); err != nil {
				return err
			}
			tr := r.CodeToTestRatioRatio()
			b := badge.New("code to test ratio", fmt.Sprintf("1:%.1f", tr))
			b.MessageColor = c.CodeToTestRatioColor(tr)
			if err := b.AddIcon(internal.Icon); err != nil {
				return err
			}
			if err := b.Render(out); err != nil {
				return err
			}
		case badgeTime:
			if err := c.TestExecutionTimeConfigReady(); err != nil {
				return err
			}
			stepNames := []string{}
			if len(c.TestExecutionTime.Steps) > 0 {
				stepNames = c.TestExecutionTime.Steps
			}
			if err := r.MeasureTestExecutionTime(ctx, stepNames); err != nil {
				return err
			}
			d := time.Duration(r.TestExecutionTimeNano())
			b := badge.New("test execution time", d.String())
			b.MessageColor = c.TestExecutionTimeColor(d)
			if err := b.AddIcon(internal.Icon); err != nil {
				return err
			}
			if err := b.Render(out); err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(badgeCmd)
	badgeCmd.Flags().StringVarP(&configPath, "config", "", "", "config file path")
	badgeCmd.Flags().StringVarP(&outPath, "out", "", "", "output file path")
}
