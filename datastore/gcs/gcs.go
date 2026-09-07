package gcs

import (
	"context"
	"io/fs"
	"path"

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
	path := r.StorePath()
	return g.Put(ctx, path, r.Bytes())
}

func (g *GCS) Put(ctx context.Context, p string, content []byte) error {
	// path.Join rather than filepath.Join, since an object name is slash separated on
	// every OS, and the separator filepath would pick on Windows would store the report
	// under a name nothing reads back.
	o := path.Join(g.prefix, p)
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
	return fsys.gscfs.Open(path.Join(fsys.prefix, name))
}

func (g *GCS) FS() (fs.FS, error) {
	return &FS{
		prefix: g.prefix,
		gscfs:  gcsfs.NewWithClient(g.client, g.bucket),
	}, nil
}
