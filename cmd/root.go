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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/k1LoW/octocov/central"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/internal"
	"github.com/k1LoW/octocov/pkg/badge"
	"github.com/k1LoW/octocov/report"
	"github.com/k1LoW/octocov/version"
	"github.com/spf13/cobra"
)

var (
	configPath  string
	createTable bool
)

var rootCmd = &cobra.Command{
	Use:          "octocov",
	Short:        "octocov is a toolkit for collecting code metrics",
	Long:         `octocov is a toolkit for collecting code metrics.`,
	Version:      version.Version,
	Args:         cobra.NoArgs,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if os.Getenv("CI") == "" {
			return printMetrics(cmd)
		}

		ctx := context.Background()
		addPaths := []string{}
		cmd.PrintErrf("%s version %s\n", version.Name, version.Version)

		c := config.New()
		if err := c.Load(configPath); err != nil {
			return err
		}
		c.Build()

		if !c.Loaded() {
			cmd.PrintErrf("%s are not found\n", strings.Join(config.DefaultConfigFilePaths, " and "))
		}

		if c.Central != nil && internal.IsEnable(c.Central.Enable) {
			cmd.PrintErrln("Central mode enabled")
			if err := c.CentralConfigReady(); err != nil {
				return err
			}

			badges := []datastore.Datastore{}
			for _, s := range c.Central.Badges.Datastores {
				d, err := datastore.New(ctx, s, c.Root())
				if err != nil {
					return err
				}
				badges = append(badges, d)
			}

			reports := []datastore.Datastore{}
			for _, s := range c.Central.Reports.Datastores {
				d, err := datastore.New(ctx, s, c.Root())
				if err != nil {
					return err
				}
				reports = append(reports, d)
			}

			ctr := central.New(&central.CentralConfig{
				Repository:             c.Repository,
				Index:                  c.Central.Root,
				Wd:                     c.Getwd(),
				Badges:                 badges,
				Reports:                reports,
				CoverageColor:          c.CoverageColor,
				CodeToTestRatioColor:   c.CodeToTestRatioColor,
				TestExecutionTimeColor: c.TestExecutionTimeColor,
			})
			paths, err := ctr.Generate(ctx)
			if err != nil {
				return err
			}
			// git push
			if err := c.CentralPushConfigReady(); err != nil {
				cmd.PrintErrf("Skip commit and push central report: %v\n", err)
			} else {
				cmd.PrintErrln("Commit and push central report")
				if err := gh.PushUsingLocalGit(ctx, c.GitRoot, paths, "Update by octocov"); err != nil {
					return err
				}
			}
			return nil
		}

		r, err := report.New(c.Repository)
		if err != nil {
			return err
		}

		if err := c.CoverageConfigReady(); err != nil {
			cmd.PrintErrf("Skip measuring code coverage: %v\n", err)
		} else {
			if err := r.MeasureCoverage(c.Coverage.Paths); err != nil {
				cmd.PrintErrf("Skip measuring code coverage: %v\n", err)
			}
		}

		if err := c.CodeToTestRatioConfigReady(); err != nil {
			cmd.PrintErrf("Skip measuring code to test ratio: %v\n", err)
		} else {
			if err := r.MeasureCodeToTestRatio(c.Root(), c.CodeToTestRatio.Code, c.CodeToTestRatio.Test); err != nil {
				cmd.PrintErrf("Skip measuring code to test ratio: %v\n", err)
			}
		}

		if err := c.TestExecutionTimeConfigReady(); err != nil {
			cmd.PrintErrf("Skip measuring test execution time: %v\n", err)
		} else {
			stepNames := []string{}
			if len(c.TestExecutionTime.Steps) > 0 {
				stepNames = c.TestExecutionTime.Steps
			}
			if err := r.MeasureTestExecutionTime(ctx, stepNames); err != nil {
				cmd.PrintErrf("Skip measuring test execution time: %v\n", err)
			}
		}

		if r.CountMeasured() == 0 {
			return errors.New("nothing could be measured")
		}

		cmd.Println("")
		if err := r.Out(os.Stdout); err != nil {
			return err
		}
		cmd.Println("")

		// Generate coverage report badge
		if err := c.CoverageBadgeConfigReady(); err == nil {
			if err := func() error {
				if !r.IsMeasuredCoverage() {
					cmd.PrintErrf("Skip generating badge: %s\n", "coverage is not measured")
					return nil
				}
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
					bp, err := filepath.Abs(filepath.Clean(c.Coverage.Badge.Path))
					if err != nil {
						return err
					}
					out, err = os.OpenFile(bp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
					if err != nil {
						return err
					}
					addPaths = append(addPaths, bp)
				}

				b := badge.New("coverage", fmt.Sprintf("%.1f%%", cp))
				b.MessageColor = c.CoverageColor(cp)
				if err := b.AddIcon(internal.Icon); err != nil {
					return err
				}
				if err := b.Render(out); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return err
			}
		}

		// Generate code-to-test-ratio report badge
		if err := c.CodeToTestRatioBadgeConfigReady(); err == nil {
			if err := func() error {
				if !r.IsMeasuredCodeToTestRatio() {
					cmd.PrintErrf("Skip generating badge: %s\n", "coverage is not measured")
					return nil
				}

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
					bp, err := filepath.Abs(filepath.Clean(c.CodeToTestRatio.Badge.Path))
					if err != nil {
						return err
					}
					out, err = os.OpenFile(bp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
					if err != nil {
						return err
					}
					addPaths = append(addPaths, bp)
				}

				b := badge.New("code to test ratio", fmt.Sprintf("1:%.1f", tr))
				b.MessageColor = c.CodeToTestRatioColor(tr)
				if err := b.AddIcon(internal.Icon); err != nil {
					return err
				}
				if err := b.Render(out); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return err
			}
		}

		// Generate test-execution-time report badge
		if err := c.TestExecutionTimeBadgeConfigReady(); err == nil {
			if err := func() error {
				if !r.IsMeasuredTestExecutionTime() {
					cmd.PrintErrf("Skip generating badge: %s\n", "test-execution-time is not measured")
					return nil
				}

				var out *os.File
				if c.TestExecutionTime.Badge.Path == "" {
					out = os.Stdout
				} else {
					cmd.PrintErrln("Generate test-execution-time report badge...")
					err := os.MkdirAll(filepath.Dir(c.TestExecutionTime.Badge.Path), 0755) // #nosec
					if err != nil {
						return err
					}
					bp, err := filepath.Abs(filepath.Clean(c.TestExecutionTime.Badge.Path))
					if err != nil {
						return err
					}
					out, err = os.OpenFile(bp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644) // #nosec
					if err != nil {
						return err
					}
					addPaths = append(addPaths, bp)
				}

				d := time.Duration(r.TestExecutionTimeNano())
				b := badge.New("test execution time", d.String())
				b.MessageColor = c.TestExecutionTimeColor(d)
				if err := b.AddIcon(internal.Icon); err != nil {
					return err
				}
				if err := b.Render(out); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				return err
			}
		}

		// Get previous report for comparing reports
		var rPrev *report.Report
		if err := c.DiffConfigReady(); err == nil {
			repo, err := gh.Parse(c.Repository)
			if err != nil {
				return err
			}
			path := fmt.Sprintf("%s/%s/report.json", repo.Owner, repo.Reponame())
			for _, s := range c.Diff.Datastores {
				d, err := datastore.New(ctx, s, c.Root())
				if err != nil {
					return err
				}
				fsys, err := d.FS()
				if err != nil {
					return err
				}
				f, err := fsys.Open(path)
				if err != nil {
					continue
				}
				defer f.Close()
				b, err := io.ReadAll(f)
				if err != nil {
					continue
				}
				rt := &report.Report{}
				if err := json.Unmarshal(b, rt); err != nil {
					continue
				}
				if rPrev == nil || rPrev.Timestamp.UnixNano() < rt.Timestamp.UnixNano() {
					rPrev = rt
				}
			}
			if c.Diff.Path != "" {
				rt, err := report.New(c.Repository)
				if err != nil {
					return err
				}
				if err := rt.MeasureCoverage([]string{c.Diff.Path}); err == nil {
					if rPrev == nil || rPrev.Timestamp.UnixNano() < rt.Timestamp.UnixNano() {
						rPrev = rt
					}
				}
			}
		}

		// Comment report to pull request
		if err := c.CommentConfigReady(); err != nil {
			cmd.PrintErrf("Skip commenting report to pull request: %v\n", err)
		} else {
			if err := func() error {
				cmd.PrintErrln("Commenting report...")
				if err := c.DiffConfigReady(); rPrev == nil || err != nil {
					cmd.PrintErrf("Skip comparing reports: %v\n", err)
				}
				if err := commentReport(ctx, c, r, rPrev); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				cmd.PrintErrf("Skip commenting the report to pull request: %v\n", err)
			}
		}

		// Store report
		if err := c.ReportConfigReady(); err != nil {
			cmd.PrintErrf("Skip storing the report: %v\n", err)
		} else {
			cmd.PrintErrln("Storing report...")
			if c.Report.Path != "" {
				rp, err := filepath.Abs(filepath.Clean(c.Report.Path))
				if err != nil {
					return err
				}
				if err := os.WriteFile(rp, r.Bytes(), os.ModePerm); err != nil {
					return err
				}
				addPaths = append(addPaths, rp)
			}
			if r.Coverage != nil {
				r.Coverage.DeleteBlockCoverages()
			}
			if r.CodeToTestRatio != nil {
				r.CodeToTestRatio.DeleteFiles()
			}
			datastores := []datastore.Datastore{}
			for _, s := range c.Report.Datastores {
				d, err := datastore.New(ctx, s, c.Root())
				if err != nil {
					return err
				}
				datastores = append(datastores, d)
			}
			for _, d := range datastores {
				if err := d.StoreReport(ctx, r); err != nil {
					return err
				}
			}
		}

		// Push generated files
		if err := c.PushConfigReady(); err != nil {
			cmd.PrintErrf("Skip pushing generate files: %v\n", err)
		} else {
			cmd.PrintErrln("Pushing generated files...")
			if err := gh.PushUsingLocalGit(ctx, c.GitRoot, addPaths, "Update by octocov"); err != nil {
				return err
			}
		}

		// Check for acceptable code metrics
		if err := c.Acceptable(r, rPrev); err != nil {
			return err
		}

		return nil
	},
}

func printMetrics(cmd *cobra.Command) error {
	c := config.New()
	if err := c.Load(configPath); err != nil {
		return err
	}
	c.Build()

	r, err := report.New(c.Repository)
	if err != nil {
		return err
	}

	if err := c.CoverageConfigReady(); err == nil {
		if err := r.MeasureCoverage(c.Coverage.Paths); err != nil {
			cmd.PrintErrf("Skip measuring code coverage: %v\n", err)
		}
	}

	if err := c.CodeToTestRatioConfigReady(); err == nil {
		if err := r.MeasureCodeToTestRatio(c.Root(), c.CodeToTestRatio.Code, c.CodeToTestRatio.Test); err != nil {
			cmd.PrintErrf("Skip measuring code to test ratio: %v\n", err)
		}
	}

	if r.CountMeasured() == 0 {
		return errors.New("nothing could be measured")
	}

	cmd.Println("")
	if err := r.Out(os.Stdout); err != nil {
		return err
	}
	cmd.Println("")

	return nil
}

func init() {
	rootCmd.Flags().StringVarP(&configPath, "config", "", "", "config file path")
	rootCmd.Flags().BoolVarP(&createTable, "create-bq-table", "", false, "create table of BigQuery dataset")
}

func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	log.SetOutput(io.Discard)
	if env := os.Getenv("DEBUG"); env != "" {
		log.SetOutput(os.Stderr)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
