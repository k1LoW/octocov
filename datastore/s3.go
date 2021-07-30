package datastore

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/report"
)

type S3 struct {
	config *config.Config
	client s3iface.S3API
}

func NewS3(c *config.Config, client s3iface.S3API) (*S3, error) {
	return &S3{
		config: c,
		client: client,
	}, nil
}

func (s *S3) Store(ctx context.Context, r *report.Report) error {
	content := r.String()
	bucket := s.config.Datastore.S3.Bucket
	path := s.config.Datastore.S3.Path
	from := r.Repository
	if s.config.Repository != "" {
		from = s.config.Repository
	}
	if from == "" {
		return fmt.Errorf("report '%s' is not set", "repository")
	}
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
