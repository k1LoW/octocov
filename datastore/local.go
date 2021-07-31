package datastore

import (
	"context"
	"fmt"
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

func (g *Local) Store(ctx context.Context, path string, r *report.Report) error {
	return os.WriteFile(filepath.Join(root, path), []byte(r.String()), os.ModePerm)
}
