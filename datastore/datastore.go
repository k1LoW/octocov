package datastore

import (
	"context"
	"io/fs"

	"github.com/k1LoW/octocov/report"
)

var (
	_ Datastore = (*Github)(nil)
	_ Datastore = (*S3)(nil)
	_ Datastore = (*GCS)(nil)
	_ Datastore = (*BQ)(nil)
)

type Datastore interface {
	Store(ctx context.Context, path string, r *report.Report) error
	FS() (fs.FS, error)
}
