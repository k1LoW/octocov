package report

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/k1LoW/octocov/gh"
)

// viewerBaseURL is where a report stored in a GitHub Actions artifact can be browsed.
const viewerBaseURL = "https://octocov.dev"

// Viewer turns the coverage cells of the rendered tables into links to the pages that
// browse the report behind them. A report is addressed there by the artifact it was stored
// under, so a Viewer is built from that name.
//
// A nil Viewer renders every cell as plain text, which lets a caller with no artifact
// datastore, or one that turned the links off, pass nil rather than branch at each cell.
type Viewer struct {
	artifactName string
}

// NewViewer returns the viewer of the report stored under artifactName, or nil when the
// name is empty.
func NewViewer(artifactName string) *Viewer {
	if artifactName == "" {
		return nil
	}
	return &Viewer{artifactName: artifactName}
}

// reportURL returns the page of the report as a whole. Only a pull request has one, since
// that page is reached by the pull request number rather than by the artifact name.
func (v *Viewer) reportURL(r *Report) string {
	if v == nil || r == nil || r.PullRequest <= 0 {
		return ""
	}
	repo, ok := viewerRepo(r)
	if !ok {
		return ""
	}
	return fmt.Sprintf("%s/%s/%s/pull/%d", viewerBaseURL, repo.Owner, repo.Repo, r.PullRequest)
}

// fileURL returns the page of one file's coverage. The pull request path carries no
// per-file page, so the report is named by its artifact in the query instead.
func (v *Viewer) fileURL(r *Report, path string) string {
	if v == nil || r == nil || r.Commit == "" || path == "" {
		return ""
	}
	repo, ok := viewerRepo(r)
	if !ok {
		return ""
	}
	q := url.Values{}
	q.Set("artifact_name", v.artifactName)
	q.Set("commit", r.Commit)
	return fmt.Sprintf("%s/%s/%s/file/%s?%s", viewerBaseURL, repo.Owner, repo.Repo, escapePath(path), q.Encode())
}

// viewerRepo returns the repository the pages are served under, and reports whether the
// run is one they can serve at all. They read the artifacts through the github.com API, so
// a GitHub Enterprise Server run has nothing there for them to show.
func viewerRepo(r *Report) (*gh.Repository, bool) {
	if s := os.Getenv("GITHUB_SERVER_URL"); s != "" && s != "https://github.com" {
		return nil, false
	}
	repo, err := gh.Parse(r.Repository)
	if err != nil {
		return nil, false
	}
	return repo, true
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
