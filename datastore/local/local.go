package local

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/k1LoW/octocov/report"
	"github.com/k1LoW/osfs"
)

type Local struct {
	root string
}

func New(root string) (*Local, error) {
	fi, err := os.Stat(root)
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
	return osfs.New().Sub(strings.TrimPrefix(l.root, "/"))
}
