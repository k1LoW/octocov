package datastore

import (
	"context"
	"fmt"
	"strings"

	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

type Github struct {
	gh         *gh.Gh
	repository string
	branch     string
}

func NewGithub(gh *gh.Gh, r, b string) (*Github, error) {
	return &Github{
		gh:         gh,
		repository: r,
		branch:     b,
	}, nil
}

func (g *Github) Store(ctx context.Context, path string, r *report.Report) error {
	branch := g.branch
	content := r.String()
	from := r.Repository
	if from == "" {
		from = "?"
	}
	message := fmt.Sprintf("Store coverage report of %s", from)
	splitted := strings.Split(g.repository, "/")
	owner := splitted[0]
	repo := splitted[1]

	return g.gh.PushContent(ctx, owner, repo, branch, content, path, message)
}
