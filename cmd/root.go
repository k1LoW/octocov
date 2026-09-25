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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/k1LoW/octocov/central"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/internal"
	"github.com/k1LoW/octocov/report"
	"github.com/k1LoW/octocov/version"
	"github.com/spf13/cobra"
)

const defaultCommitMessage = "Update by octocov"

var (
	configPath  string
	reportPath  string
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

		var addPaths []string
		cmd.PrintErrf("%s version %s\n", version.Name, version.Version)

		c := config.New()
		if err := c.Load(configPath); err != nil {
			return err
		}
		c.Build()

		if !c.Loaded() {
			cmd.PrintErrf("%s are not found\n", strings.Join(config.Paths, " and "))
		}

		ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
		defer cancel()

		if reportPath != "" {
			c.Coverage.Paths = []string{reportPath}
			c.CodeToTestRatio = nil
			c.TestExecutionTime = nil
		}

		if c.Central != nil {
			cmd.PrintErrln("Central mode enabled")
			if err := c.CentralConfigReady(); err != nil {
				return err
			}

			var badges []datastore.Datastore
			for _, s := range c.Central.Badges.Datastores {
				d, err := datastore.New(ctx, s, datastore.Root(c.Root()))
				if err != nil {
					return err
				}
				badges = append(badges, d)
			}

			var reports []central.ReportDatastore
			for _, s := range c.Central.Reports.Datastores {
				d, err := datastore.New(ctx, s, datastore.Root(c.Root()))
				if err != nil {
					return err
				}
				reports = append(reports, central.ReportDatastore{URL: s, Datastore: d})
			}

			ctr := central.New(&central.Config{
				Repository:             c.Repository,
				Index:                  c.Central.Root,
				Wd:                     c.Wd(),
				Badges:                 badges,
				Reports:                reports,
				CoverageBadge:          c.Central.Badges.Coverage.RenderCoverage,
				CodeToTestRatioBadge:   c.Central.Badges.CodeToTestRatio.RenderCodeToTestRatio,
				TestExecutionTimeBadge: c.Central.Badges.TestExecutionTime.RenderTestExecutionTime,
				BadgeViewer:            resolveBadgeViewer(ctx, cmd.ErrOrStderr(), c),
			})

			paths, err := ctr.Generate(ctx)
			if err != nil {
				return err
			}

			// re report
			if err := c.CentralReReportReady(); err != nil {
				cmd.PrintErrf("Skip re storing report: %v\n", err)
			} else {
				cmd.PrintErrln("Storing re-report...")
				for _, r := range ctr.CollectedReports() {
					if err := reportToDatastores(ctx, c, c.Central.ReReport.Datastores, r); err != nil {
						return err
					}
				}
				// TODO: Get the path to the report stored in local:// and target it for git push
			}

			// git push
			if err := c.CentralPushConfigReady(); err != nil {
				cmd.PrintErrf("Skip commit and push central report: %v\n", err)
			} else {
				cmd.PrintErrln("Commit and push central report")

				m := defaultCommitMessage
				if c.Central.Push.Message != "" {
					m = c.Central.Push.Message
				}

				c, err := gh.PushUsingLocalGit(ctx, c.GitRoot, paths, m)
				if err != nil {
					return err
				}
				if c == 0 {
					cmd.PrintErrln("No files to be commit")
				}
			}
			return nil
		}

		r, err := report.New(c.Repository, report.Locale(c.Locale))
		if err != nil {
			return err
		}

		// Not fatal, since without it the report is stored where the reports of the
		// default branch go, which is where every report went before refs were separated.
		if err := r.DetectRef(ctx); err != nil {
			cmd.PrintErrf("Skip detecting the ref of the report: %v\n", err)
		}

		if err := c.CoverageConfigReady(); err != nil {
			cmd.PrintErrf("Skip measuring code coverage: %v\n", err)
		} else {
			if err := r.MeasureCoverage(c.Coverage.Paths, c.Coverage.Exclude); err != nil {
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
			var stepNames []string
			if len(c.TestExecutionTime.Steps) > 0 {
				stepNames = c.TestExecutionTime.Steps
			}
			if err := r.MeasureTestExecutionTime(ctx, stepNames); err != nil {
				cmd.PrintErrf("Skip measuring test execution time: %v\n", err)
			}
		}

		if err := r.CollectCustomMetrics(); err != nil {
			cmd.PrintErrf("Skip collecting custom metrics: %v\n", err)
		}

		if r.CountMeasured() == 0 {
			return errors.New("nothing could be measured")
		}

		if err := r.Validate(); err != nil {
			return fmt.Errorf("validation error: %w", err)
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
				cp := r.CoveragePercent()
				cmd.PrintErrln("Generate coverage report badge...")
				bp, err := filepath.Abs(filepath.Clean(c.Coverage.Badge.Path))
				if err != nil {
					return err
				}
				addPaths = append(addPaths, bp)

				return badgeFile(c.Coverage.Badge.Path, func(out io.Writer) error {
					return c.Coverage.Badge.RenderCoverage(out, cp)
				})
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

				tr := r.CodeToTestRatioRatio()
				cmd.PrintErrln("Generate code-to-test-ratio report badge...")
				bp, err := filepath.Abs(filepath.Clean(c.CodeToTestRatio.Badge.Path))
				if err != nil {
					return err
				}
				addPaths = append(addPaths, bp)

				return badgeFile(c.CodeToTestRatio.Badge.Path, func(out io.Writer) error {
					return c.CodeToTestRatio.Badge.RenderCodeToTestRatio(out, tr)
				})
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

				cmd.PrintErrln("Generate test-execution-time report badge...")
				bp, err := filepath.Abs(filepath.Clean(c.TestExecutionTime.Badge.Path))
				if err != nil {
					return err
				}
				addPaths = append(addPaths, bp)

				return badgeFile(c.TestExecutionTime.Badge.Path, func(out io.Writer) error {
					return c.TestExecutionTime.Badge.RenderTestExecutionTime(out, r.TestExecutionTimeNano())
				})
			}(); err != nil {
				return err
			}
		}

		// Get previous report for comparing reports
		var rPrev *report.Report
		// comparedArtifact names the artifact holding the metadata rPrev was read through, and
		// stays empty when it was read from anywhere but an artifact or without metadata. The
		// pages find a report by its metadata, so a comparison no metadata points at has no
		// page describing it to link at.
		var comparedArtifact string
		if err := c.DiffConfigReady(); err == nil {
			log.Println("Get previous report for comparing reports")
			repo, err := gh.Parse(c.Repository)
			if err != nil {
				return err
			}

			// Collect filesystem files once for normalizing all loaded reports
			var gitRoot string
			var fsFiles []string
			if wd, err := os.Getwd(); err == nil {
				if gr, err := internal.GitRoot(wd); err == nil {
					if fs, err := internal.CollectFiles(gr); err == nil {
						gitRoot = gr
						fsFiles = fs
					}
				}
			}

			var stores []comparedDatastore
			for _, s := range c.Diff.Datastores {
				log.Printf("Get previous report from %s", s)
				d, err := datastore.New(ctx, s, datastore.Root(c.Root()), datastore.Report(r))
				if err != nil {
					return err
				}
				fsys, err := d.FS()
				if err != nil {
					// The previous report simply may not be there yet, which the artifact
					// datastore now says rather than answering with an empty filesystem, so
					// this is the same kind of miss the reads below already carry on from.
					log.Printf("%s: %v", s, err)
					continue
				}
				stores = append(stores, comparedDatastore{name: s, fsys: fsys, metadataRead: datastore.MetadataRead(d)})
			}
			rPrev, comparedArtifact = readBaseReport(stores, fmt.Sprintf("%s/%s", repo.Owner, repo.Reponame()), r.BaseKeys())
			if rPrev != nil && rPrev.Coverage != nil {
				rPrev.Coverage.NormalizePaths(gitRoot, fsFiles)
			}
			if c.Diff.Path != "" {
				rt, err := report.New(c.Repository, report.Locale(c.Locale))
				if err != nil {
					return err
				}
				if err := rt.MeasureCoverage([]string{c.Diff.Path}, c.Coverage.Exclude); err == nil {
					if rPrev == nil || rPrev.Timestamp.UnixNano() < rt.Timestamp.UnixNano() {
						rPrev = rt
						// Measured here rather than read out of an artifact, and it wins on
						// timestamp whenever both are configured.
						comparedArtifact = ""
					}
				}
			}
		}

		// The three pull request outputs and the patch coverage measurement all read the same
		// changed file list, and each fetch is paginated over up to 3000 files. Fetch it at
		// most once, and only if one of them actually asks for it.
		pullRequestDiff := sync.OnceValues(func() (*gh.PullRequestFiles, error) {
			return fetchPullRequestDiff(ctx, cmd, c.Repository, r.Commit)
		})
		pullRequestFiles := func() ([]*gh.PullRequestFile, error) {
			d, err := pullRequestDiff()
			if err != nil {
				return nil, err
			}
			return d.Files, nil
		}

		// Each readiness check reaches the API, and whether any output is going to be written
		// decides whether there is anything to link from, so each is checked once up front.
		commentReady := c.CommentConfigReady()
		summaryReady := c.SummaryConfigReady()
		bodyReady := c.BodyConfigReady()
		var (
			cur, prev *report.Viewer
			cleanup   func()
		)
		if commentReady == nil || summaryReady == nil || bodyReady == nil {
			cur, prev, cleanup = resolveViewers(ctx, cmd.ErrOrStderr(), c, r, rPrev, comparedArtifact, pullRequestDiff)
		}
		// Whether every output that was attempted got written, which is part of what the pages
		// earlier runs uploaded may be deleted on. An output left as it was still links to one
		// of them.
		written := true
		viewerErr := func() error {
			return errors.Join(cur.Err(), prev.Err())
		}

		// Comment report to pull request
		if err := commentReady; err != nil {
			cmd.PrintErrf("Skip commenting report to pull request: %v\n", err)
		} else {
			if err := func() error {
				cmd.PrintErrln("Commenting report...")
				if rPrev == nil {
					cmd.PrintErrln("Skip comparing reports: previous report not found")
				}
				if err := c.DiffConfigReady(); err != nil {
					cmd.PrintErrf("Skip comparing reports: %v\n", err)
				}
				files, err := pullRequestFiles()
				if err != nil {
					return err
				}
				content, err := createReportContent(c, r, rPrev, files, c.Comment.Message, c.Comment.HideFooterLink, c.Comment.ExpandDetails, cur, prev)
				if err != nil {
					return err
				}
				if err := viewerErr(); err != nil {
					return err
				}
				if err := commentReport(ctx, c, content, r.Key()); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				written = false
				cmd.PrintErrf("Skip commenting report to pull request: %v\n", err)
			}
		}

		// Add report to job summary page
		if err := summaryReady; err != nil {
			cmd.PrintErrf("Skip adding report to job summary page: %v\n", err)
		} else {
			if err := func() error {
				cmd.PrintErrln("Adding report to job summary page...")
				if rPrev == nil {
					cmd.PrintErrln("Skip comparing reports: previous report not found")
				}
				if err := c.DiffConfigReady(); err != nil {
					cmd.PrintErrf("Skip comparing reports: %v\n", err)
				}
				files, err := pullRequestFiles()
				if err != nil {
					return err
				}
				content, err := createReportContent(c, r, rPrev, files, c.Summary.Message, c.Summary.HideFooterLink, c.Summary.ExpandDetails, cur, prev)
				if err != nil {
					return err
				}
				if err := viewerErr(); err != nil {
					return err
				}
				if err := addReportContentToSummary(content); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				written = false
				cmd.PrintErrf("Skip adding report to job summary page: %v\n", err)
			}
		}

		// Insert report to body of pull request
		if err := bodyReady; err != nil {
			cmd.PrintErrf("Skip inserting report to body of pull request: %v\n", err)
		} else {
			if err := func() error {
				cmd.PrintErrln("Inserting report...")
				if rPrev == nil {
					cmd.PrintErrln("Skip comparing reports: previous report not found")
				}
				if err := c.DiffConfigReady(); err != nil {
					cmd.PrintErrf("Skip comparing reports: %v\n", err)
				}
				files, err := pullRequestFiles()
				if err != nil {
					return err
				}
				content, err := createReportContent(c, r, rPrev, files, c.Body.Message, c.Body.HideFooterLink, c.Body.ExpandDetails, cur, prev)
				if err != nil {
					return err
				}
				if err := viewerErr(); err != nil {
					return err
				}
				if err := replaceInsertReportToBody(ctx, c, content, r.Key()); err != nil {
					return err
				}
				return nil
			}(); err != nil {
				written = false
				cmd.PrintErrf("Skip inserting report to body of pull request: %v\n", err)
			}
		}

		if cleanup != nil && mayDeleteEarlierPages(c, commentReady, bodyReady, written, func() bool {
			return earlierOutputsLeft(ctx, c, r)
		}) {
			cleanup()
		}

		// Measure patch coverage before storing the report, because storing the report shrinks
		// away the block coverages that patch coverage is derived from.
		var patchCoverage *coverage.PatchCoverage
		if c.Coverage.AcceptableReferencesPatch() {
			if err := c.CoverageConfigReady(); err != nil {
				cmd.PrintErrf("Skip measuring patch coverage: %v\n", err)
			} else {
				files, err := pullRequestFiles()
				changedFiles := gh.ChangedLinesByFile(files)
				switch {
				case err != nil:
					cmd.PrintErrf("Skip measuring patch coverage: %v\n", err)
				case len(files) == 0:
					cmd.PrintErrln("Skip measuring patch coverage: no changed files were fetched")
				case len(changedFiles) == 0:
					cmd.PrintErrln("Skip measuring patch coverage: no changed lines were fetched")
				default:
					patchCoverage = r.PatchCoverage(changedFiles)
					if patchCoverage.Total == 0 {
						cmd.PrintErrln("Skip measuring patch coverage: no changed line is instrumented by the coverage report")
					}
				}
			}
		}

		// Store report
		if err := c.ReportConfigReady(); err != nil {
			cmd.PrintErrf("Skip storing report: %v\n", err)
		} else {
			cmd.PrintErrln("Storing report...")
			if c.Report.Path != "" {
				rp, err := filepath.Abs(filepath.Clean(c.Report.Path))
				if err != nil {
					return err
				}
				if err := os.WriteFile(rp, r.Bytes(), os.ModePerm); err != nil { //nolint:gosec
					return err
				}
				addPaths = append(addPaths, rp)
			}
			if err := reportToDatastores(ctx, c, c.Report.Datastores, r); err != nil {
				return err
			}
		}

		// Push generated files
		if err := c.PushConfigReady(); err != nil {
			cmd.PrintErrf("Skip pushing generate files: %v\n", err)
		} else {
			cmd.PrintErrln("Pushing generated files...")

			m := defaultCommitMessage
			if c.Push.Message != "" {
				m = c.Push.Message
			}

			c, err := gh.PushUsingLocalGit(ctx, c.GitRoot, addPaths, m)
			if err != nil {
				return err
			}
			if c == 0 {
				cmd.PrintErrln("No files to be commit")
			}
		}

		// Check for acceptable code metrics
		if err := c.Acceptable(r, rPrev, patchCoverage); err != nil {
			return err
		}

		return nil
	},
}

// fetchPullRequestFiles returns the changed files of the current pull request, or, when the run
// is not against a pull request, the files changed since the default branch. Both the patch
// coverage column of the file coverage tables and the `patch` acceptable variable are measured
// over the changed lines it carries, so those are numbered as the lines of commit, the commit
// the coverage was measured on.
func fetchPullRequestFiles(ctx context.Context, cmd *cobra.Command, repository, commit string) ([]*gh.PullRequestFile, error) {
	d, err := fetchPullRequestDiff(ctx, cmd, repository, commit)
	if err != nil {
		return nil, err
	}
	return d.Files, nil
}

// fetchPullRequestDiff is fetchPullRequestFiles with what the page it is drawn on needs to know
// besides the files, which is which commits the two sides of the patches are numbered as.
func fetchPullRequestDiff(ctx context.Context, cmd *cobra.Command, repository, commit string) (*gh.PullRequestFiles, error) {
	repo, err := gh.Parse(repository)
	if err != nil {
		return nil, err
	}
	g, err := gh.New()
	if err != nil {
		return nil, err
	}
	n, err := g.DetectCurrentPullRequestNumber(ctx, repo.Owner, repo.Repo)
	if err != nil {
		if !errors.Is(err, gh.ErrNotPullRequest) {
			// A token without access to the pull request, a rate limit, or an ambiguous head
			// fail here just like a plain push does, and the default branch comparison then
			// measures a different set of changed lines. Say so, but stay quiet for a run that
			// simply is not a pull request, which is the ordinary way to reach this.
			cmd.PrintErrf("Could not look up the current pull request, comparing against the default branch instead: %v\n", err)
		}
		files, err := g.FetchChangedFiles(ctx, repo.Owner, repo.Repo)
		if err != nil {
			return nil, err
		}
		return &gh.PullRequestFiles{Files: files, Unaligned: "the changed files are those since the default branch rather than those of a pull request"}, nil
	}
	d, err := g.FetchPullRequestFiles(ctx, repo.Owner, repo.Repo, n, commit)
	if err != nil {
		return nil, err
	}
	if d.Unaligned != "" {
		// The table and the `patch` variable still read these lines, so the job log is the one
		// place that can say some of them may be other lines than the ones the pull request
		// changed.
		cmd.PrintErrf("Patch coverage may be measured over lines other than the changed ones, since some changed lines are numbered as the pull request head's rather than as commit %s's: %s\n", commit, d.Unaligned)
	}
	return d, nil
}

func printMetrics(cmd *cobra.Command) error {
	ctx := context.Background()
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
	r, err := report.New(c.Repository, report.Locale(c.Locale))
	if err != nil {
		return err
	}

	if err := c.CoverageConfigReadyOnLocal(); err == nil {
		if err := r.MeasureCoverage(c.Coverage.Paths, c.Coverage.Exclude); err != nil {
			cmd.PrintErrf("Skip measuring code coverage: %v\n", err)
		}
	}

	if err := c.CodeToTestRatioConfigReady(); err == nil {
		if err := r.MeasureCodeToTestRatio(c.Root(), c.CodeToTestRatio.Code, c.CodeToTestRatio.Test); err != nil {
			cmd.PrintErrf("Skip measuring code to test ratio: %v\n", err)
		}
	}

	if err := c.TestExecutionTimeConfigReady(); r.Repository != "" && err == nil {
		var stepNames []string
		if len(c.TestExecutionTime.Steps) > 0 {
			stepNames = c.TestExecutionTime.Steps
		}
		if err := r.MeasureTestExecutionTime(ctx, stepNames); err != nil {
			cmd.PrintErrf("Skip measuring test execution time: %v\n", err)
		}
	}

	if err := r.CollectCustomMetrics(); err != nil {
		cmd.PrintErrf("Skip collecting custom metrics: %v\n", err)
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
	rootCmd.Flags().StringVarP(&reportPath, "report", "r", "", "coverage report file path")
	rootCmd.Flags().BoolVarP(&createTable, "create-bq-table", "", false, "create table of BigQuery dataset")
}

func reportToDatastores(ctx context.Context, c *config.Config, datastores []string, r *report.Report) error {
	for _, s := range datastores {
		if datastore.NeedToShrink(s) {
			continue
		}
		d, err := datastore.New(ctx, s, datastore.Root(c.Root()), datastore.Report(r))
		if err != nil {
			return err
		}
		log.Printf("Storing report to %s", s)
		if err := d.StoreReport(ctx, r); err != nil {
			return err
		}
	}
	log.Println("Shrink report data")
	if r.Coverage != nil {
		r.Coverage.DeleteBlockCoverages()
	}
	if r.CodeToTestRatio != nil {
		r.CodeToTestRatio.DeleteFiles()
	}
	for _, s := range datastores {
		if !datastore.NeedToShrink(s) {
			continue
		}
		d, err := datastore.New(ctx, s, datastore.Root(c.Root()), datastore.Report(r))
		if err != nil {
			return err
		}
		log.Printf("Storing report to %s", s)
		if err := d.StoreReport(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// badgeFile writes the badge render produced to path. The badge is rendered into memory
// first, so a run that fails on a configuration error leaves the badge that is already there
// rather than truncating it to nothing.
func badgeFile(path string, render func(io.Writer) error) error {
	buf := new(bytes.Buffer)
	if err := render(buf); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil { // #nosec
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644) // #nosec
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

// comparedDatastore is a datastore of diff.datastores: opened for reading the report compared.
type comparedDatastore struct {
	name string
	fsys fs.FS
	// metadataRead is the artifact holding the metadata its report was read through, if any.
	metadataRead string
}

// readBaseReport returns the newest report stored under the first of keys that any of stores
// holds a report under, and the metadata it was read through. It goes key by key across every
// datastore rather than datastore by datastore, so that the report of the default branch in
// one datastore does not win on timestamp over the report of the base branch in another.
func readBaseReport(stores []comparedDatastore, prefix string, keys []string) (*report.Report, string) {
	var (
		rPrev        *report.Report
		metadataRead string
	)
	for _, key := range keys {
		path := fmt.Sprintf("%s/%s", prefix, report.Filename)
		if key != "" {
			path = fmt.Sprintf("%s/%s/%s", prefix, key, report.Filename)
		}
		for _, s := range stores {
			b, err := fs.ReadFile(s.fsys, path)
			if err != nil {
				log.Printf("%s: %v", s.name, err)
				continue
			}
			rt := &report.Report{}
			if err := json.Unmarshal(b, rt); err != nil {
				log.Printf("%s: %v %s", s.name, err, string(b))
				continue
			}
			// Select latest report
			if rPrev == nil || rPrev.Timestamp.UnixNano() < rt.Timestamp.UnixNano() {
				rPrev = rt
				metadataRead = s.metadataRead
			}
		}
		if rPrev != nil {
			return rPrev, metadataRead
		}
	}
	return nil, ""
}
