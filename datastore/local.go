package datastore

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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
	if !strings.HasPrefix(path, "/") {
		path = filepath.Join(l.root, path)
	}
	return &LocalFS{
		root: path,
	}, nil
}

type LocalFS struct {
	root string
}

func (fsys *LocalFS) Open(name string) (fs.File, error) {
	f, err := os.Open(filepath.Clean(filepath.Join(fsys.root, name)))
	if f == nil {
		return nil, err
	}
	return f, err
}

func (fsys *LocalFS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(filepath.Join(fsys.root, name))
}
