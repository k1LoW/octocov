package cmd

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/k1LoW/octocov/config"
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

func createReportContent(ctx context.Context, c *config.Config, r, rPrev *report.Report, message string, hideFooterLink bool) (string, error) {
	repo, err := gh.Parse(c.Repository)
	if err != nil {
		return "", err
	}
	g, err := gh.New()
	if err != nil {
		return "", err
	}
	var files []*gh.PullRequestFile
	n, err := g.DetectCurrentPullRequestNumber(ctx, repo.Owner, repo.Repo)
	if err == nil {
		files, err = g.FetchPullRequestFiles(ctx, repo.Owner, repo.Repo, n)
		if err != nil {
			return "", err
		}
	} else {
		files, err = g.FetchChangedFiles(ctx, repo.Owner, repo.Repo)
		if err != nil {
			return "", err
		}
	}
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
	if err := c.Acceptable(r, rPrev); err != nil {
		merr, ok := err.(*multierror.Error) //nolint:errorlint
		if !ok {
			return "", fmt.Errorf("failed to convert error to multierror: %w", err)
		}
		merr.ErrorFormat = func(errors []error) string {
			var out string
			for _, err := range errors {
				out += fmt.Sprintf("**:no_entry_sign: %s**\n\n", capitalize(err.Error()))
			}
			return out
		}
		comment = append(comment, merr.Error())
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
