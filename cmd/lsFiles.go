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

// lsFilesCmd represents the lsFiles command.
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
		if reportPath != "" {
			c.Coverage.Paths = []string{reportPath}
			c.CodeToTestRatio = nil
			c.TestExecutionTime = nil
		}
		if err := c.CoverageConfigReady(); err != nil {
			return err
		}
		r, err := report.New(c.Repository)
		if err != nil {
			return err
		}
		if err := r.MeasureCoverage(c.Coverage.Paths); err != nil {
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
		gitRoot, err := internal.RootPath(wd)
		if err != nil {
			return err
		}
		var cfiles []string
		for _, f := range r.Coverage.Files {
			cfiles = append(cfiles, f.File)
		}
		var files []string
		if err := filepath.Walk(wd, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() && strings.Contains(path, ".git/") {
				return filepath.SkipDir
			}
			if !info.IsDir() && !strings.Contains(path, ".git/") {
				files = append(files, path)
			}
			return nil
		}); err != nil {
			return err
		}
		sort.Slice(files, func(i int, j int) bool {
			return files[i] < files[j]
		})

		prefix := internal.DetectPrefix(gitRoot, wd, files, cfiles)
		for _, f := range r.Coverage.Files {
			p := filepath.Clean(f.File)
			if !strings.HasPrefix(p, prefix) {
				continue
			}
			trimed := strings.TrimPrefix(strings.TrimPrefix(p, prefix), "/")
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
			cmd.Printf("%s [%s] %s\n", c.Sprint(fmt.Sprintf("%5s%%", fmt.Sprintf("%.1f", cover))), fmt.Sprintf(fmt.Sprintf("%%%ds", w), fmt.Sprintf("%d/%d", f.Covered, f.Total)), trimed)
		}

		return nil
	},
}

func detectTermColor(cl string) (*color.Color, error) {
	termGreen, err := colorful.Hex("#4e9a06")
	if err != nil {
		return nil, err
	}
	termYellow, err := colorful.Hex("#c4a000")
	if err != nil {
		return nil, err
	}
	termRed, err := colorful.Hex("#cc0000")
	if err != nil {
		return nil, err
	}
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
	lsFilesCmd.Flags().StringVarP(&configPath, "config", "", "", "config file path")
	lsFilesCmd.Flags().StringVarP(&reportPath, "report", "r", "", "coverage report file path")
}
