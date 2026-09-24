package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/internal/page"
	"github.com/k1LoW/octocov/report"
)

// maxSourceBytes is the largest file whose text is handed to the page, and
// maxSourcesBytes the most that all of them may hold together. They are the page's own
// ceilings on one card's body and on a page's bodies, so what is past them would only be
// memory spent on text nothing draws.
const (
	maxSourceBytes  = 1024 * 1024
	maxSourcesBytes = 4 * 1024 * 1024
)

// resolveViewers returns the viewers the tables of the comment, the job summary and the
// pull request body link through. cur links the report of this run and prev the one it is
// compared against. A failure to work out where to link is not a reason to hold the
// report back, so it is said on stderr and the values are left unlinked.
func resolveViewers(ctx context.Context, stderr io.Writer, c *config.Config, r, rPrev *report.Report, comparedArtifact string, diff func() (*gh.PullRequestFiles, error)) (cur, prev *report.Viewer) {
	switch c.ResolveViewer(ctx) {
	case config.ViewerOctocovDev:
		return viewersFor(storedArtifactViewer(c, r), comparedArtifact)
	case config.ViewerArtifact:
		u, anchors, err := uploadChangesPage(ctx, c, r, rPrev, diff)
		if err != nil {
			fmt.Fprintf(stderr, "Skip linking to the page of the report: %v\n", err) //nostyle:handlerrors
			return nil, nil
		}
		// The page shows the compared report beside this one, so the compared column has
		// nowhere of its own to link to.
		return report.NewArtifactViewer(u, anchors), nil
	case config.ViewerCustom:
		links, err := c.CustomLinks()
		if err != nil {
			fmt.Fprintf(stderr, "Skip linking the values of the report: %v\n", err) //nostyle:handlerrors
			return nil, nil
		}
		return report.NewCustomViewer(links, false), report.NewCustomViewer(links, true)
	default:
		return nil, nil
	}
}

// resolveBadgeViewer returns the viewer the badges of the central mode link through.
func resolveBadgeViewer(ctx context.Context, stderr io.Writer, c *config.Config) *report.Viewer {
	switch c.ResolveCentralViewer(ctx) {
	case config.ViewerOctocovDev:
		return report.NewOctocovDevRefViewer()
	case config.ViewerCustom:
		links, err := c.CustomLinks()
		if err != nil {
			fmt.Fprintf(stderr, "Skip linking the badges: %v\n", err) //nostyle:handlerrors
			return nil
		}
		return report.NewCustomViewer(links, false)
	default:
		// The page uploaded as an artifact has no URL that stays the same from one run to
		// the next, so a badge has nothing to link to.
		return nil
	}
}

// uploadChangesPage renders the changes page of the pull request, uploads it as an
// artifact and returns the URL it opens at, with the anchors of the file cards drawn on it.
//
// It is uploaded before the comment, the job summary and the body are written, since the
// URL carries the artifact ID, which is known only once the upload is finalized.
func uploadChangesPage(ctx context.Context, c *config.Config, r, rPrev *report.Report, fetchDiff func() (*gh.PullRequestFiles, error)) (string, map[string]bool, error) {
	// A branch has no changes to draw, and a page of every file with its source is too
	// large and too slow to render on every push.
	n, ok := pullRequestNumber(r)
	if !ok {
		return "", nil, errors.New("the page is rendered only for a pull request")
	}
	if !r.IsMeasuredCoverage() {
		return "", nil, errors.New("coverage is not measured")
	}
	if rPrev == nil || !rPrev.IsMeasuredCoverage() {
		return "", nil, errors.New("there is no previous report to compare the coverage against")
	}
	repo, err := gh.Parse(os.Getenv("GITHUB_REPOSITORY"))
	if err != nil {
		return "", nil, err
	}
	runID, err := strconv.ParseInt(os.Getenv("GITHUB_RUN_ID"), 10, 64)
	if err != nil {
		return "", nil, fmt.Errorf("failed to parse GITHUB_RUN_ID: %w", err)
	}
	g, err := gh.New()
	if err != nil {
		return "", nil, err
	}
	d, err := fetchDiff()
	if err != nil {
		return "", nil, err
	}
	files := d.Files
	aligned := baseAligned(ctx, g, repo, n, d, rPrev)
	in := &page.ChangesInput{
		Report:   r,
		RootPath: c.GitRoot,
		// Where the patches could not be numbered like the report, the page draws no head
		// gutter rather than one beside other lines.
		Aligned:   d.Unaligned == "",
		ServerURL: os.Getenv("GITHUB_SERVER_URL"),
		Base: page.Base{
			Report:   rPrev,
			RootPath: c.GitRoot,
			Label:    baseLabel(rPrev),
			Aligned:  aligned,
		},
		Files: changedFiles(files),
	}
	if aligned {
		// The page draws the files whose coverage moved while their code did not only when
		// the base report addresses the same lines, so their text is read only then.
		in.Sources = affectedSources(c.GitRoot, r, rPrev, files)
	}
	title := fmt.Sprintf("Coverage of %s#%d", r.Repository, n)
	p, err := page.RenderChanges(ctx, title, in)
	if err != nil {
		return "", nil, err
	}

	var datastores []string
	if c.Report != nil {
		datastores = c.Report.Datastores
	}
	name, err := datastore.PageArtifactName(datastores, r)
	if err != nil {
		return "", nil, err
	}
	id, err := g.PutUnarchivedArtifact(ctx, repo.Owner, repo.Repo, runID, name, p.HTML)
	if err != nil {
		return "", nil, err
	}
	if err := g.DeleteArtifactsBeforeRun(ctx, repo.Owner, repo.Repo, name, runID); err != nil {
		// Deleting needs actions: write, which a workflow may not grant and a pull request
		// from a fork never has. The page is uploaded by now, so it is linked all the same.
		fmt.Fprintf(os.Stderr, "Skip deleting the previous pages of %s: %v\n", name, err) //nostyle:handlerrors
	}
	return gh.ArtifactURL(repo.Owner, repo.Repo, runID, id), p.Anchors, nil
}

// pullRequestNumber returns the number of the pull request the report is of. It is read
// off the ref key rather than off PullRequest, since the key falls back to the ref of a
// pull request whose number could not be detected, and the report and its page are named
// after that pull request all the same.
func pullRequestNumber(r *report.Report) (int, bool) {
	n, err := strconv.Atoi(strings.TrimPrefix(r.RefKey(), "refs/pull/"))
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

// baseAligned reports whether the base report was taken at the commit the old side of the
// patches is numbered as. That is the first parent of the merge commit where the patches are its
// diff against that parent, and the merge base where they are those of the pull request files
// API. Where some files have one and some the other, no one commit is the old side of all of them.
func baseAligned(ctx context.Context, g *gh.Gh, repo *gh.Repository, n int, d *gh.PullRequestFiles, rPrev *report.Report) bool {
	if d.Parent != "" {
		return d.Unaligned == "" && d.Parent == rPrev.Commit
	}
	mb, err := g.FetchMergeBase(ctx, repo.Owner, repo.Repo, n)
	if err != nil {
		return false
	}
	return mb != "" && mb == rPrev.Commit
}

// changedFiles returns the files the page draws as changed. A file the merge commit changes
// nothing in is left out, since against the tree the coverage was measured on it is not
// changed, and the page draws it as one whose coverage moved where the coverage did.
func changedFiles(files []*gh.PullRequestFile) []*page.ChangedFile {
	out := make([]*page.ChangedFile, 0, len(files))
	for _, f := range files {
		if f.UnchangedByMerge {
			continue
		}
		out = append(out, &page.ChangedFile{
			Filename:         f.Filename,
			PreviousFilename: f.PreviousFilename,
			Status:           f.Status,
			Additions:        f.Additions,
			Deletions:        f.Deletions,
			Patch:            f.Patch,
		})
	}
	return out
}

// baseLabel names the compared report on the page, by its branch where it has one.
func baseLabel(r *report.Report) string {
	if ref := report.NormalizeRef(r.Ref); ref != "" {
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	if len(r.Commit) > 7 {
		return r.Commit[:7]
	}
	return r.Commit
}

// affectedSources reads the text of the files whose coverage moved while the pull request
// did not change them, keyed by the name the report gives them.
func affectedSources(gitRoot string, r, rPrev *report.Report, files []*gh.PullRequestFile) map[string]string {
	if gitRoot == "" {
		return nil
	}
	changed := map[string]bool{}
	for _, f := range files {
		// Not a changed file of the page either, so its text is what it is drawn from.
		if f.UnchangedByMerge {
			continue
		}
		changed[f.Filename] = true
		if f.PreviousFilename != "" {
			changed[f.PreviousFilename] = true
		}
	}
	d := r.Compare(rPrev)
	if d.Coverage == nil {
		return nil
	}
	// The paths come out of a report, which a pull request can write, so they are opened
	// through the checkout rather than joined onto it: a `..` or a symlink pointing out of
	// it would otherwise put a file of the runner into the uploaded page.
	root, err := os.OpenRoot(gitRoot)
	if err != nil {
		return nil
	}
	defer root.Close()
	sources := map[string]string{}
	total := 0
	for _, fc := range d.Coverage.Files {
		if fc.Diff == 0 || fc.FileCoverageA == nil || fc.FileCoverageB == nil {
			continue
		}
		path := fc.FileCoverageA.NormalizedPath
		if path == "" || changed[path] {
			continue
		}
		b, err := readSource(root, filepath.FromSlash(path))
		if err != nil {
			continue
		}
		// Ends the list rather than skipping past the file, the way the page ends its own,
		// so which files get their text does not depend on which of them happen to be small.
		if total+len(b) > maxSourcesBytes {
			break
		}
		total += len(b)
		sources[fc.FileCoverageA.File] = string(b)
	}
	return sources
}

// readSource reads the file at path under root, up to maxSourceBytes.
//
// The size is checked on what is read rather than on what Stat says, since a file can grow
// between the two. Anything but a regular file is refused before it is opened, because
// opening a FIFO for reading waits for a writer that never comes.
func readSource(root *os.Root, path string) ([]byte, error) {
	fi, err := root.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	f, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	// Once more on what was opened, since the path can have been replaced in between.
	if fi, err := f.Stat(); err != nil || !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a regular file", path)
	}
	b, err := io.ReadAll(io.LimitReader(f, maxSourceBytes+1))
	if err != nil {
		return nil, err
	}
	if len(b) > maxSourceBytes {
		return nil, fmt.Errorf("%s is larger than the page draws", path)
	}
	return b, nil
}

// storedArtifactViewer returns the octocov.dev viewer for the artifact this run stores its
// report in, and nil when it stores none. The pages the coverage cells link to read the
// report out of a GitHub Actions artifact, so the links have to be gated on the same
// readiness that decides whether the report is stored at all. Gating on the configured
// datastores alone would keep linking on a run whose report.if: holds the storing back, at
// an artifact nothing ever writes.
func storedArtifactViewer(c *config.Config, r *report.Report) *report.Viewer {
	if err := c.ReportConfigReady(); err != nil {
		return nil
	}
	name, ok := datastore.ArtifactName(c.Report.Datastores, r)
	if !ok {
		return nil
	}
	return report.NewOctocovDevViewer(name)
}

// viewersFor pairs the viewer of the report being described with the one of the report it
// is compared against. The compared side follows the current one, so that a link appears
// only on a run that stores a report in an artifact of its own, and then only when the
// comparison was read out of an artifact too.
func viewersFor(stored *report.Viewer, comparedArtifact string) (cur, prev *report.Viewer) {
	if stored == nil {
		return nil, nil
	}
	return stored, report.NewOctocovDevViewer(comparedArtifact)
}
