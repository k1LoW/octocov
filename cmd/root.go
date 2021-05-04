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
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/k1LoW/octocov/report"
	"github.com/k1LoW/octocov/version"
	"github.com/spf13/cobra"
)

var (
	configPath string
	dump       bool
	genbadge   bool
)

var rootCmd = &cobra.Command{
	Use:          "octocov",
	Short:        "octocov is a tool for collecting code coverage",
	Long:         `octocov is a tool for collecting code coverage.`,
	Version:      version.Version,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 && args[0] == "completion" {
			return completionCmd(cmd, args[1:])
		}
		c := config.New()
		if err := c.Load(configPath); err != nil {
			return err
		}
		path := c.Coverage.Path
		r := report.New()
		if err := r.MeasureCoverage(path); err != nil {
			return err
		}
		c.Build()

		if dump {
			cmd.Println(r.String())
			return nil
		}

		// Generate badge
		if c.BadgeConfigReady() || genbadge {
			var out *os.File
			cp := r.CoveragePercent()
			if c.Coverage.Badge == "" {
				out = os.Stdout
			} else {
				cmd.PrintErrln("Generate coverage report badge...")
				err := os.MkdirAll(filepath.Dir(c.Coverage.Badge), 0755) // #nosec
				if err != nil {
					return err
				}
				out, err = os.OpenFile(filepath.Clean(c.Coverage.Badge), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
				if err != nil {
					return err
				}
			}

			b := badge.New("coverage", fmt.Sprintf("%.1f%%", cp))
			b.ValueColor = coverageColor(cp)
			if err := b.Render(out); err != nil {
				return err
			}
			if genbadge {
				return nil
			}
		}

		// Store report
		if c.DatastoreConfigReady() {
			cmd.PrintErrln("Store coverage report...")
			if err := c.BuildDatastoreConfig(); err != nil {
				return err
			}
			g, err := datastore.NewGithub(c)
			if err != nil {
				return err
			}
			ctx := context.Background()
			if err := g.Store(ctx, r); err != nil {
				return err
			}
		}

		// Check for acceptable coverage
		if err := c.Accepptable(r); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "", "", "config file path")
	rootCmd.Flags().BoolVarP(&dump, "dump", "", false, "dump coverage report")
	rootCmd.Flags().BoolVarP(&genbadge, "badge", "", false, "generate coverage report badge")
}

func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	log.SetOutput(ioutil.Discard)
	if env := os.Getenv("DEBUG"); env != "" {
		debug, err := os.Create(fmt.Sprintf("%s.debug", version.Name))
		if err != nil {
			rootCmd.PrintErrln(err)
			os.Exit(1)
		}
		log.SetOutput(debug)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
