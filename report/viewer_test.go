package report

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/k1LoW/octocov/config"
)

func TestViewerReportURL(t *testing.T) {
	tests := []struct {
		name         string
		artifactName string
		serverURL    string
		repository   string
		ref          string
		baseRef      string
		pullRequest  int
		want         string
	}{
		{"a pull request has a page of its own", "octocov-report@refs_pull_722", "", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
		{"the default branch is the repository itself", "octocov-report", "", "k1LoW/octocov", "refs/heads/main", "refs/heads/main", 0, "https://octocov.dev/k1LoW/octocov"},
		{"a branch has a tree page", "octocov-report@refs_heads_feat_x", "", "k1LoW/octocov", "refs/heads/feat/x", "refs/heads/main", 0, "https://octocov.dev/k1LoW/octocov/tree/feat/x"},
		{"a branch name is escaped segment by segment", "octocov-report", "", "k1LoW/octocov", "refs/heads/feat/a b", "refs/heads/main", 0, "https://octocov.dev/k1LoW/octocov/tree/feat/a%20b"},
		{"a tag has no page of its own", "octocov-report", "", "k1LoW/octocov", "refs/tags/v1.0.0", "refs/heads/main", 0, ""},
		{"a report predating the base ref is read as the repository", "octocov-report", "", "k1LoW/octocov", "refs/heads/whatever", "", 0, "https://octocov.dev/k1LoW/octocov"},
		{"a report of a sub directory is served under its repository", "octocov-report-sub@refs_pull_722", "", "k1LoW/octocov/sub", "refs/pull/722/merge", "refs/heads/main", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
		{"a report stored in no artifact links nowhere", "", "", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, ""},
		{"a GitHub Enterprise Server run links nowhere", "octocov-report@refs_pull_722", "https://github.example.com", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, ""},
		{"github.com stated explicitly is served", "octocov-report@refs_pull_722", "https://github.com", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
		// The same server either way, so the slash must not read as another host.
		{"github.com with a trailing slash is served", "octocov-report@refs_pull_722", "https://github.com/", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			v := NewOctocovDevViewer(tt.artifactName)
			got := v.CoverageURL(&Report{Repository: tt.repository, Ref: tt.ref, BaseRef: tt.baseRef, PullRequest: tt.pullRequest, Commit: "0123456789abcdef"})
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

// The central index links the badges of the reports it collected, and it holds no artifact
// name for any of them, so the page of a report has to be reachable without one.
func TestReportViewerURL(t *testing.T) {
	tests := []struct {
		name       string
		serverURL  string
		repository string
		ref        string
		baseRef    string
		want       string
	}{
		{"a collected report links at its repository", "", "k1LoW/tbls", "refs/heads/main", "refs/heads/main", "https://octocov.dev/k1LoW/tbls"},
		{"a report carrying no ref at all links at its repository", "", "k1LoW/tbls", "", "", "https://octocov.dev/k1LoW/tbls"},
		{"a GitHub Enterprise Server run links nowhere", "https://github.example.com", "k1LoW/tbls", "refs/heads/main", "refs/heads/main", ""},
		{"a repository that is not owner/repo links nowhere", "", "tbls", "refs/heads/main", "refs/heads/main", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			r := &Report{Repository: tt.repository, Ref: tt.ref, BaseRef: tt.baseRef}
			if got := r.ViewerURL(); got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestViewerFileURL(t *testing.T) {
	tests := []struct {
		name         string
		artifactName string
		serverURL    string
		path         string
		want         string
	}{
		{"a file is named by the artifact holding its report", "octocov-report@refs_pull_722", "", "report/report.go", "https://octocov.dev/k1LoW/octocov/file/report/report.go?artifact_name=octocov-report%40refs_pull_722"},
		{"the default branch is addressed the same way", "octocov-report", "", "report/report.go", "https://octocov.dev/k1LoW/octocov/file/report/report.go?artifact_name=octocov-report"},
		{"a separator inside a name does not end the path", "octocov-report", "", "some dir/a#b.go", "https://octocov.dev/k1LoW/octocov/file/some%20dir/a%23b.go?artifact_name=octocov-report"},
		{"an empty path links nowhere", "octocov-report", "", "", ""},
		{"a report stored in no artifact links nowhere", "", "", "report/report.go", ""},
		{"a GitHub Enterprise Server run links nowhere", "octocov-report", "https://github.example.com", "report/report.go", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			v := NewOctocovDevViewer(tt.artifactName)
			got := v.fileURL(&Report{Repository: "k1LoW/octocov", Commit: "0123456789abcdef"}, tt.path)
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestLinkCell(t *testing.T) {
	if got, want := linkCell("82.3%", ""), "82.3%"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	if got, want := linkCell("82.3%", "https://octocov.dev/o/r/pull/1"), "[82.3%](https://octocov.dev/o/r/pull/1)"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestArtifactViewer(t *testing.T) {
	const page = "https://github.com/k1LoW/octocov/actions/runs/1/artifacts/2"
	v := NewArtifactViewer(page, map[string]bool{"file-docs/my%20file.md": true})
	r := &Report{Repository: "k1LoW/octocov"}
	if got := v.CoverageURL(r); got != page {
		t.Errorf("got %v\nwant %v", got, page)
	}
	if got, want := v.fileURL(r, "docs/my file.md"), page+"#file-docs/my%20file.md"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	// A file past the page's budgets has no card, so it is linked to the page itself.
	if got := v.fileURL(r, "not/drawn.go"); got != page {
		t.Errorf("got %v\nwant %v", got, page)
	}
	// The page carries the coverage alone.
	if got := v.CodeToTestRatioURL(r); got != "" {
		t.Errorf("got %v\nwant no link", got)
	}
	if got := v.TestExecutionTimeURL(r); got != "" {
		t.Errorf("got %v\nwant no link", got)
	}
	if NewArtifactViewer("", nil) != nil {
		t.Error("no page must be no viewer")
	}
}

func customLinks(t *testing.T, links string) *config.CustomLinks {
	t.Helper()
	c := config.New()
	if err := yaml.Unmarshal([]byte("viewer:\n  type: custom\n  links:\n"+links), c); err != nil {
		t.Fatal(err)
	}
	return c.Viewer.CustomLinks(map[string]any{})
}

func TestCustomViewer(t *testing.T) {
	links := customLinks(t, `    coverage: 'report.is_base ? nil : "https://example.com/" + report.commit'
    coverageFile: '"https://example.com/" + report.commit + "/" + file.path'
    codeToTestRatio: '"https://example.com/ratio/" + report.ref'
    testExecutionTime: '"https://example.com/time/" + report.ref'
`)
	cur := NewCustomViewer(links, false)
	prev := NewCustomViewer(links, true)
	a := &Report{Repository: "k1LoW/octocov", Ref: "refs/pull/1/merge", Commit: "aaa"}
	b := &Report{Repository: "k1LoW/octocov", Ref: "refs/heads/main", Commit: "bbb"}
	tests := []struct {
		name string
		got  string
		want string
	}{
		{"the current coverage", cur.CoverageURL(a), "https://example.com/aaa"},
		{"the compared coverage is left out by is_base", prev.CoverageURL(b), ""},
		{"a file", cur.fileURL(a, "a.go"), "https://example.com/aaa/a.go"},
		// The compared column is evaluated with the base report's ref and commit.
		{"a file of the compared report", prev.fileURL(b, "a.go"), "https://example.com/bbb/a.go"},
		{"the code to test ratio", cur.CodeToTestRatioURL(a), "https://example.com/ratio/refs/pull/1/merge"},
		{"the test execution time", prev.TestExecutionTimeURL(b), "https://example.com/time/refs/heads/main"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("%s: got %v\nwant %v", tt.name, tt.got, tt.want)
		}
	}
	if err := errors.Join(cur.Err(), prev.Err()); err != nil {
		t.Error(err)
	}
}

func TestCustomViewerKey(t *testing.T) {
	// The key is the report's own path under owner/repo, including in the central mode, where
	// the report is of another repository than GITHUB_REPOSITORY.
	t.Setenv("GITHUB_REPOSITORY", "k1LoW/central")
	v := NewCustomViewer(customLinks(t, "    coverage: 'report.key'\n"), false)
	tests := []struct {
		repository string
		want       string
	}{
		{"k1LoW/central", ""},
		{"k1LoW/central/sub/dir", "sub/dir"},
		{"k1LoW/tbls", ""},
		{"k1LoW/monorepo/services/a", "services/a"},
	}
	for _, tt := range tests {
		if got := v.CoverageURL(&Report{Repository: tt.repository}); got != tt.want {
			t.Errorf("%s: got %q\nwant %q", tt.repository, got, tt.want)
		}
	}
	if err := v.Err(); err != nil {
		t.Error(err)
	}
}

func TestCustomViewerErr(t *testing.T) {
	v := NewCustomViewer(customLinks(t, "    coverage: '1'\n"), false)
	if got := v.CoverageURL(&Report{}); got != "" {
		t.Errorf("got %v\nwant no link", got)
	}
	if v.Err() == nil {
		t.Error("a link that is not a string must be an error")
	}
}

func TestTableLinksEveryValueACustomViewerLinks(t *testing.T) {
	r := &Report{}
	if err := r.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json")); err != nil {
		t.Fatal(err)
	}
	if !r.IsMeasuredCodeToTestRatio() || !r.IsMeasuredTestExecutionTime() {
		t.Fatal("the report measures less than this checks")
	}
	v := NewCustomViewer(customLinks(t, `    coverage: '"https://example.com/coverage"'
    codeToTestRatio: '"https://example.com/ratio"'
    testExecutionTime: '"https://example.com/time"'
`), false)
	got := r.Table(v)
	for _, want := range []string{"](https://example.com/coverage)", "](https://example.com/ratio)", "](https://example.com/time)"} {
		if !strings.Contains(got, want) {
			t.Errorf("got\n%v\nwant it to contain\n%v", got, want)
		}
	}
}
