// Package page renders the changes page of a pull request as one page of HTML, with the
// same components octocov.dev draws its pages with.
//
// The components are @octocov/ui, which is JavaScript, so they are bundled into one script
// (see bundle/) and evaluated by SpiderMonkey compiled into Go.
package page

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strings"

	spidermonkey "github.com/goccy/go-spidermonkey"
	"github.com/k1LoW/octocov/report"
)

//go:embed bundle.js
var bundle string

//go:embed ssr.css
var stylesheet string

// maxMemoryBytes is the ceiling of the engine's linear memory. The default of 256 MiB is
// what the octocov.dev package measured running out on a page of octocov's own size, and
// untouched pages never page in, so the headroom costs nothing until it is used.
const maxMemoryBytes = 1024 * 1024 * 1024

// ErrUnrenderable is returned when the reports handed over are not ones the page can be
// drawn from, as opposed to a failure of the renderer itself.
var ErrUnrenderable = errors.New("the report cannot be rendered as a page")

// ChangesInput is what the changes page is drawn from. The field names are the ones the
// JavaScript side reads, and the report is marshaled the way it is stored.
type ChangesInput struct {
	Report   *report.Report `json:"report"`
	RootPath string         `json:"rootPath,omitempty"`
	// Aligned says the report's lines are the new side of the patches. No omitempty, since the
	// page reads an absent value as true, and false is the one that has to reach it.
	Aligned bool           `json:"aligned"`
	Base    Base           `json:"base"`
	Files   []*ChangedFile `json:"files"`
	// Sources holds the text of the files whose coverage moved while their code did not,
	// which carry no patch to draw them from.
	Sources map[string]string `json:"sources,omitempty"`
}

// Base is the report a pull request is compared against.
type Base struct {
	Report   *report.Report `json:"report"`
	RootPath string         `json:"rootPath,omitempty"`
	Label    string         `json:"label"`
	// Aligned says the base report was taken at the commit the diff starts at, which is
	// what lets its coverage be laid over the old side of a hunk.
	Aligned bool `json:"aligned"`
}

// ChangedFile is a file of the pull request, named as the GitHub API names it.
type ChangedFile struct {
	Filename         string `json:"filename"`
	PreviousFilename string `json:"previous_filename,omitempty"`
	Status           string `json:"status"`
	Additions        int    `json:"additions"`
	Deletions        int    `json:"deletions"`
	Patch            string `json:"patch,omitempty"`
}

type answer struct {
	OK      bool   `json:"ok"`
	HTML    string `json:"html"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// Page is a rendered page and the anchors of the file cards drawn on it. The renderer
// stops drawing cards at its own budgets, so a file of the pull request can have none.
type Page struct {
	HTML    []byte
	Anchors map[string]bool
}

// fileIDRe matches the id of a file card, which is `file-` and the percent-encoded path.
var fileIDRe = regexp.MustCompile(`id="(file-[^"]*)"`)

// RenderChanges renders the changes page as a whole HTML document titled title.
func RenderChanges(ctx context.Context, title string, in *ChangesInput) (*Page, error) {
	js, err := spidermonkey.New(spidermonkey.Config{MaxMemoryBytes: maxMemoryBytes})
	if err != nil {
		return nil, err
	}
	defer js.Close()

	if _, err := eval(ctx, js, bundle); err != nil {
		return nil, fmt.Errorf("failed to evaluate the page bundle: %w", err)
	}
	payload, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	// Passed as a string literal, since a string each way is all the bridge carries.
	quoted, err := json.Marshal(string(payload))
	if err != nil {
		return nil, err
	}
	out, err := eval(ctx, js, "renderChangesPageJSON("+string(quoted)+")")
	if err != nil {
		return nil, fmt.Errorf("failed to render the page: %w", err)
	}
	var a answer
	if err := json.Unmarshal([]byte(out), &a); err != nil {
		return nil, fmt.Errorf("failed to read what the page renderer answered: %w", err)
	}
	if !a.OK {
		if a.Kind == "report" {
			return nil, fmt.Errorf("%w: %s", ErrUnrenderable, a.Message)
		}
		return nil, fmt.Errorf("failed to render the page: %s", a.Message)
	}
	// Read off the bundle rather than embedded on its own, so it cannot come from another
	// release of the package than the markup it binds.
	themeScript, err := eval(ctx, js, "THEME_SCRIPT")
	if err != nil {
		return nil, fmt.Errorf("failed to read the theme script: %w", err)
	}
	// Read off the markup rather than worked out from the budgets, so the anchors are the
	// cards that were drawn whatever the release of the package decides to leave out.
	anchors := map[string]bool{}
	for _, m := range fileIDRe.FindAllStringSubmatch(a.HTML, -1) {
		anchors[html.UnescapeString(m[1])] = true
	}
	return &Page{HTML: document(title, a.HTML, themeScript), Anchors: anchors}, nil
}

func eval(ctx context.Context, js *spidermonkey.JS, src string) (string, error) {
	r, err := js.Eval(ctx, src)
	if err != nil {
		return "", err
	}
	if r.Error != nil {
		return "", r.Error
	}
	return r.Value.String(), nil
}

// document wraps the markup the renderer answers with, which is the body alone, in a page
// that stands on its own: an artifact is opened with nothing around it to load a stylesheet
// or a script from.
func document(title, body, themeScript string) []byte {
	var b strings.Builder
	b.WriteString(`<!doctype html><html lang="en"><head><meta charset="utf-8">`)
	b.WriteString(`<meta name="viewport" content="width=device-width, initial-scale=1">`)
	fmt.Fprintf(&b, "<title>%s</title>", html.EscapeString(title))
	fmt.Fprintf(&b, "<style>%s</style>", strings.ReplaceAll(stylesheet, "</style", `<\/style`))
	fmt.Fprintf(&b, "<script>%s</script>", themeScript)
	b.WriteString("</head><body>")
	b.WriteString(body)
	b.WriteString("</body></html>\n")
	return []byte(b.String())
}
