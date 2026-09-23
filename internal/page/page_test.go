package page

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/report"
)

func testReport(commit string, covered int) *report.Report {
	return &report.Report{
		Repository: "k1LoW/octocov",
		Ref:        "refs/heads/main",
		Commit:     strings.Repeat(commit, 40),
		Coverage: &coverage.Coverage{
			Type:    coverage.TypeLOC,
			Format:  "Go coverage",
			Total:   10,
			Covered: covered,
			Files: coverage.FileCoverages{
				{
					File:    "report/report.go",
					Total:   6,
					Covered: covered - 2,
					Blocks: coverage.BlockCoverages{
						{Type: coverage.TypeLOC, StartLine: new(1), EndLine: new(2), Count: new(coverage.ExecCount(4))},
						{Type: coverage.TypeLOC, StartLine: new(3), EndLine: new(3), Count: new(coverage.ExecCount(0))},
					},
				},
				{File: "internal/icon.go", Total: 4, Covered: 2},
			},
		},
	}
}

func TestRenderChanges(t *testing.T) {
	names := []string{"report/report.go", "docs/my file.md", "日本語/ファイル.go"}
	var files []*ChangedFile
	for _, n := range names {
		files = append(files, &ChangedFile{
			Filename: n, Status: "modified", Additions: 1,
			Patch: "@@ -1,1 +1,2 @@\n x\n+y\n",
		})
	}
	in := &ChangesInput{
		Report: testReport("a", 7),
		Base:   Base{Report: testReport("b", 5), Label: "main", Aligned: true},
		Files:  files,
	}
	got, err := RenderChanges(t.Context(), "Coverage of k1LoW/octocov#1", in)
	if err != nil {
		t.Fatal(err)
	}
	page := string(got)
	for _, want := range []string{
		"<!doctype html>",
		"<title>Coverage of k1LoW/octocov#1</title>",
		`"octocov:theme"`,
		"Patch coverage",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("the page does not carry %q", want)
		}
	}
	// Every card is reachable by the anchor a link works out from the path alone.
	for _, n := range names {
		if want := `id="` + report.FileAnchor(n) + `"`; !strings.Contains(page, want) {
			t.Errorf("no card carries %s for %q", want, n)
		}
	}
	// A file the pull request did not touch, and whose coverage did not move, is not drawn.
	if strings.Contains(page, "internal/icon.go") {
		t.Error("the page draws a file the pull request did not touch")
	}
}

func TestRenderChangesUnrenderable(t *testing.T) {
	r := testReport("a", 7)
	r.Coverage.Files[0].Blocks[0].StartLine = new(-1)
	in := &ChangesInput{
		Report: r,
		Base:   Base{Report: testReport("b", 5), Label: "main", Aligned: true},
		Files:  []*ChangedFile{{Filename: "report/report.go", Status: "modified", Patch: "@@ -1,1 +1,2 @@\n x\n+y\n"}},
	}
	if _, err := RenderChanges(t.Context(), "", in); !errors.Is(err, ErrUnrenderable) {
		t.Errorf("got %v, want %v", err, ErrUnrenderable)
	}
}

func TestFileAnchorMatchesThePage(t *testing.T) {
	b, err := os.ReadFile("testdata/fileAnchor.vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Vectors []struct {
			Path   string `json:"path"`
			Anchor string `json:"anchor"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(b, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Vectors) == 0 {
		t.Fatal("no vectors")
	}
	for _, v := range fixture.Vectors {
		if got := report.FileAnchor(v.Path); got != v.Anchor {
			t.Errorf("report.FileAnchor(%q) = %q, want %q", v.Path, got, v.Anchor)
		}
	}
}
