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
	"strings"
	"time"

	"github.com/k1LoW/octocov/central"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/k1LoW/octocov/report"
	"github.com/k1LoW/octocov/version"
	"github.com/spf13/cobra"
)

var (
	configPath    string
	dump          bool
	coverageBadge bool
	ratioBadge    bool
	timeBadge     bool
)

var rootCmd = &cobra.Command{
	Use:          "octocov",
	Short:        "octocov is a tool for collecting code metrics",
	Long:         `octocov is a tool for collecting code metrics.`,
	Version:      version.Version,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.PrintErrf("%s version %s\n", version.Name, version.Version)
		if len(args) > 0 && args[0] == "completion" {
			return completionCmd(cmd, args[1:])
		}
		c := config.New()
		if err := c.Load(configPath); err != nil {
			return err
		}
		c.Build()

		if c.CentralConfigReady() {
			cmd.PrintErrln("Central mode enabled")
			if err := c.BuildCentralConfig(); err != nil {
				return err
			}
			ctr := central.New(c)
			return ctr.Generate()
		}

		path := c.Coverage.Path
		r := report.New()
		if err := r.MeasureCoverage(path); err != nil {
			return err
		}
		if c.CodeToTestRatioReady() {
			if err := r.MeasureCodeToTestRatio(c.CodeToTestRatio.Code, c.CodeToTestRatio.Test); err != nil {
				return err
			}
		}
		if err := r.MeasureTestExecutionTime(); err != nil {
			return err
		}

		if dump {
			cmd.Println(r.String())
			return nil
		}

		if !c.Loaded() {
			return fmt.Errorf("%s are not found", strings.Join(config.DefaultConfigFilePaths, " and "))
		}

		// Generate coverage report badge
		if c.CoverageBadgeConfigReady() || coverageBadge {
			var out *os.File
			cp := r.CoveragePercent()
			if c.Coverage.Badge.Path == "" {
				out = os.Stdout
			} else {
				cmd.PrintErrln("Generate coverage report badge...")
				err := os.MkdirAll(filepath.Dir(c.Coverage.Badge.Path), 0755) // #nosec
				if err != nil {
					return err
				}
				out, err = os.OpenFile(filepath.Clean(c.Coverage.Badge.Path), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
				if err != nil {
					return err
				}
			}

			b := badge.New("coverage", fmt.Sprintf("%.1f%%", cp))
			b.MessageColor = c.CoverageColor(cp)
			if err := b.Render(out); err != nil {
				return err
			}

			if coverageBadge {
				return nil
			}
		}

		// Generate code-to-test-ratio report badge
		if c.CodeToTestRatioBadgeConfigReady() || ratioBadge {
			var out *os.File
			tr := r.CodeToTestRatioRatio()
			if c.CodeToTestRatio.Badge.Path == "" {
				out = os.Stdout
			} else {
				cmd.PrintErrln("Generate code-to-test-ratio report badge...")
				err := os.MkdirAll(filepath.Dir(c.CodeToTestRatio.Badge.Path), 0755) // #nosec
				if err != nil {
					return err
				}
				out, err = os.OpenFile(filepath.Clean(c.CodeToTestRatio.Badge.Path), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
				if err != nil {
					return err
				}
			}

			b := badge.New("code to test ratio", fmt.Sprintf("1:%.1f", tr))
			b.MessageColor = c.CodeToTestRatioColor(tr)
			if err := b.Render(out); err != nil {
				return err
			}

			if ratioBadge {
				return nil
			}
		}

		// Generate test-execution-time report badge
		if c.TestExecutionTimeBadgeConfigReady() || timeBadge {
			var out *os.File
			if r.TestExecutionTime == nil {
				cmd.PrintErrln("Skip generating test-execution-time badge: in order to generate the test-execution-time badge, it is necessary to measure the code coverage on GitHub Actions.")
			} else {
				if c.TestExecutionTime.Badge.Path == "" {
					out = os.Stdout
				} else {
					cmd.PrintErrln("Generate test-execution-time report badge...")
					err := os.MkdirAll(filepath.Dir(c.TestExecutionTime.Badge.Path), 0755) // #nosec
					if err != nil {
						return err
					}
					out, err = os.OpenFile(filepath.Clean(c.TestExecutionTime.Badge.Path), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
					if err != nil {
						return err
					}
				}

				d := time.Duration(*r.TestExecutionTime)
				b := badge.New("test execution time", d.String())
				b.MessageColor = c.TestExecutionTimeColor(d)
				if err := b.Render(out); err != nil {
					return err
				}
			}
			if timeBadge {
				return nil
			}
		}

		// Store report
		if c.DatastoreConfigReady() {
			cmd.PrintErrln("Store coverage report...")
			if err := r.Validate(); err != nil {
				return err
			}
			if err := c.BuildDatastoreConfig(); err != nil {
				return err
			}
			gh, err := gh.New()
			if err != nil {
				return err
			}
			g, err := datastore.NewGithub(c, gh)
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
	rootCmd.Flags().BoolVarP(&coverageBadge, "coverage-badge", "", false, "generate coverage report badge")
	rootCmd.Flags().BoolVarP(&ratioBadge, "code-to-test-ratio-badge", "", false, "generate code-to-test-ratio report badge")
	rootCmd.Flags().BoolVarP(&timeBadge, "test-execution-time-badge", "", false, "generate test-execution-time report badge")
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
