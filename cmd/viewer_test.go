package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
