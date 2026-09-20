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
	"bytes"
	"context"
	"io"
	"math"
	"os"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/report"
	"github.com/spf13/cobra"
)

var outPath string

// badgeCmd represents the badge command.
var badgeCmd = &cobra.Command{
	Use:       "badge",
	Short:     "generate badge",
	Long:      `generate badge.`,
	Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	ValidArgs: []string{"coverage", "ratio", "time"},
	RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	},
}

// coverage subcommand.
var badgeCoverageCmd = &cobra.Command{
	Use:   "coverage",
	Short: "generate coverage badge",
	RunE: func(_ *cobra.Command, _ []string) error {
		c, r, err := loadConfigAndReport(configPath)
		if err != nil {
			return err
		}
		if err := c.CoverageConfigReady(); err != nil {
			return err
		}
		if err := r.MeasureCoverage(c.Coverage.Paths, c.Coverage.Exclude); err != nil {
			return err
		}
		return writeOut(outPath, func(w io.Writer) error {
			return c.Coverage.Badge.RenderCoverage(w, r.CoveragePercent())
		})
	},
}

// ratio subcommand.
var badgeRatioCmd = &cobra.Command{
	Use:   "ratio",
	Short: "generate code to test ratio badge",
	RunE: func(cmd *cobra.Command, _ []string) error {
		c, r, err := loadConfigAndReport(configPath)
		if err != nil {
			return err
		}
		if !c.Loaded() {
			cmd.PrintErrf("%s are not found\n", strings.Join(config.Paths, " and "))
		}
		if err := c.CodeToTestRatioConfigReady(); err != nil {
			return err
		}
		if err := r.MeasureCodeToTestRatio(c.Root(), c.CodeToTestRatio.Code, c.CodeToTestRatio.Test); err != nil {
			return err
		}
		return writeOut(outPath, func(w io.Writer) error {
			return c.CodeToTestRatio.Badge.RenderCodeToTestRatio(w, r.CodeToTestRatioRatio())
		})
	},
}

// time subcommand.
var badgeTimeCmd = &cobra.Command{
	Use:   "time",
	Short: "generate test execution time badge",
	RunE: func(_ *cobra.Command, _ []string) error {
		c, r, err := loadConfigAndReport(configPath)
		if err != nil {
			return err
		}
		if err := c.TestExecutionTimeConfigReady(); err != nil {
			return err
		}
		var stepNames []string
		if len(c.TestExecutionTime.Steps) > 0 {
			stepNames = c.TestExecutionTime.Steps
		}
		if err := r.MeasureTestExecutionTime(context.Background(), stepNames); err != nil {
			return err
		}
		return writeOut(outPath, func(w io.Writer) error {
			return c.TestExecutionTime.Badge.RenderTestExecutionTime(w, r.TestExecutionTimeNano())
		})
	},
}

// loadConfigAndReport load config and create report.
func loadConfigAndReport(cfgPath string) (*config.Config, *report.Report, error) {
	c := config.New()
	if err := c.Load(cfgPath); err != nil {
		return nil, nil, err
	}
	c.Build()
	r, err := report.New(c.Repository)
	if err != nil {
		return nil, nil, err
	}
	return c, r, nil
}

// writeOut writes what render produced to path, or to stdout when no path is given. The badge
// is rendered into memory first, so a run that fails on a configuration error leaves whatever
// badge is already on disk rather than truncating it to nothing.
func writeOut(path string, render func(io.Writer) error) error {
	buf := new(bytes.Buffer)
	if err := render(buf); err != nil {
		return err
	}
	if path == "" {
		_, err := os.Stdout.Write(buf.Bytes())
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644) // #nosec
}

// floor1 round down to one decimal place.
func floor1(v float64) float64 {
	return math.Floor(v*10) / 10
}

// setBadgeFlags set flags for badge subcommands.
func setBadgeFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&configPath, "config", "", "", "config file path")
	cmd.Flags().StringVarP(&outPath, "out", "", "", "output file path")
}

func init() {
	rootCmd.AddCommand(badgeCmd)
	badgeCmd.AddCommand(badgeCoverageCmd, badgeRatioCmd, badgeTimeCmd)
	setBadgeFlags(badgeCoverageCmd)
	setBadgeFlags(badgeRatioCmd)
	setBadgeFlags(badgeTimeCmd)
}
