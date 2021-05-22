package datastore

import (
	"context"
	"fmt"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

type Github struct {
	config *config.Config
	gh     *gh.Gh
}

func NewGithub(c *config.Config, gh *gh.Gh) (*Github, error) {
	return &Github{
		config: c,
		gh:     gh,
	}, nil
}

func (g *Github) Store(ctx context.Context, r *report.Report) error {
	branch := g.config.Datastore.Github.Branch
	content := r.String()
	path := g.config.Datastore.Github.Path
	from := r.Repository
	if g.config.Repository != "" {
		from = g.config.Repository
	}
	message := fmt.Sprintf("Store coverage report of %s", from)
	splitted := strings.Split(g.config.Datastore.Github.Repository, "/")
	owner := splitted[0]
	repo := splitted[1]

	return g.gh.PushContent(ctx, owner, repo, branch, content, path, message)
}
