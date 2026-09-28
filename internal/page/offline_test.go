package page

import (
	"regexp"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

// The attributes a browser loads a resource from, or sends a request to, without the reader
// following a link.
var loadingAttrs = map[string]bool{
	"src":        true,
	"srcset":     true,
	"href":       true,
	"xlink:href": true,
	"action":     true,
	"formaction": true,
	"poster":     true,
	"data":       true,
	"background": true,
	"ping":       true,
	"manifest":   true,
	"codebase":   true,
	"archive":    true,
	"longdesc":   true,
	"lowsrc":     true,
	"dynsrc":     true,
}

var (
	cssURLRe     = regexp.MustCompile(`(?i)url\(\s*['"]?([^'")\s]*)`)
	cssImportRe  = regexp.MustCompile(`(?i)@import`)
	scriptNetRes = []*regexp.Regexp{
		regexp.MustCompile(`\bfetch\s*\(`),
		regexp.MustCompile(`\bXMLHttpRequest\b`),
		regexp.MustCompile(`\bWebSocket\b`),
		regexp.MustCompile(`\bEventSource\b`),
		regexp.MustCompile(`\bsendBeacon\b`),
		regexp.MustCompile(`\bimportScripts\b`),
		regexp.MustCompile(`\bimport\s*\(`),
		regexp.MustCompile(`\bnew\s+Image\b`),
		regexp.MustCompile(`\.src\s*=`),
		regexp.MustCompile(`(?:location|window\.open)\b`),
	}
)

// inPage reports whether a reference resolves within the page itself, which a page opened
// on its own can do without asking anything of the network. An empty reference is not one,
// since it resolves to the document itself, which an <iframe> or a <link> fetches again.
func inPage(ref string) bool {
	ref = strings.TrimSpace(ref)
	return strings.HasPrefix(ref, "#") || strings.HasPrefix(strings.ToLower(ref), "data:")
}

func TestRenderChangesLoadsNothingFromOutside(t *testing.T) {
	// The page is uploaded as an artifact and opened with nothing around it, so what it
	// needs has to be in it, and opening it must tell no one outside that it was opened.
	in := &ChangesInput{
		Report:  testReport("a", 7),
		Aligned: true,
		Base:    Base{Report: testReport("b", 5), Label: "main", Aligned: true},
		Files: []*ChangedFile{
			{Filename: "report/report.go", Status: "modified", Additions: 1, Deletions: 1, Patch: "@@ -1,2 +1,2 @@\n package report\n-func old() {}\n+func a() {}\n"},
			{Filename: "docs/my file.md", Status: "added", Additions: 1, Patch: "@@ -0,0 +1,1 @@\n+![image](https://example.com/a.png)\n"},
			{Filename: "web/index.html", Status: "modified", Additions: 1, Patch: "@@ -1,1 +1,2 @@\n <html>\n+<script src=\"https://example.com/a.js\"></script>\n"},
		},
	}
	got, err := RenderChanges(t.Context(), "Coverage of k1LoW/octocov#1", in)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(strings.NewReader(string(got.HTML)))
	if err != nil {
		t.Fatal(err)
	}

	checkCSS := func(where, css string) {
		t.Helper()
		if cssImportRe.MatchString(css) {
			t.Errorf("%s imports a stylesheet", where)
		}
		for _, m := range cssURLRe.FindAllStringSubmatch(css, -1) {
			if !inPage(m[1]) {
				t.Errorf("%s loads %q", where, m[1])
			}
		}
	}

	var scripts, styles int
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			tag := n.Data
			for _, a := range n.Attr {
				key := a.Key
				if a.Namespace != "" {
					key = a.Namespace + ":" + a.Key
				}
				switch {
				case key == "href" && tag == "a":
					// Followed only when the reader clicks it.
				case loadingAttrs[key] && !inPage(a.Val):
					t.Errorf("<%s %s=%q> loads from outside the page", tag, key, a.Val)
				case key == "http-equiv":
					t.Errorf("<%s http-equiv=%q> can send the reader elsewhere", tag, a.Val)
				case key == "style":
					checkCSS("the style of <"+tag+">", a.Val)
				}
			}
			switch tag {
			case "script":
				scripts++
				var b strings.Builder
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					b.WriteString(c.Data)
				}
				for _, re := range scriptNetRes {
					if re.MatchString(b.String()) {
						t.Errorf("a script reaches the network by %s", re)
					}
				}
			case "style":
				styles++
				for c := n.FirstChild; c != nil; c = c.NextSibling {
					checkCSS("the stylesheet", c.Data)
				}
			case "link", "base", "iframe", "frame", "object", "embed", "form", "portal":
				t.Errorf("the page carries <%s>", tag)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	// Without these the checks above would pass on a page that lost them rather than on
	// one that carries them inline.
	if scripts == 0 {
		t.Error("the page carries no script, so the theme script was not checked")
	}
	if styles == 0 {
		t.Error("the page carries no stylesheet, so its rules were not checked")
	}
}
