package report

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/k1LoW/octocov/gh"
)

func TestTable(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report.json"),
			`| Coverage |
|---------:|
| 68.5%    |
`,
		},
		{
			filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json"),
			`| Coverage | Code to Test Ratio | Test Execution Time |
|---------:|-------------------:|--------------------:|
| 68.5%    | 1:0.5              | 4m40s               |
`,
		},
	}
	for _, tt := range tests {
		r := &Report{}
		if err := r.MeasureCoverage(tt.path); err != nil {
			t.Fatal(err)
		}
		if got := r.Table(); got != tt.want {
			t.Errorf("got\n%v\nwant\n%v", got, tt.want)
		}
		orig := r.String()
		r.Coverage.DeleteBlockCoverages()
		deleted := r.String()
		if len(orig) <= len(deleted) {
			t.Error("DeleteBlockCoverages error")
		}
	}
}

func TestFileCoveagesTable(t *testing.T) {
	tests := []struct {
		files []*gh.PullRequestFile
		want  string
	}{
		{[]*gh.PullRequestFile{}, ""},
		{
			[]*gh.PullRequestFile{&gh.PullRequestFile{Filename: "config/yaml.go", BlobURL: "https://github.com/owner/repo/blob/xxx/config/yaml.go"}},
			`### Code coverage of files in pull request scope (41.7%)

|                                  Files                                  | Coverage |
|-------------------------------------------------------------------------|---------:|
| [config/yaml.go](https://github.com/owner/repo/blob/xxx/config/yaml.go) | 41.7%    |
`,
		},
	}
	path := filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report.json")
	r := &Report{}
	if err := r.MeasureCoverage(path); err != nil {
		t.Fatal(err)
	}
	for _, tt := range tests {
		if got := r.FileCoveagesTable(tt.files); got != tt.want {
			t.Errorf("got\n%v\nwant\n%v", got, tt.want)
		}
	}
}

func TestMergeExecutionTimes(t *testing.T) {
	tests := []struct {
		steps []gh.Step
		want  time.Duration
	}{
		{[]gh.Step{}, 0},
		{
			[]gh.Step{
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 15, 15, 5, 0, time.UTC),
				},
			},
			(11 * time.Minute),
		},
		{
			[]gh.Step{
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 15, 15, 5, 0, time.UTC),
				},
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 16, 4, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 16, 15, 5, 0, time.UTC),
				},
			},
			(22 * time.Minute),
		},
		{
			[]gh.Step{
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 15, 15, 5, 0, time.UTC),
				},
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 15, 5, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 15, 14, 5, 0, time.UTC),
				},
			},
			(11 * time.Minute),
		},
		{
			[]gh.Step{
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 15, 15, 5, 0, time.UTC),
				},
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 15, 5, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 15, 16, 5, 0, time.UTC),
				},
			},
			(12 * time.Minute),
		},
		{
			[]gh.Step{
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 15, 15, 5, 0, time.UTC),
				},
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 15, 5, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 15, 16, 5, 0, time.UTC),
				},
				gh.Step{
					StartedAt:   time.Date(2006, 1, 2, 15, 3, 5, 0, time.UTC),
					CompletedAt: time.Date(2006, 1, 2, 15, 13, 5, 0, time.UTC),
				},
			},
			(13 * time.Minute),
		},
	}
	for _, tt := range tests {
		got := mergeExecutionTimes(tt.steps)
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestCompare(t *testing.T) {
	a := &Report{}
	if err := a.MeasureCoverage(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	if err := b.MeasureCoverage(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json")); err != nil {
		t.Fatal(err)
	}
	got := a.Compare(b)
	if want := 0.0; got.Coverage.Diff != want {
		t.Errorf("got %v\nwant %v", got.Coverage.Diff, want)
	}
	if got.CodeToTestRatio != nil {
		t.Errorf("got %v\nwant %v", got.CodeToTestRatio, nil)
	}
	if got.TestExecutionTime != nil {
		t.Errorf("got %v\nwant %v", got.TestExecutionTime, nil)
	}
	{
		got := b.Compare(a)
		if want := 0.0; got.Coverage.Diff != want {
			t.Errorf("got %v\nwant %v", got.Coverage.Diff, want)
		}
		if want := -0.5143015828936407; got.CodeToTestRatio.Diff != want {
			t.Errorf("got %v\nwant %v", got.CodeToTestRatio.Diff, want)
		}
		if want := -280000000000.000000; got.TestExecutionTime.Diff != want {
			t.Errorf("got %v\nwant %v", got.TestExecutionTime.Diff, want)
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
