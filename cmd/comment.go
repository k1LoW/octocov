package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/k1LoW/errors"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

func commentReport(ctx context.Context, c *config.Config, content, key string) error {
	repo, err := gh.Parse(c.Repository)
	if err != nil {
		return err
	}
	g, err := gh.New()
	if err != nil {
		return err
	}
	n, err := g.DetectCurrentPullRequestNumber(ctx, repo.Owner, repo.Repo)
	if err != nil {
		return err
	}
	if c.Comment.DeletePrevious {
		if err := g.PutCommentWithDeletion(ctx, repo.Owner, repo.Repo, n, content, key); err != nil {
			return err
		}
	} else if c.Comment.UpdatePrevious {
		if err := g.PutCommentWithUpdate(ctx, repo.Owner, repo.Repo, n, content, key); err != nil {
			return err
		}
	} else {
		if err := g.PutComment(ctx, repo.Owner, repo.Repo, n, content, key); err != nil {
			return err
		}
	}
	return nil
}

// createReportContent renders the report for one of the pull request outputs. files is the
// changed file list, fetched by the caller so that the three outputs and the patch coverage
// measurement share one paginated fetch instead of repeating it.
func createReportContent(c *config.Config, r, rPrev *report.Report, files []*gh.PullRequestFile, message string, hideFooterLink bool) (string, error) {
	footer := "Reported by [octocov](https://github.com/k1LoW/octocov)"
	if hideFooterLink {
		footer = "Reported by octocov"
	}
	var (
		table, fileTable string
		customTables     []string
	)
	if rPrev != nil {
		d := r.Compare(rPrev)
		table = d.Table()
		relWd := c.Root()
		if c.GitRoot != "" {
			if rw, err := filepath.Rel(c.GitRoot, c.Root()); err == nil {
				relWd = filepath.ToSlash(rw)
			}
		}
		if relWd == "." {
			relWd = ""
		}
		fileTable = d.FileCoveragesTable(files, relWd)
		for _, s := range d.CustomMetrics {
			customTables = append(customTables, s.Table(), s.MetadataTable())
		}
	} else {
		table = r.Table()
		fileTable = r.FileCoveragesTable(files)
		for _, s := range r.CustomMetrics {
			customTables = append(customTables, s.Table(), s.MetadataTable())
		}
	}

	var comment []string
	if r.IsMeasuredCoverage() || r.IsMeasuredTestExecutionTime() || r.IsMeasuredCodeToTestRatio() {
		comment = append(comment, fmt.Sprintf("## %s", r.Title()))
	}
	if message != "" {
		comment = append(comment, message)
	}
	// Measure patch coverage only when the condition can read it, so a configuration that
	// never references `patch` does not pay for the lookup over every changed file.
	var pc *coverage.PatchCoverage
	if c.Coverage.AcceptableReferencesPatch() && c.CoverageConfigReady() == nil {
		pc = r.PatchCoverage(gh.ChangedLinesByFile(files))
	}
	if err := c.Acceptable(r, rPrev, pc); err != nil {
		errs := errors.Errors(err)
		var b strings.Builder
		for _, e := range errs {
			fmt.Fprintf(&b, "**:no_entry_sign: %s**\n\n", capitalize(e.Error()))
		}
		comment = append(comment, b.String())
	}
	if r.IsMeasuredCoverage() || r.IsMeasuredTestExecutionTime() || r.IsMeasuredCodeToTestRatio() {
		comment = append(comment, table, "", fileTable)
	}
	comment = append(comment, customTables...)
	comment = append(comment, "---", footer)

	return strings.Join(comment, "\n"), nil
}

func capitalize(w string) string {
	splitted := strings.SplitN(w, "", 2)
	switch len(splitted) {
	case 0:
		return ""
	case 1:
		return strings.ToUpper(splitted[0])
	default:
		return strings.ToUpper(splitted[0]) + splitted[1]
	}
}
