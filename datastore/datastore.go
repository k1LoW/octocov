package datastore

import (
	"context"
	"io/fs"

	"github.com/k1LoW/octocov/report"
)

type Datastore interface {
	Store(ctx context.Context, path string, r *report.Report) error
	FS(path string) (fs.FS, error)
}
