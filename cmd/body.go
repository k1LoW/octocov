package cmd

import (
	"context"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/gh"
)

// replaceInsertReportToBody
func replaceInsertReportToBody(ctx context.Context, c *config.Config, content, key string) error {
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
	if err := g.ReplaceInsertToBody(ctx, repo.Owner, repo.Repo, n, content, key); err != nil {
		return err
	}
	return nil
}
