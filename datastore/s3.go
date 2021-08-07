package datastore

import (
	"bytes"
	"context"
	"io/fs"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/jszwec/s3fs"
	"github.com/k1LoW/octocov/report"
)

type S3 struct {
	client s3iface.S3API
	bucket string
	prefix string
}

func NewS3(client s3iface.S3API, bucket, prefix string) (*S3, error) {
	return &S3{
		client: client,
		bucket: bucket,
		prefix: prefix,
	}, nil
}

func (s *S3) Store(ctx context.Context, path string, r *report.Report) error {
	content := r.String()
	key := filepath.Join(s.prefix, path)
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          bytes.NewReader([]byte(content)),
		ContentLength: aws.Int64(int64(len(content))),
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *S3) FS() (fs.FS, error) {
	return fs.Sub(s3fs.New(s.client, s.bucket), s.prefix)
}
