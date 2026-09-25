package report

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/gh"
)

// viewerBaseURL is where a report stored in a GitHub Actions artifact can be browsed.
const viewerBaseURL = "https://octocov.dev"

// Viewer turns the values of the rendered tables into links to the pages that show the
// report behind them. Each kind of value has a link of its own, and a kind a viewer does
// not link is rendered as plain text.
//
// A nil Viewer renders every value as plain text, which lets a caller with nowhere to link
// to pass nil rather than branch at each cell.
type Viewer struct {
	coverage          func(r *Report) (string, error)
	coverageFile      func(r *Report, path string) (string, error)
	codeToTestRatio   func(r *Report) (string, error)
	testExecutionTime func(r *Report) (string, error)
	readsArtifacts    bool
	// err is the first link that could not be worked out. The tables are rendered as
	// strings, so it is kept here for the caller to ask after rendering.
	err error
}

// NewOctocovDevViewer returns the viewer linking to the octocov.dev pages of the report whose
// metadata is stored under metadataName, or nil when the name is empty. A file is addressed
// there by the metadata of the ref its report was stored for, while the report as a whole is
// routed by its ref, which is why Report.ViewerURL answers for it without a Viewer at all.
func NewOctocovDevViewer(metadataName string) *Viewer {
	if metadataName == "" {
		return nil
	}
	return &Viewer{
		coverage: func(r *Report) (string, error) {
			return r.ViewerURL(), nil
		},
		coverageFile: func(r *Report, path string) (string, error) {
			return octocovDevFileURL(metadataName, r, path), nil
		},
		readsArtifacts: true,
	}
}

// NewOctocovDevRefViewer returns the viewer linking a report as a whole to its
// octocov.dev page, which is routed by the ref and so needs no artifact name. Files are not
// linked, since their pages are addressed by one.
func NewOctocovDevRefViewer() *Viewer {
	return &Viewer{
		coverage: func(r *Report) (string, error) {
			return r.ViewerURL(), nil
		},
		readsArtifacts: true,
	}
}

// NewArtifactViewer returns the viewer linking to the page of HTML uploaded as an artifact
// at pageURL, or nil when there is no such page. A file is linked to its card on the page,
// by the anchor the page gives it, which is worked out from the path alone. anchors holds
// the cards the page drew, and a file it drew no card for is linked to the page itself,
// since the renderer stops drawing cards at its own budgets.
func NewArtifactViewer(pageURL string, anchors map[string]bool) *Viewer {
	if pageURL == "" {
		return nil
	}
	return &Viewer{
		coverage: func(r *Report) (string, error) {
			return pageURL, nil
		},
		coverageFile: func(r *Report, path string) (string, error) {
			if path == "" {
				return "", nil
			}
			a := FileAnchor(path)
			if !anchors[a] {
				return pageURL, nil
			}
			return pageURL + "#" + a, nil
		},
	}
}

// NewCustomViewer returns the viewer linking to where the expressions of `viewer.links:`
// say, or nil when there are none. isBase says the viewer renders the compared column.
func NewCustomViewer(links *config.CustomLinks, isBase bool) *Viewer {
	if links == nil {
		return nil
	}
	link := func(key string, r *Report, file map[string]any) (string, error) {
		if r == nil {
			return "", nil
		}
		vars := map[string]any{
			"report": map[string]any{
				"key":     repositoryKey(r),
				"ref":     r.Ref,
				"commit":  r.Commit,
				"is_base": isBase,
			},
		}
		if file != nil {
			vars["file"] = file
		}
		return links.Link(key, vars)
	}
	return &Viewer{
		coverage: func(r *Report) (string, error) {
			return link(config.LinkCoverage, r, nil)
		},
		coverageFile: func(r *Report, path string) (string, error) {
			return link(config.LinkCoverageFile, r, map[string]any{"path": path})
		},
		codeToTestRatio: func(r *Report) (string, error) {
			return link(config.LinkCodeToTestRatio, r, nil)
		},
		testExecutionTime: func(r *Report) (string, error) {
			return link(config.LinkTestExecutionTime, r, nil)
		},
	}
}

// repositoryKey returns the key of the report within its repository, which is the path under
// owner/repo. It is read off the report's own repository rather than off Report.Key, which is
// relative to GITHUB_REPOSITORY, since in the central mode the reports are of other
// repositories than the one running.
func repositoryKey(r *Report) string {
	repo, err := gh.Parse(r.Repository)
	if err != nil {
		return ""
	}
	return repo.Path
}

// ReadsArtifacts reports whether the pages linked to read the report out of the artifacts
// of the repository it describes, so that a report read from anywhere else has no page.
func (v *Viewer) ReadsArtifacts() bool {
	return v != nil && v.readsArtifacts
}

// Err returns the first link that could not be worked out while rendering.
func (v *Viewer) Err() error {
	if v == nil {
		return nil
	}
	return v.err
}

// CoverageURL returns the link of the overall coverage of r.
func (v *Viewer) CoverageURL(r *Report) string {
	if v == nil || v.coverage == nil {
		return ""
	}
	return v.keep(v.coverage(r))
}

// CodeToTestRatioURL returns the link of the code to test ratio of r.
func (v *Viewer) CodeToTestRatioURL(r *Report) string {
	if v == nil || v.codeToTestRatio == nil {
		return ""
	}
	return v.keep(v.codeToTestRatio(r))
}

// TestExecutionTimeURL returns the link of the test execution time of r.
func (v *Viewer) TestExecutionTimeURL(r *Report) string {
	if v == nil || v.testExecutionTime == nil {
		return ""
	}
	return v.keep(v.testExecutionTime(r))
}

// fileURL returns the link of the coverage of the file at path, which is repository relative.
func (v *Viewer) fileURL(r *Report, path string) string {
	if v == nil || v.coverageFile == nil {
		return ""
	}
	return v.keep(v.coverageFile(r, path))
}

func (v *Viewer) keep(u string, err error) string {
	if err != nil {
		if v.err == nil {
			v.err = err
		}
		return ""
	}
	return u
}

// ViewerURL returns the octocov.dev page that browses this report. That page is routed by
// the ref rather than by the artifact name, and the ref key is what says which one, so the
// report of the default branch is the repository itself and every other ref has a page
// beside it. It is empty when the report has no page there at all.
func (r *Report) ViewerURL() string {
	if r == nil {
		return ""
	}
	repo, ok := viewerRepo(r)
	if !ok {
		return ""
	}
	base := fmt.Sprintf("%s/%s/%s", viewerBaseURL, repo.Owner, repo.Repo)
	key := r.RefKey()
	switch {
	case key == "":
		return base
	case strings.HasPrefix(key, "refs/pull/"):
		return fmt.Sprintf("%s/pull/%s", base, strings.TrimPrefix(key, "refs/pull/"))
	case strings.HasPrefix(key, "refs/heads/"):
		return fmt.Sprintf("%s/tree/%s", base, escapePath(strings.TrimPrefix(key, "refs/heads/")))
	default:
		// A tag, which has no page of its own.
		return ""
	}
}

// octocovDevFileURL returns the octocov.dev page of one file's coverage. The pull request
// path carries no per-file page, so the report is named in the query by the artifact holding
// the metadata of its ref, beside the ref itself. The artifact holding the report itself
// shares its name with the reports of every other ref, and its ID is known only once the
// report is stored, which is after the links are written.
func octocovDevFileURL(metadataName string, r *Report, path string) string {
	if r == nil || path == "" {
		return ""
	}
	repo, ok := viewerRepo(r)
	if !ok {
		return ""
	}
	q := url.Values{}
	q.Set("metadata_name", metadataName)
	// The metadata name carries the ref with its separators replaced, which two refs can
	// share, so the page is also told the ref the metadata it opens has to record.
	if ref := r.RunRef(); ref != "" {
		q.Set("ref", ref)
	}
	return fmt.Sprintf("%s/%s/%s/file/%s?%s", viewerBaseURL, repo.Owner, repo.Repo, escapePath(path), q.Encode())
}

// viewerRepo returns the repository the octocov.dev pages are served under, and reports
// whether the run is one they can serve at all. They read the artifacts through the
// github.com API, so a GitHub Enterprise Server run has nothing there for them to show.
func viewerRepo(r *Report) (*gh.Repository, bool) {
	// Trailing slashes trimmed, since the same server can be named with or without one and
	// only a different host means the artifacts are somewhere the pages cannot reach.
	if s := strings.TrimRight(os.Getenv("GITHUB_SERVER_URL"), "/"); s != "" && s != gh.DefaultGithubServerURL {
		return nil, false
	}
	repo, err := gh.Parse(r.Repository)
	if err != nil {
		return nil, false
	}
	return repo, true
}

// FileAnchor returns the id the page uploaded as an artifact gives the card of the file at
// path, which is repository relative: `file-` and the path percent-encoded per RFC 3986,
// keeping the unreserved characters and `/`. It is the rule the page works the id out by,
// written out because neither url.PathEscape nor url.QueryEscape escapes that exact set.
func FileAnchor(path string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	b.WriteString("file-")
	for i := 0; i < len(path); i++ {
		c := path[i]
		switch {
		case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z', '0' <= c && c <= '9',
			c == '-', c == '.', c == '_', c == '~', c == '/':
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hex[c>>4])
			b.WriteByte(hex[c&0x0f])
		}
	}
	return b.String()
}

// escapePath escapes each segment of a file path on its own, so that the separators stay
// separators while a space or a hash inside a name does not end the path.
func escapePath(path string) string {
	segments := strings.Split(path, "/")
	for i, s := range segments {
		segments[i] = url.PathEscape(s)
	}
	return strings.Join(segments, "/")
}

// linkCell wraps a rendered cell in a markdown link, and leaves it as it is when there is
// no page to point at.
func linkCell(cell, u string) string {
	if u == "" {
		return cell
	}
	return fmt.Sprintf("[%s](%s)", cell, u)
}
