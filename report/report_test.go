package report

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/pkg/coverage"
	"github.com/k1LoW/octocov/pkg/ratio"
)

func TestNew(t *testing.T) {
	tests := []struct {
		envrepo   string
		ownerrepo string
		want      string
	}{
		{"", "", ""},
		{"owner/repo", "", "owner/repo"},
		{"", "owner/repo", "owner/repo"},
		{"owner/repoenv", "owner/repo", "owner/repo"},
	}
	for _, tt := range tests {
		t.Setenv("GITHUB_REPOSITORY", tt.envrepo)
		r, err := New(tt.ownerrepo)
		if err != nil {
			t.Error(err)
			continue
		}
		got := r.Repository
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestMeasureCoverage(t *testing.T) {
	tests := []struct {
		paths   []string
		want    int
		wantErr bool
	}{
		{
			[]string{
				filepath.Join(coverageTestdataDir(t), "gocover"),
			},
			1,
			false,
		},
		{
			[]string{
				filepath.Join(coverageTestdataDir(t), "gocover"),
				filepath.Join(coverageTestdataDir(t), "lcov"),
			},
			2,
			false,
		},
		{
			[]string{
				filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report.json"),
			},
			1,
			false,
		},
		{
			// Read only one report.json
			[]string{
				filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report.json"),
				filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json"),
			},
			0,
			true,
		},
	}
	for _, tt := range tests {
		r := &Report{}
		if err := r.MeasureCoverage(tt.paths); err != nil {
			if !tt.wantErr {
				t.Error(err)
			}
			continue
		}
		if tt.wantErr {
			t.Error("want error")
		}
		got := len(r.covPaths)
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestCountMeasured(t *testing.T) {
	tet := 1000.0
	tests := []struct {
		r    *Report
		want int
	}{
		{&Report{}, 0},
		{&Report{Coverage: &coverage.Coverage{}}, 1},
		{&Report{CodeToTestRatio: &ratio.Ratio{}}, 1},
		{&Report{TestExecutionTime: &tet}, 1},
	}
	for _, tt := range tests {
		got := tt.r.CountMeasured()
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestDeleteBlockCoverages(t *testing.T) {
	tests := []struct {
		path string
	}{
		{
			filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report.json"),
		},
		{
			filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json"),
		},
	}
	for _, tt := range tests {
		r := &Report{}
		if err := r.Load(tt.path); err != nil {
			t.Fatal(err)
		}
		orig := r.String()
		r.Coverage.DeleteBlockCoverages()
		deleted := r.String()
		if len(orig) <= len(deleted) {
			t.Error("DeleteBlockCoverages error")
		}
	}
}

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
		if err := r.Load(tt.path); err != nil {
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

func TestOut(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{
			filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report.json"),
			"            master (896d3c5)  \n------------------------------\n  \x1b[1mCoverage\x1b[0m             68.5%  \n",
		},
		{
			filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json"),
			"                       master (896d3c5)  \n-----------------------------------------\n  \x1b[1mCoverage\x1b[0m                        68.5%  \n  \x1b[1mCode to Test Ratio\x1b[0m              1:0.5  \n  \x1b[1mTest Execution Time\x1b[0m             4m40s  \n",
		},
	}
	for _, tt := range tests {
		r := &Report{}
		if err := r.Load(tt.path); err != nil {
			t.Fatal(err)
		}
		buf := new(bytes.Buffer)
		if err := r.Out(buf); err != nil {
			t.Fatal(err)
		}
		got := buf.String()
		if got != tt.want {
			t.Errorf("got\n%v\n%#v\nwant\n%v\n%#v", got, got, tt.want, tt.want)
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
	if err := r.Load(path); err != nil {
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
	if err := a.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	if err := b.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json")); err != nil {
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

func TestValidate(t *testing.T) {
	tests := []struct {
		r    *Report
		want string
	}{
		{&Report{}, fmt.Sprintf("coverage report '%s' (env %s) is not set", "repository", "GITHUB_REPOSITORY")},
		{&Report{Repository: "owner/repo"}, fmt.Sprintf("coverage report '%s' (env %s) is not set", "ref", "GITHUB_REF")},
		{&Report{Repository: "owner/repo", Ref: "refs/heads/main"}, fmt.Sprintf("coverage report '%s' (env %s) is not set", "commit", "GITHUB_SHA")},
	}
	for _, tt := range tests {
		err := tt.r.Validate()
		if err == nil {
			t.Error("should be error")
			continue
		}
		if got := err.Error(); got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
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

func coverageTestdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(wd), "pkg", "coverage", "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
