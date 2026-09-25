package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

func TestChangedFilesLeaveOutWhatTheMergeDoesNotChange(t *testing.T) {
	// A file whose patch is empty because the merge changes nothing in it is not a changed
	// file of the tree the coverage was measured on, while one the API sent no patch for is.
	got := changedFiles([]*gh.PullRequestFile{
		{Filename: "a.go", Patch: "@@ -1,1 +1,2 @@\n x\n+y\n"},
		{Filename: "large.go"},
		{Filename: "merged.go", UnchangedByMerge: true},
	})
	var names []string
	for _, f := range got {
		names = append(names, f.Filename)
	}
	if diff := cmp.Diff(names, []string{"a.go", "large.go"}); diff != "" {
		t.Error(diff)
	}
}

func TestMayDeleteEarlierPages(t *testing.T) {
	skipped := errors.New("the condition in the `if` section is not met")
	tests := []struct {
		name         string
		c            *config.Config
		commentReady error
		bodyReady    error
		written      bool
		left         bool
		want         bool
	}{
		{"every output written", &config.Config{Comment: &config.Comment{}, Body: &config.Body{}}, nil, nil, true, false, true},
		{"an output failed", &config.Config{Comment: &config.Comment{}}, nil, nil, false, false, false},
		// The comment an earlier run wrote is left as it was, with its link.
		{"a configured comment was not attempted", &config.Config{Comment: &config.Comment{}}, skipped, nil, true, false, false},
		{"a configured body was not attempted", &config.Config{Body: &config.Body{}}, nil, skipped, true, false, false},
		// Nothing an earlier run wrote is there to keep a link.
		{"no comment or body is left", &config.Config{}, errors.New("comment: is not set"), errors.New("body: is not set"), true, false, true},
		// One taken out of the config is still on the pull request with its link.
		{"a comment is left from before", &config.Config{Body: &config.Body{}}, errors.New("comment: is not set"), nil, true, true, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			left := func() bool { return tt.left }
			if got := mayDeleteEarlierPages(tt.c, tt.commentReady, tt.bodyReady, tt.written, left); got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestPageArtifactName(t *testing.T) {
	const base = "octocov-report@refs_pull_1"
	tests := []struct {
		attempt int
		want    string
	}{
		{1, "octocov-report@refs_pull_1.html"},
		{2, "octocov-report@refs_pull_1@attempt_2.html"},
	}
	for _, tt := range tests {
		got := pageArtifactName(base, tt.attempt)
		if got != tt.want {
			t.Errorf("attempt %d: got %q\nwant %q", tt.attempt, got, tt.want)
		}
		if !isPageArtifactOf(base, got) {
			t.Errorf("%q is not read as a page of %q", got, base)
		}
	}
	// Not a page of this report: another report's, the report itself, and a name that only
	// looks like an attempt.
	for _, name := range []string{
		"octocov-report-sub@refs_pull_1.html",
		"octocov-report@refs_pull_1",
		"octocov-report@refs_pull_10.html",
		"octocov-report@refs_pull_1@attempt_x.html",
		"octocov-report@refs_pull_1@attempt_1.html",
	} {
		if isPageArtifactOf(base, name) {
			t.Errorf("%q is read as a page of %q", name, base)
		}
	}
}

func TestRunAttempt(t *testing.T) {
	for _, tt := range []struct {
		env  string
		want int
	}{{"", 1}, {"3", 3}, {"0", 1}, {"x", 1}} {
		t.Setenv("GITHUB_RUN_ATTEMPT", tt.env)
		if got := runAttempt(); got != tt.want {
			t.Errorf("GITHUB_RUN_ATTEMPT=%q: got %d, want %d", tt.env, got, tt.want)
		}
	}
}

func TestBaseLabel(t *testing.T) {
	tests := []struct {
		name string
		r    *report.Report
		want string
	}{
		{"a branch is named by itself", &report.Report{Ref: "refs/heads/main", Commit: "0123456789abcdef"}, "main"},
		{"a bare branch name", &report.Report{Ref: "main", Commit: "0123456789abcdef"}, "main"},
		{"no ref is named by its commit", &report.Report{Commit: "0123456789abcdef"}, "0123456"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := baseLabel(tt.r); got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestPullRequestNumber(t *testing.T) {
	tests := []struct {
		name   string
		r      *report.Report
		want   int
		wantOK bool
	}{
		{"the detected number", &report.Report{Ref: "refs/pull/1/merge", BaseRef: "refs/heads/main", PullRequest: 1}, 1, true},
		// The number could not be detected, and the ref still says which pull request it is.
		{"the number the ref carries", &report.Report{Ref: "refs/pull/2/merge", BaseRef: "refs/heads/main"}, 2, true},
		{"a branch", &report.Report{Ref: "refs/heads/feat", BaseRef: "refs/heads/main"}, 0, false},
		{"the default branch", &report.Report{Ref: "refs/heads/main", BaseRef: "refs/heads/main"}, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := pullRequestNumber(tt.r)
			if got != tt.want || ok != tt.wantOK {
				t.Errorf("got %v, %v\nwant %v, %v", got, ok, tt.want, tt.wantOK)
			}
		})
	}
}

func TestBaseAlignedOnTheMergeDiff(t *testing.T) {
	// The patches of the merge diff number their old side as the first parent, which the base
	// report has to have been taken at. No API call is made to tell.
	rPrev := &report.Report{Commit: "parent"}
	tests := []struct {
		name string
		d    *gh.PullRequestFiles
		want bool
	}{
		{"taken at the first parent", &gh.PullRequestFiles{Parent: "parent"}, true},
		{"taken at another commit", &gh.PullRequestFiles{Parent: "other"}, false},
		// Some files kept the patches of the pull request, whose old side is the merge base.
		{"some patches are not the merge diff's", &gh.PullRequestFiles{Parent: "parent", Unaligned: "past the limit"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := baseAligned(t.Context(), nil, nil, 1, tt.d, rPrev); got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestAffectedSources(t *testing.T) {
	root := t.TempDir()
	for _, p := range []string{"moved.go", "changed.go", "same.go"} {
		if err := os.WriteFile(filepath.Join(root, p), []byte("package "+p+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	file := func(path string, covered int) *coverage.FileCoverage {
		return &coverage.FileCoverage{File: "github.com/k1LoW/octocov/" + path, NormalizedPath: path, Total: 10, Covered: covered}
	}
	head := &report.Report{Coverage: &coverage.Coverage{Total: 30, Covered: 18, Files: coverage.FileCoverages{file("moved.go", 8), file("changed.go", 5), file("same.go", 5)}}}
	base := &report.Report{Coverage: &coverage.Coverage{Total: 30, Covered: 12, Files: coverage.FileCoverages{file("moved.go", 4), file("changed.go", 3), file("same.go", 5)}}}
	files := []*gh.PullRequestFile{{Filename: "changed.go"}}

	got := affectedSources(root, head, base, files)
	// Only the file whose coverage moved while the pull request did not change it, by the
	// name the report gives it.
	want := map[string]string{"github.com/k1LoW/octocov/moved.go": "package moved.go\n"}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Error(diff)
	}
}

func TestAffectedSourcesStayInTheCheckout(t *testing.T) {
	// A report is something a pull request can write, so a path out of it that leaves the
	// checkout, by `..` or by a symlink, must not put the file it reaches into the page.
	outside := t.TempDir()
	secret := filepath.Join(outside, "secret")
	if err := os.WriteFile(secret, []byte("secret\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.Symlink(secret, filepath.Join(root, "link.go")); err != nil {
		t.Fatal(err)
	}
	rel, err := filepath.Rel(root, secret)
	if err != nil {
		t.Fatal(err)
	}
	file := func(path string, covered int) *coverage.FileCoverage {
		return &coverage.FileCoverage{File: path, NormalizedPath: filepath.ToSlash(path), Total: 10, Covered: covered}
	}
	head := &report.Report{Coverage: &coverage.Coverage{Total: 20, Covered: 16, Files: coverage.FileCoverages{file("link.go", 8), file(rel, 8)}}}
	base := &report.Report{Coverage: &coverage.Coverage{Total: 20, Covered: 8, Files: coverage.FileCoverages{file("link.go", 4), file(rel, 4)}}}

	if got := affectedSources(root, head, base, nil); len(got) != 0 {
		t.Errorf("got %v\nwant nothing read from outside the checkout", got)
	}
}

func TestAffectedSourcesStopAtThePageBudget(t *testing.T) {
	// Reading on past what the page draws would be memory spent on nothing, so the reads
	// stop once the files read hold as much as the page's bodies may.
	root := t.TempDir()
	body := strings.Repeat("x", maxSourceBytes)
	var headFiles, baseFiles coverage.FileCoverages
	n := maxSourcesBytes/maxSourceBytes + 2
	for i := range n {
		p := fmt.Sprintf("f%d.go", i)
		if err := os.WriteFile(filepath.Join(root, p), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		headFiles = append(headFiles, &coverage.FileCoverage{File: p, NormalizedPath: p, Total: 10, Covered: 8})
		baseFiles = append(baseFiles, &coverage.FileCoverage{File: p, NormalizedPath: p, Total: 10, Covered: 4})
	}
	head := &report.Report{Coverage: &coverage.Coverage{Total: 10 * n, Covered: 8 * n, Files: headFiles}}
	base := &report.Report{Coverage: &coverage.Coverage{Total: 10 * n, Covered: 4 * n, Files: baseFiles}}

	got := affectedSources(root, head, base, nil)
	if want := maxSourcesBytes / maxSourceBytes; len(got) != want {
		t.Errorf("got %d sources\nwant %d", len(got), want)
	}
}

func TestResolveViewersFallBackToThePageOnlyByDefault(t *testing.T) {
	tests := []struct {
		name     string
		viewer   *config.Viewer
		wantPage bool
	}{
		{"an unset viewer gives way to the page", nil, true},
		{"an explicit octocov.dev stays unlinked", &config.Viewer{Type: config.ViewerOctocovDev}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// The unset viewer resolves to octocov.dev only for a public repository, which is
			// asked of the API.
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/repos/k1LoW/octocov" {
					http.NotFound(w, r)
					return
				}
				fmt.Fprint(w, `{"private":false}`)
			}))
			t.Cleanup(ts.Close)
			t.Setenv("GH_HOST", "")
			t.Setenv("GITHUB_API_URL", ts.URL)
			t.Setenv("GITHUB_TOKEN", "dummy")
			t.Setenv("GITHUB_SERVER_URL", "https://github.com")
			t.Setenv("GITHUB_REPOSITORY", "k1LoW/octocov")
			// Without an event report.if: cannot be evaluated, so this run stores no report and
			// octocov.dev has none of its own to open.
			t.Setenv("GITHUB_EVENT_NAME", "")
			c := config.New()
			c.Repository = "k1LoW/octocov"
			c.Report = &config.Report{If: "is_default_branch", Datastores: []string{"artifact://k1LoW/octocov"}}
			c.Viewer = tt.viewer
			// A branch, on which the page refuses before reaching the API, so reaching it is
			// told by the reason it gives.
			r := &report.Report{
				Repository: "k1LoW/octocov",
				Ref:        "refs/heads/feat",
				BaseRef:    "refs/heads/main",
				Coverage:   &coverage.Coverage{Total: 100, Covered: 50},
			}
			stderr := &bytes.Buffer{}
			cur, prev, cleanup := resolveViewers(t.Context(), stderr, c, r, nil, "", nil)
			if cur != nil || prev != nil || cleanup != nil {
				t.Errorf("got %v, %v, cleanup %v\nwant no link", cur, prev, cleanup != nil)
			}
			const want = "Skip linking to the page of the report: the page is rendered only for a pull request"
			if got := strings.Contains(stderr.String(), want); got != tt.wantPage {
				t.Errorf("got stderr %q\nwant the page tried: %v", stderr.String(), tt.wantPage)
			}
		})
	}
}
