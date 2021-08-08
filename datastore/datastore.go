package datastore

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

type DatastoreType int

var (
	_ Datastore = (*Github)(nil)
	_ Datastore = (*S3)(nil)
	_ Datastore = (*GCS)(nil)
	_ Datastore = (*BQ)(nil)
)

type Datastore interface {
	Store(ctx context.Context, r *report.Report) error
	FS() (fs.FS, error)
}

func New(ctx context.Context, u, configRoot string) (Datastore, error) {
	d, args, err := parse(u, configRoot)
	if err != nil {
		return nil, err
	}
	switch d {
	case "github":
		repo := args[0]
		branch := args[1]
		prefix := args[2]
		g, err := gh.New()
		if err != nil {
			return nil, err
		}
		return NewGithub(g, repo, branch, prefix)
	case "s3":
		bucket := args[0]
		prefix := args[1]
		sess, err := session.NewSession()
		if err != nil {
			return nil, err
		}
		sc := s3.New(sess)
		return NewS3(sc, bucket, prefix)
	case "gs":
		bucket := args[0]
		prefix := args[1]
		client, err := storage.NewClient(ctx)
		if err != nil {
			return nil, err
		}
		return NewGCS(client, bucket, prefix)
	case "bq":
		project := args[0]
		dataset := args[1]
		table := args[2]
		client, err := bigquery.NewClient(ctx, project)
		if err != nil {
			return nil, err
		}
		return NewBQ(client, dataset, table)
	case "file":
		root := args[0]
		return NewLocal(root)
	}
	return nil, fmt.Errorf("invalid datastore: %s", u)
}

func parse(u, configRoot string) (datastore string, args []string, err error) {
	switch {
	case strings.HasPrefix(u, "github://"):
		branch := ""
		{
			splitted := strings.Split(u, "@")
			u = splitted[0]
			if len(splitted) == 2 {
				branch = splitted[1]
			}
		}
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "github://"), "/"), "/")
		if len(splitted) < 2 {
			return "", nil, fmt.Errorf("invalid datastore: %s", u)
		}
		repo := fmt.Sprintf("%s/%s", splitted[0], splitted[1])
		prefix := strings.Join(splitted[2:], "/")
		return "github", []string{repo, branch, prefix}, nil
	case strings.HasPrefix(u, "s3://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "s3://"), "/"), "/")
		if len(splitted) == 0 {
			return "", nil, fmt.Errorf("invalid datastore: %s", u)
		}
		bucket := splitted[0]
		prefix := strings.Join(splitted[1:], "/")
		return "s3", []string{bucket, prefix}, nil
	case strings.HasPrefix(u, "gs://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "gs://"), "/"), "/")
		if len(splitted) == 0 {
			return "", nil, fmt.Errorf("invalid datastore: %s", u)
		}
		bucket := splitted[0]
		prefix := ""
		if len(splitted) > 1 {
			prefix = strings.Join(splitted[1:], "/")
		}
		return "gs", []string{bucket, prefix}, nil
	case strings.HasPrefix(u, "bq://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "bq://"), "/"), "/")
		if len(splitted) != 3 {
			return "", nil, fmt.Errorf("invalid datastore: %s", u)
		}
		project := splitted[0]
		dataset := splitted[1]
		table := splitted[2]
		return "bq", []string{project, dataset, table}, nil
	default:
		root := configRoot
		p := strings.TrimSuffix(strings.TrimPrefix(u, "file://"), "/")
		if strings.HasPrefix(p, "/") {
			root = p
		} else {
			root = filepath.Join(root, p)
		}
		return "file", []string{root}, nil
	}
}
