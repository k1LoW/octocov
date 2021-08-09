package github

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

type Github struct {
	gh         *gh.Gh
	repository string
	branch     string
	prefix     string
}

func New(gh *gh.Gh, r, b, prefix string) (*Github, error) {
	return &Github{
		gh:         gh,
		repository: r,
		branch:     b,
		prefix:     prefix,
	}, nil
}

func (g *Github) Store(ctx context.Context, r *report.Report) error {
	path := fmt.Sprintf("%s/report.json", r.Repository)
	branch := g.branch
	content := r.String()
	message := fmt.Sprintf("Store coverage report of %s", r.Repository)
	owner, repo, err := gh.SplitRepository(g.repository)
	if err != nil {
		return err
	}
	cp := filepath.Join(g.prefix, path)
	return g.gh.PushContent(ctx, owner, repo, branch, content, cp, message)
}

func (g *Github) FS() (fs.FS, error) {
	return nil, errors.New("not implemented")
}
