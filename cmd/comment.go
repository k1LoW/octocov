package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

func commentReport(ctx context.Context, c *config.Config, r, rPrev *report.Report) error {
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
	files, err := g.GetPullRequestFiles(ctx, repo.Owner, repo.Repo, n)
	if err != nil {
		return err
	}
	footer := "Reported by [octocov](https://github.com/k1LoW/octocov)"
	if c.Comment.HideFooterLink {
		footer = "Reported by octocov"
	}
	var table, fileTable string
	if rPrev != nil {
		d := rPrev.Compare(r)
		table = d.Table()
		fileTable = d.FileCoveagesTable(files)
	} else {
		table = r.Table()
		fileTable = r.FileCoveagesTable(files)
	}

	comment := strings.Join([]string{
		"## Code Metrics Report",
		table,
		"",
		fileTable,
		"---",
		footer,
	}, "\n")

	if err := c.Acceptable(r, rPrev); err != nil {
		comment = fmt.Sprintf("**:no_entry_sign: %s**\n\n%s", err.Error(), comment)
	}

	if err := g.PutComment(ctx, repo.Owner, repo.Repo, n, comment); err != nil {
		return err
	}
	return nil
}
