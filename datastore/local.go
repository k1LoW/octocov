package datastore

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/k1LoW/octocov/report"
)

type Local struct {
	root string
}

func NewLocal(root string) (*Local, error) {
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

func (l *Local) Store(ctx context.Context, path string, r *report.Report) error {
	return os.WriteFile(filepath.Join(l.root, path), []byte(r.String()), os.ModePerm)
}

func (l *Local) ReadDirDS(path string) (fs.ReadDirFS, error) {
	return nil, errors.New("not implemented")
}
