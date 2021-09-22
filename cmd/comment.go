package cmd

import (
	"context"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

func commentReport(ctx context.Context, c *config.Config, r, r2 *report.Report) error {
	owner, repo, err := gh.SplitRepository(c.Repository)
	if err != nil {
		return err
	}
	gh, err := gh.New()
	if err != nil {
		return err
	}
	n, err := gh.DetectCurrentPullRequestNumber(ctx, owner, repo)
	if err != nil {
		return err
	}
	files, err := gh.GetPullRequestFiles(ctx, owner, repo, n)
	if err != nil {
		return err
	}
	footer := "Reported by [octocov](https://github.com/k1LoW/octocov)"
	if c.Comment.HideFooterLink {
		footer = "Reported by octocov"
	}
	comment := strings.Join([]string{
		"## Code Metrics Report",
		r.Table(),
		"",
		r.FileCoveagesTable(files),
		"---",
		footer,
	}, "\n")
	if err := gh.PutComment(ctx, owner, repo, n, comment); err != nil {
		return err
	}
	return nil
}
