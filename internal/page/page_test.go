package page

import (
	"encoding/json"
	"errors"
	"os"
	"regexp"
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
		Report:  testReport("a", 7),
		Aligned: true,
		Base:    Base{Report: testReport("b", 5), Label: "main", Aligned: true},
		Files:   files,
	}
	got, err := RenderChanges(t.Context(), "Coverage of k1LoW/octocov#1", in)
	if err != nil {
		t.Fatal(err)
	}
	page := string(got.HTML)
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
		if !got.Anchors[report.FileAnchor(n)] {
			t.Errorf("the anchors drawn do not name the card of %q", n)
		}
	}
	// A file the pull request did not touch, and whose coverage did not move, is not drawn.
	if strings.Contains(page, "internal/icon.go") {
		t.Error("the page draws a file the pull request did not touch")
	}
	if len(got.Anchors) != len(names) {
		t.Errorf("got %d anchors\nwant %d, one per card", len(got.Anchors), len(names))
	}
}

func TestRenderChangesUnrenderable(t *testing.T) {
	r := testReport("a", 7)
	r.Coverage.Files[0].Blocks[0].StartLine = new(-1)
	in := &ChangesInput{
		Report:  r,
		Aligned: true,
		Base:    Base{Report: testReport("b", 5), Label: "main", Aligned: true},
		Files:   []*ChangedFile{{Filename: "report/report.go", Status: "modified", Patch: "@@ -1,1 +1,2 @@\n x\n+y\n"}},
	}
	if _, err := RenderChanges(t.Context(), "", in); !errors.Is(err, ErrUnrenderable) {
		t.Errorf("got %v, want %v", err, ErrUnrenderable)
	}
}

func TestRenderChangesWithoutTheHeadGutter(t *testing.T) {
	// Where the patches could not be numbered like the report, the report's lines are not read
	// at all, so a block the page would refuse does not stop it either.
	r := testReport("a", 7)
	r.Coverage.Files[0].Blocks[0].StartLine = new(-1)
	in := &ChangesInput{
		Report:  r,
		Aligned: false,
		Base:    Base{Report: testReport("b", 5), Label: "main", Aligned: true},
		Files:   []*ChangedFile{{Filename: "report/report.go", Status: "modified", Patch: "@@ -1,1 +1,2 @@\n x\n+y\n"}},
	}
	got, err := RenderChanges(t.Context(), "", in)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Anchors[report.FileAnchor("report/report.go")] {
		t.Error("the card of the changed file is not drawn")
	}
}

func TestRenderChangesLinksTheServerOfTheRepository(t *testing.T) {
	// A GitHub Enterprise Server run links the commits to its own server, not to github.com.
	in := &ChangesInput{
		Report:    testReport("a", 7),
		Aligned:   true,
		ServerURL: "https://github.example.com",
		Base:      Base{Report: testReport("b", 5), Label: "main", Aligned: true},
		Files:     []*ChangedFile{{Filename: "report/report.go", Status: "modified", Patch: "@@ -1,1 +1,2 @@\n x\n+y\n"}},
	}
	got, err := RenderChanges(t.Context(), "", in)
	if err != nil {
		t.Fatal(err)
	}
	page := string(got.HTML)
	if want := `href="https://github.example.com/k1LoW/octocov/commit/` + strings.Repeat("a", 40) + `"`; !strings.Contains(page, want) {
		t.Errorf("the page does not carry %s", want)
	}
	if strings.Contains(page, `href="https://github.com/`) {
		t.Error("the page links to github.com")
	}
}

func TestRenderChangesInSyntaxColours(t *testing.T) {
	in := &ChangesInput{
		Report:  testReport("a", 7),
		Aligned: true,
		Base:    Base{Report: testReport("b", 5), Label: "main", Aligned: true},
		Files: []*ChangedFile{
			{Filename: "report/report.go", Status: "modified", Additions: 1, Deletions: 1, Patch: "@@ -1,2 +1,2 @@\n package report\n-func old() {}\n+func a() {}\n"},
			{Filename: "notes.txt", Status: "modified", Additions: 1, Patch: "@@ -1,1 +1,2 @@\n package notes\n+more\n"},
		},
	}
	got, err := RenderChanges(t.Context(), "", in)
	if err != nil {
		t.Fatal(err)
	}
	page := string(got.HTML)
	// `package` in github-light's keyword colour, with github-dark's beside it for a reader
	// whose scheme is dark
	if want := "--code-light:#D73A49;--code-dark:#F97583"; !strings.Contains(page, want) {
		t.Errorf("the Go card is not coloured: the page does not carry %s", want)
	}
	// Both sides of the patch, since the deleted line is only on the old one
	if !regexp.MustCompile(`--code-light:[^"]*">old<`).MatchString(page) {
		t.Error("the old side of the patch is not coloured")
	}
	// A file in no language the page carries a grammar for is drawn plain
	card := page[strings.Index(page, `id="`+report.FileAnchor("notes.txt")+`"`):]
	if strings.Contains(card, "--code-light") {
		t.Error("a text file is coloured")
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
