package gcs

import (
	"context"
	"fmt"
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

func New(client *storage.Client, bucket, prefix string) (*GCS, error) {
	return &GCS{
		client: client,
		bucket: bucket,
		prefix: prefix,
	}, nil
}

func (g *GCS) StoreReport(ctx context.Context, r *report.Report) error {
	path := fmt.Sprintf("%s/report.json", r.Repository)
	return g.Put(ctx, path, r.Bytes())
}

func (g *GCS) Put(ctx context.Context, path string, content []byte) error {
	o := filepath.Join(g.prefix, path)
	w := g.client.Bucket(g.bucket).Object(o).NewWriter(ctx)
	if _, err := w.Write(content); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return nil
}

type FS struct {
	prefix string
	gscfs  *gcsfs.FS
}

func (fsys *FS) Open(name string) (fs.File, error) { //nostyle:recvnames
	return fsys.gscfs.Open(filepath.Join(fsys.prefix, name))
}

func (g *GCS) FS() (fs.FS, error) {
	return &FS{
		prefix: g.prefix,
		gscfs:  gcsfs.NewWithClient(g.client, g.bucket),
	}, nil
}
