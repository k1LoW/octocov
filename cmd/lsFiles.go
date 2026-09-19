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
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/coverage"
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
		if c.Coverage == nil {
			return errors.New("coverage: is not set")
		}
		r, err := report.New(c.Repository)
		if err != nil {
			return err
		}
		if err := r.MeasureCoverage(c.Coverage.Paths, c.Coverage.Exclude); err != nil {
			return err
		}
		if len(r.Coverage.Files) == 0 {
			return nil
		}
		sort.Slice(r.Coverage.Files, func(i int, j int) bool {
			return r.Coverage.Files[i].EffectivePath() < r.Coverage.Files[j].EffectivePath()
		})

		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		root, err := internal.RootPath(wd)
		if err != nil {
			return err
		}
		// MeasureCoverage normalizes against the git root, so a repository identified by its
		// config file alone leaves every entry unresolved. Re-normalizing against root picks
		// those up, and is skipped when nothing is unresolved so the walk is not repeated.
		if slices.ContainsFunc(r.Coverage.Files, func(fc *coverage.FileCoverage) bool {
			return fc.NormalizedPath == ""
		}) {
			if files, err := internal.CollectFiles(root); err == nil {
				r.Coverage.NormalizePaths(root, files)
			}
		}
		rel, err := filepath.Rel(root, wd)
		if err != nil {
			return err
		}

		scope := filepath.ToSlash(rel)
		rows, unresolved := lsFilesRows(r.Coverage.Files, scope)

		t := 0
		for _, fr := range rows {
			if fr.total > t {
				t = fr.total
			}
		}
		w := len(strconv.Itoa(t))*2 + 1
		for _, fr := range rows {
			cover := float64(fr.covered) / float64(fr.total) * 100
			if fr.total == 0 {
				cover = 0.0
			}
			cl := c.CoverageColor(cover)
			c, err := detectTermColor(cl)
			if err != nil {
				return err
			}
			cmd.Printf("%s [%s] %s\n", c.Sprint(fmt.Sprintf("%5s%%", fmt.Sprintf("%.1f", floor1(cover)))), fmt.Sprintf(fmt.Sprintf("%%%ds", w), fmt.Sprintf("%d/%d", fr.covered, fr.total)), fr.path)
		}
		if len(unresolved) > 0 {
			cmd.PrintErrf("%d file(s) in the report could not be placed under %s and were not listed: %s\n", len(unresolved), scope, strings.Join(unresolved, ", "))
		}

		return nil
	},
}

type lsFilesRow struct {
	path           string
	covered, total int
}

// lsFilesRows lists the files of a report that sit in scope, a slash-separated directory
// relative to the repository root, with their paths relative to it. It also returns the
// entries that could not be placed, which the caller reports rather than dropping in silence.
func lsFilesRows(files coverage.FileCoverages, scope string) ([]lsFilesRow, []string) {
	if scope == "." {
		scope = ""
	}
	var (
		rows       []lsFilesRow
		unresolved []string
	)
	for _, f := range files {
		p := f.NormalizedPath
		if p == "" {
			// Nothing on disk answers to this entry, so which directory it belongs to is
			// unknown. At the root every entry is listed, under its path as the report writes
			// it; below the root it cannot be placed and is reported instead.
			if scope != "" {
				unresolved = append(unresolved, f.File)
				continue
			}
			p = path.Clean(filepath.ToSlash(f.File))
		} else {
			p = path.Clean(filepath.ToSlash(p))
			if !underDir(p, scope) {
				continue
			}
			p = strings.TrimPrefix(strings.TrimPrefix(p, scope), "/")
		}
		rows = append(rows, lsFilesRow{path: p, covered: f.Covered, total: f.Total})
	}
	return rows, unresolved
}

// underDir reports whether the slash-separated path p sits below dir. An empty dir is the root,
// which everything is below. The comparison is per segment, so cmd/apple is not read as being
// below cmd/app, and dir itself is not below dir, which keeps a path equal to it from being
// listed under an empty name.
func underDir(p, dir string) bool {
	if dir == "" {
		return true
	}
	return strings.HasPrefix(p, dir+"/")
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
