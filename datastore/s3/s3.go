package s3

import (
	"bytes"
	"context"
	"fmt"
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

func New(client s3iface.S3API, bucket, prefix string) (*S3, error) {
	return &S3{
		client: client,
		bucket: bucket,
		prefix: prefix,
	}, nil
}

func (s *S3) StoreReport(ctx context.Context, r *report.Report) error {
	path := fmt.Sprintf("%s/report.json", r.Repository)
	return s.Put(ctx, path, r.Bytes())
}

func (s *S3) Put(ctx context.Context, path string, content []byte) error {
	key := filepath.Join(s.prefix, path)
	_, err := s.client.PutObject(&s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          bytes.NewReader(content),
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
