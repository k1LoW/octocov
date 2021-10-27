package datastore

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/k1LoW/octocov/datastore/bq"
	"github.com/k1LoW/octocov/datastore/gcs"
	"github.com/k1LoW/octocov/datastore/github"
	"github.com/k1LoW/octocov/datastore/local"
	s3d "github.com/k1LoW/octocov/datastore/s3"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
	"google.golang.org/api/option"
)

type DatastoreType int

var (
	_ Datastore = (*github.Github)(nil)
	_ Datastore = (*s3d.S3)(nil)
	_ Datastore = (*gcs.GCS)(nil)
	_ Datastore = (*bq.BQ)(nil)
	_ Datastore = (*local.Local)(nil)
)

type Datastore interface {
	Put(ctx context.Context, path string, content []byte) error
	StoreReport(ctx context.Context, r *report.Report) error
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
		if branch == "" {
			owner, repo, err := gh.SplitRepository(repo)
			if err != nil {
				return nil, err
			}
			branch, err = g.GetDefaultBranch(ctx, owner, repo)
			if err != nil {
				return nil, err
			}
		}
		return github.New(g, repo, branch, prefix)
	case "s3":
		bucket := args[0]
		prefix := args[1]
		sess, err := session.NewSession()
		if err != nil {
			return nil, err
		}
		sc := s3.New(sess)
		return s3d.New(sc, bucket, prefix)
	case "gs":
		bucket := args[0]
		prefix := args[1]
		var client *storage.Client
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON") != "" {
			client, err = storage.NewClient(ctx, option.WithCredentialsJSON([]byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON"))))
		} else {
			client, err = storage.NewClient(ctx)
		}
		if err != nil {
			return nil, err
		}
		return gcs.New(client, bucket, prefix)
	case "bq":
		project := args[0]
		dataset := args[1]
		table := args[2]
		var client *bigquery.Client
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON") != "" {
			client, err = bigquery.NewClient(ctx, project, option.WithCredentialsJSON([]byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON"))))
		} else {
			client, err = bigquery.NewClient(ctx, project)
		}
		if err != nil {
			return nil, err
		}
		return bq.New(client, dataset, table)
	case "local":
		root := args[0]
		return local.New(root)
	}
	return nil, fmt.Errorf("invalid datastore: %s", u)
}

func parse(u, configRoot string) (datastore string, args []string, err error) {
	switch {
	case strings.HasPrefix(u, "github://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "github://"), "/"), "/")
		if len(splitted) < 2 {
			return "", nil, fmt.Errorf("invalid datastore: %s", u)
		}
		branch := ""
		owner := splitted[0]
		repo := splitted[1]
		if strings.Contains(repo, "@") {
			splitted := strings.Split(repo, "@")
			repo = splitted[0]
			branch = splitted[1]
		}
		ownerrepo := fmt.Sprintf("%s/%s", owner, repo)
		prefix := strings.Join(splitted[2:], "/")
		return "github", []string{ownerrepo, branch, prefix}, nil
	case strings.HasPrefix(u, "s3://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "s3://"), "/"), "/")
		if splitted[0] == "" {
			return "", nil, fmt.Errorf("invalid datastore: %s", u)
		}
		bucket := splitted[0]
		prefix := strings.Join(splitted[1:], "/")
		return "s3", []string{bucket, prefix}, nil
	case strings.HasPrefix(u, "gs://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "gs://"), "/"), "/")
		if splitted[0] == "" {
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
		p := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(u, "file://"), "local://"), "/")
		if strings.HasPrefix(p, "/") {
			root = p
		} else {
			root = filepath.Join(root, p)
		}
		return "local", []string{root}, nil
	}
}
