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
	from       string
}

func New(gh *gh.Gh, r, b, prefix string) (*Github, error) {
	return &Github{
		gh:         gh,
		repository: r,
		branch:     b,
		prefix:     prefix,
	}, nil
}

func (g *Github) StoreReport(ctx context.Context, r *report.Report) error {
	path := fmt.Sprintf("%s/report.json", r.Repository)
	g.from = r.Repository
	return g.Put(ctx, path, r.Bytes())
}

func (g *Github) Put(ctx context.Context, path string, content []byte) error {
	branch := g.branch
	message := fmt.Sprintf("Store coverage report of %s", g.from)
	repo, err := gh.Parse(g.repository)
	if err != nil {
		return err
	}
	cp := filepath.Join(g.prefix, path)
	return g.gh.PushContent(ctx, repo.Owner, repo.Repo, branch, string(content), cp, message)
}

func (g *Github) FS() (fs.FS, error) {
	return nil, errors.New("not implemented")
}
