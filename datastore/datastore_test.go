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
		wantType  string
		wantArgs  []string
		wantError bool
	}{
		{"github://owner/repo", "github", []string{"owner/repo", "", ""}, false},
		{"github://owner/repo/reports", "github", []string{"owner/repo", "", "reports"}, false},
		{"github://owner/repo/path/to/reports", "github", []string{"owner/repo", "", "path/to/reports"}, false},
		{"github://owner", "", []string{}, true},
		{"github://owner/repo@branch/reports", "github", []string{"owner/repo", "branch", "reports"}, false},
		{"github://owner/repo@branch/reports/", "github", []string{"owner/repo", "branch", "reports"}, false},
		{"s3://bucket/reports", "s3", []string{"bucket", "reports"}, false},
		{"s3://bucket/path/to/reports", "s3", []string{"bucket", "path/to/reports"}, false},
		{"s3://bucket", "s3", []string{"bucket", ""}, false},
		{"s3://bucket/", "s3", []string{"bucket", ""}, false},
		{"gs://bucket/reports", "gs", []string{"bucket", "reports"}, false},
		{"gs://bucket/path/to/reports", "gs", []string{"bucket", "path/to/reports"}, false},
		{"gs://bucket", "gs", []string{"bucket", ""}, false},
		{"gs://bucket/", "gs", []string{"bucket", ""}, false},
		{"bq://project/dataset/table", "bq", []string{"project", "dataset", "table"}, false},
		{"bq://project/dataset", "", []string{}, true},
		{"bq://project/dataset/table/more", "", []string{}, true},
		{"file://reports", "file", []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"reports", "file", []string{filepath.Join(testdataDir(t), "reports")}, false},
		{"file:///reports", "file", []string{"/reports"}, false},
		{"/reports", "file", []string{"/reports"}, false},
	}
	for _, tt := range tests {
		gotType, gotArgs, err := parse(tt.in, testdataDir(t))
		if err != nil {
			if !tt.wantError {
				t.Errorf("got %v", err)
			}
			continue
		}
		if err == nil && tt.wantError {
			t.Error("want error")
		}
		if gotType != tt.wantType {
			t.Errorf("got %v\nwant %v", gotType, tt.wantType)
		}
		if diff := cmp.Diff(gotArgs, tt.wantArgs, nil); diff != "" {
			t.Errorf("%s", diff)
		}
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
