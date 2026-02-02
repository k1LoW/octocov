package datastore

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cloud.google.com/go/auth/credentials"
	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/k1LoW/octocov/datastore/artifact"
	"github.com/k1LoW/octocov/datastore/bq"
	"github.com/k1LoW/octocov/datastore/gcs"
	"github.com/k1LoW/octocov/datastore/github"
	"github.com/k1LoW/octocov/datastore/local"
	"github.com/k1LoW/octocov/datastore/mackerel"
	s3d "github.com/k1LoW/octocov/datastore/s3"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
	mkr "github.com/mackerelio/mackerel-client-go"
	"google.golang.org/api/option"
)

type Type int

const (
	GitHub Type = iota + 1
	Artifact
	S3
	GCS
	BigQuery
	Mackerel
	Local

	UnknownType Type = 0
)

var (
	_ Datastore = (*github.Github)(nil)
	_ Datastore = (*artifact.Artifact)(nil)
	_ Datastore = (*s3d.S3)(nil)
	_ Datastore = (*gcs.GCS)(nil)
	_ Datastore = (*bq.BQ)(nil)
	_ Datastore = (*mackerel.Mackerel)(nil)
	_ Datastore = (*local.Local)(nil)
)

type Datastore interface {
	Put(ctx context.Context, path string, content []byte) error
	StoreReport(ctx context.Context, r *report.Report) error
	FS() (fs.FS, error)
}

func New(ctx context.Context, u string, hints ...HintFunc) (Datastore, error) {
	h := &hint{}
	for _, hf := range hints {
		if err := hf(h); err != nil {
			return nil, err
		}
	}
	d, args, err := parse(u, h.root)
	if err != nil {
		return nil, err
	}
	switch d {
	case GitHub:
		ownerrepo := args[0]
		branch := args[1]
		prefix := args[2]
		g, err := gh.New()
		if err != nil {
			return nil, err
		}
		if branch == "" {
			repo, err := gh.Parse(ownerrepo)
			if err != nil {
				return nil, err
			}
			branch, err = g.FetchDefaultBranch(ctx, repo.Owner, repo.Repo)
			if err != nil {
				return nil, err
			}
		}
		return github.New(g, ownerrepo, branch, prefix)
	case Artifact:
		ownerrepo := args[0]
		name := args[1]
		g, err := gh.New()
		if err != nil {
			return nil, err
		}
		return artifact.New(g, ownerrepo, name, h.report)
	case S3:
		bucket := args[0]
		prefix := args[1]
		cfg, err := config.LoadDefaultConfig(ctx)
		if err != nil {
			return nil, err
		}
		region, err := manager.GetBucketRegion(ctx, s3.NewFromConfig(cfg), bucket)
		if err != nil {
			return nil, err
		}
		cfg.Region = region
		sc := s3.NewFromConfig(cfg)
		return s3d.New(sc, bucket, prefix)
	case GCS:
		bucket := args[0]
		prefix := args[1]
		var client *storage.Client
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON") != "" {
			creds, err := credentials.DetectDefault(&credentials.DetectOptions{
				CredentialsJSON: []byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")),
			})
			if err != nil {
				return nil, err
			}
			client, err = storage.NewClient(ctx, option.WithAuthCredentials(creds))
			if err != nil {
				return nil, err
			}
		} else {
			client, err = storage.NewClient(ctx)
			if err != nil {
				return nil, err
			}
		}
		return gcs.New(client, bucket, prefix)
	case BigQuery:
		project := args[0]
		dataset := args[1]
		table := args[2]
		var client *bigquery.Client
		if os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON") != "" {
			creds, err := credentials.DetectDefault(&credentials.DetectOptions{
				CredentialsJSON: []byte(os.Getenv("GOOGLE_APPLICATION_CREDENTIALS_JSON")),
			})
			if err != nil {
				return nil, err
			}
			client, err = bigquery.NewClient(ctx, project, option.WithAuthCredentials(creds))
			if err != nil {
				return nil, err
			}
		} else {
			client, err = bigquery.NewClient(ctx, project)
			if err != nil {
				return nil, err
			}
		}
		return bq.New(client, dataset, table)
	case Mackerel:
		service := args[0]
		client := mkr.NewClient(os.Getenv("MACKEREL_API_KEY"))
		return mackerel.New(client, service)
	case Local:
		root := args[0]
		return local.New(root)
	}
	return nil, fmt.Errorf("invalid datastore: %s", u)
}

func parse(u, root string) (Type, []string, error) {
	switch {
	case strings.HasPrefix(u, "github://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "github://"), "/"), "/")
		if len(splitted) < 2 {
			return UnknownType, nil, fmt.Errorf("invalid datastore: %s", u)
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
		return GitHub, []string{ownerrepo, branch, prefix}, nil
	case strings.HasPrefix(u, "artifact://") || strings.HasPrefix(u, "artifacts://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(strings.TrimPrefix(u, "artifact://"), "artifacts://"), "/"), "/")
		if len(splitted) < 2 || len(splitted) > 3 {
			return UnknownType, nil, fmt.Errorf("invalid datastore: %s", u)
		}
		owner := splitted[0]
		repo := splitted[1]
		ownerrepo := fmt.Sprintf("%s/%s", owner, repo)
		name := ""
		if len(splitted) == 3 {
			name = splitted[2]
		}
		return Artifact, []string{ownerrepo, name}, nil
	case strings.HasPrefix(u, "s3://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "s3://"), "/"), "/")
		if splitted[0] == "" {
			return UnknownType, nil, fmt.Errorf("invalid datastore: %s", u)
		}
		bucket := splitted[0]
		prefix := strings.Join(splitted[1:], "/")
		return S3, []string{bucket, prefix}, nil
	case strings.HasPrefix(u, "gs://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "gs://"), "/"), "/")
		if splitted[0] == "" {
			return UnknownType, nil, fmt.Errorf("invalid datastore: %s", u)
		}
		bucket := splitted[0]
		prefix := ""
		if len(splitted) > 1 {
			prefix = strings.Join(splitted[1:], "/")
		}
		return GCS, []string{bucket, prefix}, nil
	case strings.HasPrefix(u, "bq://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(u, "bq://"), "/"), "/")
		if len(splitted) != 3 {
			return UnknownType, nil, fmt.Errorf("invalid datastore: %s", u)
		}
		project := splitted[0]
		dataset := splitted[1]
		table := splitted[2]
		return BigQuery, []string{project, dataset, table}, nil
	case strings.HasPrefix(u, "mackerel://") || strings.HasPrefix(u, "mkr://"):
		splitted := strings.Split(strings.Trim(strings.TrimPrefix(strings.TrimPrefix(u, "mackerel://"), "mkr://"), "/"), "/")
		if len(splitted) != 1 {
			return UnknownType, nil, fmt.Errorf("invalid datastore: %s", u)
		}
		service := splitted[0]
		return Mackerel, []string{service}, nil
	default:
		p := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(u, "file://"), "local://"), "/")
		if runtime.GOOS == "windows" && strings.HasPrefix(p, "/") {
			return UnknownType, nil, fmt.Errorf("invalid file path: %s", u)
		}
		p = filepath.FromSlash(p)
		if filepath.IsAbs(p) {
			root = p
		} else {
			root = filepath.Join(root, p)
		}
		return Local, []string{root}, nil
	}
}

func NeedToShrink(u string) bool {
	return strings.HasPrefix(u, "bq://")
}
