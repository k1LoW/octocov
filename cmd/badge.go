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
	"fmt"
	"os"
	"path/filepath"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/k1LoW/octocov/report"
	"github.com/spf13/cobra"
)

const (
	// https://github.com/badges/shields/blob/7d452472defa0e0bd71d6443393e522e8457f856/badge-maker/lib/color.js#L8-L12
	green       = "#97CA00"
	yellowgreen = "#A4A61D"
	yellow      = "#DFB317"
	orange      = "#FE7D37"
	red         = "#E05D44"
)

// badgeCmd represents the badge command
var badgeCmd = &cobra.Command{
	Use:   "badge",
	Short: "Generate coverage report badge",
	Long:  `Generate coverage report badge.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := config.New()
		if err := c.Load(configPath); err != nil {
			return err
		}
		path := c.Coverage.Path
		if len(args) > 0 {
			path = args[0]
		}
		r := report.New()
		if err := r.MeasureCoverage(path); err != nil {
			return err
		}

		cover := float64(r.Coverage.Covered) / float64(r.Coverage.Total) * 100
		var (
			o   *os.File
			err error
		)
		if out == "" {
			out = c.Badge.Path
		}
		if out == "" {
			o = os.Stdout
		} else {
			o, err = os.OpenFile(filepath.Clean(out), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
			if err != nil {
				return err
			}
		}
		b := badge.New("coverage", fmt.Sprintf("%.1f%%", cover))
		b.ValueColor = coverageColor(cover)
		if err := b.Render(o); err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(badgeCmd)
	badgeCmd.Flags().StringVarP(&out, "out", "o", "", "output file path")
	badgeCmd.Flags().StringVarP(&configPath, "config", "c", "", "config file path")
}

func coverageColor(cover float64) string {
	switch {
	case cover >= 80.0:
		return green
	case cover >= 60.0:
		return yellowgreen
	case cover >= 40.0:
		return yellow
	case cover >= 20.0:
		return orange
	default:
		return red
	}
}
