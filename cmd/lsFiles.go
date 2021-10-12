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
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/internal"
	"github.com/k1LoW/octocov/report"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/spf13/cobra"
)

// lsFilesCmd represents the lsFiles command
var lsFilesCmd = &cobra.Command{
	Use:   "ls-files",
	Short: "list files logged in code coverage report",
	Long:  `list files logged in code coverage report.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := config.New()
		if err := c.Load(configPath); err != nil {
			return err
		}
		c.Build()
		if err := c.CoverageConfigReady(); err != nil {
			return err
		}
		r, err := report.New()
		if err != nil {
			return err
		}
		path := c.Coverage.Path
		if reportPath != "" {
			path = reportPath
		}
		if err := r.MeasureCoverage(path); err != nil {
			return err
		}
		t := 0
		sort.Slice(r.Coverage.Files, func(i int, j int) bool {
			if r.Coverage.Files[i].Total > t {
				t = r.Coverage.Files[i].Total
			}
			return r.Coverage.Files[i].File < r.Coverage.Files[j].File
		})

		if len(r.Coverage.Files) == 0 {
			return nil
		}
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		gitRoot, err := internal.GetRootPath(wd)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(gitRoot, wd)
		if err != nil {
			return err
		}
		rel2 := filepath.Clean(rel)
		for _, f := range r.Coverage.Files {
			p := filepath.Clean(f.File)
			rel, err := filepath.Rel(rel2, f.File)
			if err != nil {
				continue
			}
			if strings.Contains(rel, "..") {
				continue
			}
			p = filepath.Clean(rel)
			cover := float64(f.Covered) / float64(f.Total) * 100
			if f.Total == 0 {
				cover = 0.0
			}
			cl := c.CoverageColor(cover)
			c, err := detectTermColor(cl)
			if err != nil {
				return err
			}
			w := len(strconv.Itoa(t))*2 + 1
			cmd.Printf("%s [%s] %s\n", c.Sprint(fmt.Sprintf("%5s%%", fmt.Sprintf("%.1f", cover))), fmt.Sprintf(fmt.Sprintf("%%%ds", w), fmt.Sprintf("%d/%d", f.Covered, f.Total)), p)
		}

		return nil
	},
}

func detectTermColor(cl string) (*color.Color, error) {
	termGreen, _ := colorful.Hex("#4e9a06")
	termYellow, _ := colorful.Hex("#c4a000")
	termRed, _ := colorful.Hex("#cc0000")
	tc, err := colorful.Hex(cl)
	if err != nil {
		return nil, err
	}
	dg := tc.DistanceLab(termGreen)
	dy := tc.DistanceLab(termYellow)
	dr := tc.DistanceLab(termRed)
	switch {
	case dg <= dy && dg <= dr:
		c := color.New(color.FgGreen)
		c.EnableColor()
		return c, nil
	case dy <= dg && dy <= dr:
		c := color.New(color.FgYellow)
		c.EnableColor()
		return c, nil
	default:
		c := color.New(color.FgRed)
		c.EnableColor()
		return c, nil
	}
}

func init() {
	rootCmd.AddCommand(lsFilesCmd)
	lsFilesCmd.Flags().StringVarP(&reportPath, "report", "r", "", "coverage report file path")
}
