package datastore

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"cloud.google.com/go/storage"
	"github.com/k1LoW/octocov/report"
	"github.com/mauri870/gcsfs"
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

type GCSFS struct {
	prefix string
	gscfs  *gcsfs.FS
}

func (fsys *GCSFS) Open(name string) (fs.File, error) {
	return fsys.gscfs.Open(filepath.Join(fsys.prefix, name))
}

func (g *GCS) FS(path string) (fs.FS, error) {
	prefix := strings.Trim(path, "/")
	return &GCSFS{
		prefix: prefix,
		gscfs:  gcsfs.NewWithClient(g.client, g.bucket),
	}, nil
}
