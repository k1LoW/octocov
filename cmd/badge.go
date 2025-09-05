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
	"math"
	"os"
	"strings"
	"time"

	"github.com/k1LoW/octocov/badge"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/internal"
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
		out, cleanup, err := openOut(outPath)
		if err != nil {
			return err
		}
		defer cleanup()

		if err := c.CoverageConfigReady(); err != nil {
			return err
		}
		if err := r.MeasureCoverage(c.Coverage.Paths, c.Coverage.Exclude); err != nil {
			return err
		}
		cp := r.CoveragePercent()
		return renderBadgeWithIcon("coverage", fmt.Sprintf("%.1f%%", floor1(cp)), c.CoverageColor(cp), out)
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
		out, cleanup, err := openOut(outPath)
		if err != nil {
			return err
		}
		defer cleanup()

		if !c.Loaded() {
			cmd.PrintErrf("%s are not found\n", strings.Join(config.Paths, " and "))
		}
		if err := c.CodeToTestRatioConfigReady(); err != nil {
			return err
		}
		if err := r.MeasureCodeToTestRatio(c.Root(), c.CodeToTestRatio.Code, c.CodeToTestRatio.Test); err != nil {
			return err
		}
		tr := r.CodeToTestRatioRatio()
		return renderBadgeWithIcon("code to test ratio", fmt.Sprintf("1:%.1f", floor1(tr)), c.CodeToTestRatioColor(tr), out)
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
		out, cleanup, err := openOut(outPath)
		if err != nil {
			return err
		}
		defer cleanup()

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
		d := time.Duration(r.TestExecutionTimeNano())
		return renderBadgeWithIcon("test execution time", d.String(), c.TestExecutionTimeColor(d), out)
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

// openOut open output writer and return cleanup function.
func openOut(path string) (io.Writer, func(), error) {
	if path == "" {
		return os.Stdout, func() {}, nil
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		if err := file.Close(); err != nil {
			// keep the original behavior: exit on close error
			os.Exit(1)
		}
	}
	return file, cleanup, nil
}

// renderBadgeWithIcon render badge with icon.
func renderBadgeWithIcon(name, message, color string, out io.Writer) error {
	b := badge.New(name, message)
	b.MessageColor = color
	if err := b.AddIcon(internal.Icon); err != nil {
		return err
	}
	return b.Render(out)
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
