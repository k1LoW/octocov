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
	"os"

	"github.com/k1LoW/octocov/report"
	"github.com/spf13/cobra"
)

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:     "diff [REPORT_A] [REPORT_B]",
	Short:   "compare reports (code coverage report or octocov report.json)",
	Long:    `compare reports (code coverage report or octocov report.json).`,
	Aliases: []string{"compare"},
	Args:    cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		a := &report.Report{}
		if err := a.Load(args[0]); err != nil {
			return err
		}
		if a.Timestamp.IsZero() {
			fi, err := os.Stat(args[0])
			if err != nil {
				return err
			}
			a.Timestamp = fi.ModTime()
		}

		b := &report.Report{}
		if err := b.Load(args[1]); err != nil {
			return err
		}
		if b.Timestamp.IsZero() {
			fi, err := os.Stat(args[1])
			if err != nil {
				return err
			}
			b.Timestamp = fi.ModTime()
		}

		a.Compare(b).Out(os.Stdout)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
