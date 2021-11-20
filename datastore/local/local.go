package local

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/k1LoW/octocov/report"
)

type Local struct {
	root string
}

func New(root string) (*Local, error) {
	p, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(p)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("%s is not directory", root)
	}
	return &Local{
		root: root,
	}, nil
}

func (l *Local) Root() string {
	return l.root
}

func (l *Local) StoreReport(ctx context.Context, r *report.Report) error {
	path := fmt.Sprintf("%s/report.json", r.Repository)
	return l.Put(ctx, path, r.Bytes())
}

func (l *Local) Put(ctx context.Context, path string, content []byte) error {
	p := filepath.Join(l.root, path)
	dir := filepath.Dir(p)
	if _, err := os.Stat(dir); err != nil {
		err := os.MkdirAll(dir, 0755) // #nosec
		if err != nil {
			return err
		}
	}
	return os.WriteFile(p, content, os.ModePerm)
}

func (l *Local) FS() (fs.FS, error) {
	return os.DirFS(l.root), nil
}
