package datastore

import (
	"bytes"
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/jszwec/s3fs"
	"github.com/k1LoW/octocov/report"
)

type S3 struct {
	client s3iface.S3API
	bucket string
}

func NewS3(client s3iface.S3API, b string) (*S3, error) {
	return &S3{
		client: client,
		bucket: b,
	}, nil
}

func (s *S3) Store(ctx context.Context, path string, r *report.Report) error {
	content := r.String()
	bucket := s.bucket
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket:        &bucket,
		Key:           &path,
		Body:          bytes.NewReader([]byte(content)),
		ContentLength: aws.Int64(int64(len(content))),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *S3) ReadDirFS(path string) (fs.ReadDirFS, error) {
	return &S3FSWithPrefix{
		prefix: strings.Trim(path, "/"),
		s3fs:   s3fs.New(s.client, s.bucket),
	}, nil
}

type S3FSWithPrefix struct {
	prefix string
	s3fs   *s3fs.S3FS
}

func (fsys *S3FSWithPrefix) Open(name string) (fs.File, error) {
	return fsys.s3fs.Open(filepath.Join(fsys.prefix, name))
}

func (fsys *S3FSWithPrefix) ReadDir(name string) ([]fs.DirEntry, error) {
	return fsys.s3fs.ReadDir(filepath.Join(fsys.prefix, name))
}
