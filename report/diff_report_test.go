package report

import (
	"bytes"
	"os"
	"path/filepath"
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

	got := a.Compare(b).Table()
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
	}, "")
	f := "diff_file_coverages_table"
	if os.Getenv("UPDATE_GOLDEN") != "" {
		golden.Update(t, testdataDir(t), f, got)
		return
	}
	if diff := golden.Diff(t, testdataDir(t), f, got); diff != "" {
		t.Error(diff)
	}
}
