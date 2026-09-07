package bq

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io/fs"
	"testing/fstest"
	"time"

	"cloud.google.com/go/bigquery"
	"github.com/goccy/go-json"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
	"github.com/oklog/ulid/v2"
	"google.golang.org/api/iterator"
)

type BQ struct {
	client  *bigquery.Client
	dataset string
	table   string
}

func New(client *bigquery.Client, dataset, table string) (*BQ, error) {
	return &BQ{
		client:  client,
		dataset: dataset,
		table:   table,
	}, nil
}

type ReportRecord struct {
	Id                  string               `bigquery:"id"`
	Owner               string               `bigquery:"owner"`
	Repo                string               `bigquery:"repo"`
	Ref                 string               `bigquery:"ref"`
	Commit              string               `bigquery:"commit"`
	CoverageTotal       bigquery.NullInt64   `bigquery:"coverage_total"`
	CoverageCovered     bigquery.NullInt64   `bigquery:"coverage_covered"`
	CodeToTestRatioCode bigquery.NullInt64   `bigquery:"code_to_test_ratio_code"`
	CodeToTestRatioTest bigquery.NullInt64   `bigquery:"code_to_test_ratio_test"`
	TestExecutionTime   bigquery.NullFloat64 `bigquery:"test_execution_time"`
	Timestamp           time.Time            `bigquery:"timestamp"`
	Raw                 string               `bigquery:"raw"`
}

var reportsSchema = bigquery.Schema{
	&bigquery.FieldSchema{Name: "id", Type: bigquery.StringFieldType, Required: true},
	&bigquery.FieldSchema{Name: "owner", Type: bigquery.StringFieldType, Required: true},
	&bigquery.FieldSchema{Name: "repo", Type: bigquery.StringFieldType, Required: true},
	&bigquery.FieldSchema{Name: "ref", Type: bigquery.StringFieldType, Required: true},
	&bigquery.FieldSchema{Name: "commit", Type: bigquery.StringFieldType, Required: true},
	&bigquery.FieldSchema{Name: "coverage_total", Type: bigquery.IntegerFieldType, Required: false},
	&bigquery.FieldSchema{Name: "coverage_covered", Type: bigquery.IntegerFieldType, Required: false},
	&bigquery.FieldSchema{Name: "code_to_test_ratio_code", Type: bigquery.IntegerFieldType, Required: false},
	&bigquery.FieldSchema{Name: "code_to_test_ratio_test", Type: bigquery.IntegerFieldType, Required: false},
	&bigquery.FieldSchema{Name: "test_execution_time", Type: bigquery.NumericFieldType, Required: false},
	&bigquery.FieldSchema{Name: "timestamp", Type: bigquery.TimestampFieldType, Required: true},
	&bigquery.FieldSchema{Name: "raw", Type: bigquery.StringFieldType, Required: true},
}

func (b *BQ) StoreReport(ctx context.Context, r *report.Report) error {
	u := b.client.Dataset(b.dataset).Table(b.table).Uploader()
	repo, err := gh.Parse(r.Repository)
	if err != nil {
		return nil
	}
	id, err := ulid.New(ulid.Timestamp(time.Now()), rand.Reader)
	if err != nil {
		return nil
	}
	rr := &ReportRecord{
		Id:        id.String(),
		Owner:     repo.Owner,
		Repo:      repo.Reponame(),
		Ref:       r.Ref,
		Commit:    r.Commit,
		Timestamp: r.Timestamp,
		Raw:       r.String(),
	}

	if r.Coverage != nil {
		rr.CoverageTotal = bigquery.NullInt64{
			Int64: int64(r.Coverage.Total),
			Valid: true,
		}
		rr.CoverageCovered = bigquery.NullInt64{
			Int64: int64(r.Coverage.Covered),
			Valid: true,
		}
	}
	if r.CodeToTestRatio != nil {
		rr.CodeToTestRatioCode = bigquery.NullInt64{
			Int64: int64(r.CodeToTestRatio.Code),
			Valid: true,
		}
		rr.CodeToTestRatioTest = bigquery.NullInt64{
			Int64: int64(r.CodeToTestRatio.Test),
			Valid: true,
		}
	}
	if r.TestExecutionTime != nil {
		rr.TestExecutionTime = bigquery.NullFloat64{
			Float64: r.TestExecutionTimeNano(),
			Valid:   true,
		}
	}
	return u.Put(ctx, []*ReportRecord{rr})
}

func (b *BQ) Put(ctx context.Context, path string, context []byte) error {
	return errors.New("not implemented")
}

func (b *BQ) CreateTable(ctx context.Context) error {
	metaData := &bigquery.TableMetadata{
		Schema: reportsSchema,
	}
	tableRef := b.client.Dataset(b.dataset).Table(b.table)
	if err := tableRef.Create(ctx, metaData); err != nil {
		return err
	}
	return nil
}

func (b *BQ) FS() (fs.FS, error) {
	ctx := context.Background()
	fsys := fstest.MapFS{}
	t := fmt.Sprintf("`%s.%s`", b.dataset, b.table)
	// Grouped by the pull request as well as by the ref, which together are as fine a key
	// as the one the rows are laid out under below. Grouping by repository alone would let
	// the newest row of any ref answer for the default branch, and grouping by ref alone
	// would still collapse a pull_request_target run, where GITHUB_REF holds the base
	// branch and a pull request shares its ref with the default branch.
	stmt := `SELECT r.owner, r.repo, r.timestamp, r.raw FROM %s AS r
INNER JOIN (
    SELECT owner, repo, ref, COALESCE(JSON_VALUE(raw, '$.pull_request'), '') AS pull_request, MAX(timestamp) AS timestamp
    FROM %s GROUP BY owner, repo, ref, pull_request
) AS l ON r.owner = l.owner AND r.repo = l.repo AND r.ref = l.ref
    AND COALESCE(JSON_VALUE(r.raw, '$.pull_request'), '') = l.pull_request
    AND l.timestamp = r.timestamp
ORDER BY r.owner, r.repo, r.ref`
	q := b.client.Query(fmt.Sprintf(stmt, t, t)) //nolint:nosec
	it, err := q.Read(ctx)
	if err != nil {
		return nil, err
	}
	for {
		var rr ReportRecord
		err := it.Next(&rr)
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		path := fmt.Sprintf("%s/%s/%s", rr.Owner, rr.Repo, report.Filename)
		rt := &report.Report{}
		if err := json.Unmarshal([]byte(rr.Raw), rt); err == nil {
			if k := rt.RefKey(); k != "" {
				path = fmt.Sprintf("%s/%s/%s/%s", rr.Owner, rr.Repo, k, report.Filename)
			}
		}
		// Rows written before the ref was recorded all resolve to the same path, so the
		// newest of them is the one kept, as it was when the query returned only one row
		// per repository.
		if e, ok := fsys[path]; ok && e.ModTime.After(rr.Timestamp) {
			continue
		}
		fsys[path] = &fstest.MapFile{
			Data:    []byte(rr.Raw),
			Mode:    fs.ModePerm,
			ModTime: rr.Timestamp,
		}
	}
	return &fsys, nil
}
