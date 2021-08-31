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

func (l *Local) Store(ctx context.Context, r *report.Report) error {
	path := fmt.Sprintf("%s/report.json", r.Repository)
	return os.WriteFile(filepath.Join(l.root, path), r.Bytes(), os.ModePerm)
}

func (l *Local) FS() (fs.FS, error) {
	return osfs.New().Sub(strings.TrimPrefix(l.root, "/"))
}
