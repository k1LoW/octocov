package s3

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jszwec/s3fs/v2"
	"github.com/k1LoW/octocov/report"
)

// Client is an interface for S3 client.
type Client interface {
	// https://pkg.go.dev/github.com/aws/aws-sdk-go-v2/service/s3#Client.PutObject
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	// https://pkg.go.dev/github.com/jszwec/s3fs/v2#Client
	s3fs.Client
}

var _ Client = (*s3.Client)(nil)

type S3 struct {
	client Client
	bucket string
	prefix string
}

func New(client Client, bucket, prefix string) (*S3, error) {
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
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
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
