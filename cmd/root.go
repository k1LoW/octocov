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
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
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
		ctx := context.Background()
		addPaths := []string{}
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

			fsys, err := c.CentralReportsFS()
			if err != nil {
				return err
			}

			ctr := central.New(&central.CentralConfig{
				Repository:             c.Repository,
				Index:                  c.Central.Root,
				Wd:                     c.Getwd(),
				Badges:                 c.Central.Badges,
				Reports:                fsys,
				CoverageColor:          c.CoverageColor,
				CodeToTestRatioColor:   c.CodeToTestRatioColor,
				TestExecutionTimeColor: c.TestExecutionTimeColor,
			})
			paths, err := ctr.Generate(ctx)
			if err != nil {
				return err
			}
			// git push
			if c.CentralPushConfigReady() {
				_, _ = fmt.Fprintln(os.Stderr, "Commit and push central report")
				if err := gh.PushUsingLocalGit(ctx, c.GitRoot, paths, "Update by octocov"); err != nil {
					return err
				}
			}
			return nil
		}

		r := report.New()

		if c.CoverageConfigReady() {
			path := c.Coverage.Path
			if err := r.MeasureCoverage(path); err != nil {
				cmd.PrintErrf("Skip measuring code coverage: %v\n", err)
			}
		}

		if c.CodeToTestRatioConfigReady() {
			if err := r.MeasureCodeToTestRatio(c.CodeToTestRatio.Code, c.CodeToTestRatio.Test); err != nil {
				cmd.PrintErrf("Skip measuring code to test ratio: %v\n", err)
			}
		}

		if c.TestExecutionTimeConfigReady() {
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

		if dump {
			if err := r.Validate(); err != nil {
				cmd.PrintErrf("Validation error: %v\n", err)
			}
			cmd.Println(r.String())
			return nil
		}

		if !c.Loaded() {
			return fmt.Errorf("%s are not found", strings.Join(config.DefaultConfigFilePaths, " and "))
		}

		// Generate coverage report badge
		if c.CoverageBadgeConfigReady() || coverageBadge {
			if !r.IsMeasuredCoverage() {
				return fmt.Errorf("could not generate badge: %s", "coverage is not measured")
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
			if err := b.Render(out); err != nil {
				return err
			}

			if coverageBadge {
				return nil
			}
		}

		// Generate code-to-test-ratio report badge
		if c.CodeToTestRatioBadgeConfigReady() || ratioBadge {
			if !r.IsMeasuredCodeToTestRatio() {
				return fmt.Errorf("could not generate badge: %s", "code-to-test-ratio is not measured")
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
			if err := b.Render(out); err != nil {
				return err
			}

			if ratioBadge {
				return nil
			}
		}

		// Generate test-execution-time report badge
		if c.TestExecutionTimeBadgeConfigReady() || timeBadge {
			if !r.IsMeasuredTestExecutionTime() {
				return fmt.Errorf("could not generate badge: %s", "test-execution-time is not measured")
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

			d := time.Duration(*r.TestExecutionTime)
			b := badge.New("test execution time", d.String())
			b.MessageColor = c.TestExecutionTimeColor(d)
			if err := b.Render(out); err != nil {
				return err
			}
			if timeBadge {
				return nil
			}
		}

		// Store report
		if c.DatastoreConfigReady() {
			cmd.PrintErrln("Store report...")
			if err := c.BuildDatastoreConfig(); err != nil {
				return err
			}
			if c.Datastore.Github != nil {
				// GitHub
				gh, err := gh.New()
				if err != nil {
					return err
				}
				g, err := datastore.NewGithub(gh, c.Datastore.Github.Repository, c.Datastore.Github.Branch)
				if err != nil {
					return err
				}
				if err := g.Store(ctx, c.Datastore.Github.Path, r); err != nil {
					return err
				}
			}
			if c.Datastore.S3 != nil {
				// S3
				sess, err := session.NewSession()
				if err != nil {
					return err
				}
				sc := s3.New(sess)
				s, err := datastore.NewS3(sc, c.Datastore.S3.Bucket)
				if err != nil {
					return err
				}
				if err := s.Store(ctx, c.Datastore.S3.Path, r); err != nil {
					return err
				}
			}
		}

		// Comment report to pull request
		if c.CommentConfigReady() {
			cmd.PrintErrln("Comment report...")
			owner, repo, err := c.OwnerRepo()
			if err != nil {
				return err
			}
			gh, err := gh.New()
			if err != nil {
				return err
			}
			n, err := gh.DetectCurrentPullRequestNumber(ctx, owner, repo)
			if err != nil {
				cmd.PrintErrf("Skip commenting the report to pull request: %v\n", err)
			} else {
				footer := "Reported by [octocov](https://github.com/k1LoW/octocov)"
				if c.Comment.HideFooterLink {
					footer = "Reported by octocov"
				}
				comment := strings.Join([]string{
					r.Table(),
					"---",
					footer,
				}, "\n")
				if err := gh.PutComment(ctx, owner, repo, n, comment); err != nil {
					return err
				}
			}
		}

		// Push generated files
		if c.PushConfigReady() {
			if err := c.BuildPushConfig(); err != nil {
				return err
			}
			if err := gh.PushUsingLocalGit(ctx, c.GitRoot, addPaths, "Update by octocov"); err != nil {
				return err
			}
		}

		// Check for acceptable coverage
		if err := c.Acceptable(r); err != nil {
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

	log.SetOutput(io.Discard)
	if env := os.Getenv("DEBUG"); env != "" {
		log.SetOutput(os.Stderr)
	}

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
