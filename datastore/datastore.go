package datastore

import (
	"context"

	"github.com/k1LoW/octocov/report"
)

type Datastore interface {
	Store(ctx context.Context, path string, r *report.Report) error
}
