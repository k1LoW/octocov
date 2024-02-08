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
	"os"
	"path/filepath"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/report"
	"github.com/spf13/cobra"
)

// viewCmd represents the view command.
var viewCmd = &cobra.Command{
	Use:     "view [FILE ...]",
	Short:   "view code coverage of file",
	Long:    `view code coverage of file.`,
	Aliases: []string{"cat"},
	Args:    cobra.MinimumNArgs(1),
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
		for _, f := range args {
			err := func() error {
				if _, err := os.Stat(f); err != nil {
					return err
				}
				fp, err := os.Open(filepath.Clean(f))
				if err != nil {
					return err
				}
				fc, err := r.Coverage.Files.FuzzyFindByFile(f)
				if err != nil {
					fc = &coverage.FileCoverage{
						File: f,
					}
				}
				if err := coverage.NewPrinter(fc).Print(fp, os.Stdout); err != nil {
					_ = fp.Close() //nostyle:handlerrors
					return err
				}
				if err := fp.Close(); err != nil {
					return err
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(viewCmd)
	viewCmd.Flags().StringVarP(&configPath, "config", "", "", "config file path")
	viewCmd.Flags().StringVarP(&reportPath, "report", "r", "", "coverage report file path")
}
