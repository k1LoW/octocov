package datastore

import (
	"context"

	"cloud.google.com/go/storage"
	"github.com/k1LoW/octocov/report"
)

type GCS struct {
	client *storage.Client
	bucket string
}

func NewGCS(client *storage.Client, b string) (*GCS, error) {
	return &GCS{
		client: client,
		bucket: b,
	}, nil
}

func (g *GCS) Store(ctx context.Context, path string, r *report.Report) error {
	content := r.String()
	w := g.client.Bucket(g.bucket).Object(path).NewWriter(ctx)
	if _, err := w.Write([]byte(content)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}
