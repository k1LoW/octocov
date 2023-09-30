package datastore

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	var tests = []struct {
		in        string
		wantType  Type
		wantArgs  []string
		wantError bool
	}{
		{"github://owner/repo", GitHub, []string{"owner/repo", "", ""}, false},
		{"github://owner/repo/reports", GitHub, []string{"owner/repo", "", "reports"}, false},
		{"github://owner/repo/path/to/reports", GitHub, []string{"owner/repo", "", "path/to/reports"}, false},
		{"github://owner", UnknownType, []string{}, true},
		{"github://owner/repo@branch/reports", GitHub, []string{"owner/repo", "branch", "reports"}, false},
		{"github://owner/repo@branch/reports/", GitHub, []string{"owner/repo", "branch", "reports"}, false},
		{"artifact://owner/repo", Artifact, []string{"owner/repo", ""}, false},
		{"artifact://owner/repo/reports", Artifact, []string{"owner/repo", "reports"}, false},
		{"artifact://owner/repo/path/to/reports", UnknownType, []string{}, true},
		{"artifact://owner", UnknownType, []string{}, true},
		{"artifacts://owner/repo", Artifact, []string{"owner/repo", ""}, false},
		{"s3://bucket/reports", S3, []string{"bucket", "reports"}, false},
		{"s3://bucket/path/to/reports", S3, []string{"bucket", "path/to/reports"}, false},
		{"s3://bucket", S3, []string{"bucket", ""}, false},
		{"s3://bucket/", S3, []string{"bucket", ""}, false},
		{"s3://", UnknownType, []string{}, true},
		{"gs://bucket/reports", GCS, []string{"bucket", "reports"}, false},
		{"gs://bucket/path/to/reports", GCS, []string{"bucket", "path/to/reports"}, false},
		{"gs://bucket", GCS, []string{"bucket", ""}, false},
		{"gs://bucket/", GCS, []string{"bucket", ""}, false},
		{"gs://", UnknownType, []string{}, true},
		{"bq://project/dataset/table", BigQuery, []string{"project", "dataset", "table"}, false},
		{"bq://project/dataset", UnknownType, []string{}, true},
		{"bq://project/dataset/table/more", UnknownType, []string{}, true},
		{"mackerel://service", Mackerel, []string{"service"}, false},
		{"mkr://service", Mackerel, []string{"service"}, false},
		{"mkr://service/foo", UnknownType, []string{}, true},
		{"file://reports", Local, []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"reports", Local, []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"file:///reports", Local, []string{"/reports"}, false},
		{"/reports", Local, []string{"/reports"}, false},
		{"local://reports", Local, []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"local://./reports", Local, []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"local:///reports", Local, []string{"/reports"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			gotType, gotArgs, err := parse(tt.in, testdataDir(t))
			if err != nil {
				if !tt.wantError {
					t.Errorf("got %v", err)
				}
				return
			}
			if err == nil && tt.wantError {
				t.Error("want error")
			}
			if gotType != tt.wantType {
				t.Errorf("got %v\nwant %v", gotType, tt.wantType)
			}
			if diff := cmp.Diff(gotArgs, tt.wantArgs, nil); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func testdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(wd), "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
