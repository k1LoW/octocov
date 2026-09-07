package report

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k1LoW/octocov/gh"
	"github.com/tenntenn/golden"
)

func TestDiffOut(t *testing.T) {
	a := &Report{}
	// 896d3c5
	if err := a.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	// 5d1e926
	if err := b.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "awspec", "report.json")); err != nil {
		t.Fatal(err)
	}
	got := new(bytes.Buffer)
	a.Compare(b).Out(got)
	f := "diff_out"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		golden.Update(t, testdataDir(t), f, got)
		return
	}
	if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
		t.Error(diff)
	}
}

func TestDiffTable(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "k1LoW/octocov")
	a := &Report{}
	if err := a.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	if err := b.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "awspec", "report.json")); err != nil {
		t.Fatal(err)
	}

	got := a.Compare(b).Table(nil)
	f := "diff_table"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		golden.Update(t, testdataDir(t), f, got)
		return
	}
	if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
		t.Error(diff)
	}
}

func TestDiffFileCoveragesTable(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "k1LoW/octocov")
	a := &Report{}
	if err := a.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "octocov", "report2.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	if err := b.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "octocov", "report1.json")); err != nil {
		t.Fatal(err)
	}

	got := a.Compare(b).FileCoveragesTable([]*gh.PullRequestFile{ //nostyle:funcfmt
		{Filename: "zcase/added.go", BlobURL: "https://github.com/k1LoW/octocov/blob/afterhash/zcase/added.go", Status: "added"},
		{Filename: "zcase/added_test.go", BlobURL: "https://github.com/k1LoW/octocov/blob/afterhash/zcase/added_test.go", Status: "added"},
		{Filename: "zcase/affected_test.go", BlobURL: "https://github.com/k1LoW/octocov/blob/afterhash/zcase/affected.go", Status: "modified"},
		{Filename: "zcase/removed.go", BlobURL: "https://github.com/k1LoW/octocov/blob/beforehash/zcase/removed.go", Status: "removed"},
		{Filename: "zcase/removed_test.go", BlobURL: "https://github.com/k1LoW/octocov/blob/beforehash/zcase/removed_test.go", Status: "removed"},
		{Filename: "zcase/rename_new.go", BlobURL: "https://github.com/k1LoW/octocov/blob/afterhash/zcase/rename_new.go", Status: "renamed"},
	}, "", nil)
	f := "diff_file_coverages_table"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		golden.Update(t, testdataDir(t), f, got)
		return
	}
	if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
		t.Error(diff)
	}
}

func TestDiffFileCoveragesTableWithPatch(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "k1LoW/octocov")
	a := &Report{}
	if err := a.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "octocov", "report2.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	if err := b.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "octocov", "report1.json")); err != nil {
		t.Fatal(err)
	}

	got := a.Compare(b).FileCoveragesTable([]*gh.PullRequestFile{ //nostyle:funcfmt
		{Filename: "zcase/added.go", BlobURL: "https://github.com/k1LoW/octocov/blob/afterhash/zcase/added.go", Status: "added", ChangedLines: []int{1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{Filename: "zcase/added_test.go", BlobURL: "https://github.com/k1LoW/octocov/blob/afterhash/zcase/added_test.go", Status: "added"},
		{Filename: "zcase/affected.go", BlobURL: "https://github.com/k1LoW/octocov/blob/afterhash/zcase/affected.go", Status: "modified", ChangedLines: []int{9, 10, 11}},
		{Filename: "zcase/removed.go", BlobURL: "https://github.com/k1LoW/octocov/blob/beforehash/zcase/removed.go", Status: "removed"},
		{Filename: "zcase/removed_test.go", BlobURL: "https://github.com/k1LoW/octocov/blob/beforehash/zcase/removed_test.go", Status: "removed"},
		{Filename: "zcase/rename_new.go", BlobURL: "https://github.com/k1LoW/octocov/blob/afterhash/zcase/rename_new.go", Status: "renamed", ChangedLines: []int{5, 6, 9}},
	}, "", nil)
	f := "diff_file_coverages_table_with_patch"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		golden.Update(t, testdataDir(t), f, got)
		return
	}
	if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
		t.Error(diff)
	}
}

func TestDiffTableLinksBothCoveragesToTheViewer(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	a := &Report{}
	if err := a.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	if err := b.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json")); err != nil {
		t.Fatal(err)
	}
	a.Repository = "k1LoW/tbls"
	a.Ref = "refs/pull/722/merge"
	a.BaseRef = "refs/heads/main"
	a.PullRequest = 722
	// The compared side is the report of the default branch, which the viewer serves as the
	// repository itself.
	b.Repository = "k1LoW/tbls"
	b.Ref = "refs/heads/main"
	b.BaseRef = "refs/heads/main"

	got := a.Compare(b).Table(NewViewer("octocov-report@refs_pull_722"))
	for _, want := range []string{
		"](https://octocov.dev/k1LoW/tbls/pull/722)",
		"](https://octocov.dev/k1LoW/tbls)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("got\n%v\nwant it to contain\n%v", got, want)
		}
	}
	// The two coverage cells only, since the diff code block is not markdown.
	if n := strings.Count(got, "octocov.dev"); n != 2 {
		t.Errorf("got %d links\nwant 2\n%v", n, got)
	}
	if strings.Contains(a.Compare(b).Table(nil), "octocov.dev") {
		t.Error("a nil viewer must not link to the viewer")
	}
}

func TestDiffFileCoveragesTablePathsAreSlashSeparated(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "k1LoW/octocov")
	a := &Report{}
	if err := a.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "octocov", "report2.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	if err := b.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "octocov", "report1.json")); err != nil {
		t.Fatal(err)
	}
	a.Repository = "k1LoW/octocov"
	a.Commit = "0123456789abcdef"

	// relWd is joined onto the files that the pull request did not report, with
	// filepath.Join, which separates with a backslash on Windows. Both the source link and
	// the viewer link built from the result are URLs.
	got := a.Compare(b).FileCoveragesTable([]*gh.PullRequestFile{ //nostyle:funcfmt
		{Filename: "no-such-file.go", BlobURL: "https://github.com/k1LoW/octocov/blob/0123456789abcdef/no-such-file.go"},
	}, "sub", NewViewer("octocov-report"))
	if got == "" {
		t.Fatal("got an empty table, so nothing was checked")
	}
	if strings.Contains(got, `\`) {
		t.Errorf("got\n%v\nwant no backslash", got)
	}
	if !strings.Contains(got, "sub/") {
		t.Errorf("got\n%v\nwant it to carry the joined prefix", got)
	}
}
