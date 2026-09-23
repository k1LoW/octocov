package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

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
