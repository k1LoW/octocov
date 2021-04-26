package datastore

import (
	"context"

	"github.com/k1LoW/octocov/report"
)

type Datastore interface {
	Push(ctx context.Context, r *report.Report) error
}
