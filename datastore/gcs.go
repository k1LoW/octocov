package datastore

import (
	"context"
	"io/fs"
	"path/filepath"

	"cloud.google.com/go/storage"
	"github.com/k1LoW/octocov/report"
	"github.com/mauri870/gcsfs"
)

type GCS struct {
	client *storage.Client
	bucket string
	prefix string
}

func NewGCS(client *storage.Client, bucket, prefix string) (*GCS, error) {
	return &GCS{
		client: client,
		bucket: bucket,
		prefix: prefix,
	}, nil
}

func (g *GCS) Store(ctx context.Context, path string, r *report.Report) error {
	content := r.String()
	o := filepath.Join(g.bucket, path)
	w := g.client.Bucket(g.bucket).Object(o).NewWriter(ctx)
	if _, err := w.Write([]byte(content)); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

type GCSFS struct {
	prefix string
	gscfs  *gcsfs.FS
}

func (fsys *GCSFS) Open(name string) (fs.File, error) {
	return fsys.gscfs.Open(filepath.Join(fsys.prefix, name))
}

func (g *GCS) FS() (fs.FS, error) {
	return &GCSFS{
		prefix: g.prefix,
		gscfs:  gcsfs.NewWithClient(g.client, g.bucket),
	}, nil
}
