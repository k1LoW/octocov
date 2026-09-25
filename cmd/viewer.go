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
//
// cleanup deletes what the links of earlier runs point at and the ones returned here replace,
// and is nil when there is nothing to delete. It is left to the caller, to be called once the
// outputs carrying the new links are written.
func resolveViewers(ctx context.Context, stderr io.Writer, c *config.Config, r, rPrev *report.Report, comparedArtifact string, diff func() (*gh.PullRequestFiles, error)) (cur, prev *report.Viewer, cleanup func()) {
	switch c.ResolveViewer(ctx) {
	case config.ViewerOctocovDev:
		cur, prev := viewersFor(storedArtifactViewer(c, r), comparedArtifact)
		return cur, prev, nil
	case config.ViewerArtifact:
		u, anchors, cleanup, err := uploadChangesPage(ctx, c, r, rPrev, diff)
		if err != nil {
			fmt.Fprintf(stderr, "Skip linking to the page of the report: %v\n", err) //nostyle:handlerrors
			return nil, nil, nil
		}
		// The page shows the compared report beside this one, so the compared column has
		// nowhere of its own to link to.
		return report.NewArtifactViewer(u, anchors), nil, cleanup
	case config.ViewerCustom:
		links, err := c.CustomLinks()
		if err != nil {
			fmt.Fprintf(stderr, "Skip linking the values of the report: %v\n", err) //nostyle:handlerrors
			return nil, nil, nil
		}
		return report.NewCustomViewer(links, false), report.NewCustomViewer(links, true), nil
	default:
		return nil, nil, nil
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
// artifact and returns the URL it opens at, with the anchors of the file cards drawn on it and
// what deletes the pages earlier runs uploaded.
//
// It is uploaded before the comment, the job summary and the body are written, since the
// URL carries the artifact ID, which is known only once the upload is finalized.
func uploadChangesPage(ctx context.Context, c *config.Config, r, rPrev *report.Report, fetchDiff func() (*gh.PullRequestFiles, error)) (string, map[string]bool, func(), error) {
	// A branch has no changes to draw, and a page of every file with its source is too
	// large and too slow to render on every push.
	n, ok := pullRequestNumber(r)
	if !ok {
		return "", nil, nil, errors.New("the page is rendered only for a pull request")
	}
	if !r.IsMeasuredCoverage() {
		return "", nil, nil, errors.New("coverage is not measured")
	}
	if rPrev == nil || !rPrev.IsMeasuredCoverage() {
		return "", nil, nil, errors.New("there is no previous report to compare the coverage against")
	}
	repo, err := gh.Parse(os.Getenv("GITHUB_REPOSITORY"))
	if err != nil {
		return "", nil, nil, err
	}
	runID, err := strconv.ParseInt(os.Getenv("GITHUB_RUN_ID"), 10, 64)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to parse GITHUB_RUN_ID: %w", err)
	}
	g, err := gh.New()
	if err != nil {
		return "", nil, nil, err
	}
	d, err := fetchDiff()
	if err != nil {
		return "", nil, nil, err
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
		return "", nil, nil, err
	}

	var datastores []string
	if c.Report != nil {
		datastores = c.Report.Datastores
	}
	base, err := datastore.PageArtifactBase(datastores, r)
	if err != nil {
		return "", nil, nil, err
	}
	name := pageArtifactName(base, runAttempt())
	id, err := g.PutUnarchivedArtifact(ctx, repo.Owner, repo.Repo, runID, name, p.HTML)
	if err != nil {
		return "", nil, nil, err
	}
	cleanup := func() {
		// The earlier attempts of this run, and the first attempt of each earlier run that has
		// completed, which is the one nearly every run is. One still in progress may yet link
		// to its page. A page an earlier run uploaded on a re-run of its own
		// is named for an attempt nothing here knows, so it is left to its retention.
		if err := errors.Join(
			g.DeleteRunArtifacts(ctx, repo.Owner, repo.Repo, runID, func(n string) bool {
				return n != name && isPageArtifactOf(base, n)
			}),
			g.DeleteCompletedArtifactsBeforeRun(ctx, repo.Owner, repo.Repo, pageArtifactName(base, 1), runID),
		); err != nil {
			// Deleting needs actions: write, which a workflow may not grant and a pull request
			// from a fork never has. The outputs are written by now, so failing the run over
			// the pages it leaves behind would be worse than leaving them.
			fmt.Fprintf(os.Stderr, "Skip deleting the previous pages of %s: %v\n", name, err) //nostyle:handlerrors
		}
	}
	return gh.ArtifactURL(repo.Owner, repo.Repo, runID, id), p.Anchors, cleanup, nil
}

// mayDeleteEarlierPages reports whether the pages earlier runs uploaded can go, which is when no
// output that lasts past this run is left linking to one of them. written says whether every
// output that was attempted got written. A comment or a body that is configured and was not
// attempted, as where its if: does not hold on this run, is left as an earlier run wrote it,
// with its links. One that is not configured may still be there from before it was taken out
// of the config, which left reports whether it is, and is only asked when nothing else decides.
// The job summary is not counted, since each run has one of its own and this run's leaves the
// earlier ones as they are either way.
func mayDeleteEarlierPages(c *config.Config, commentReady, bodyReady error, written bool, left func() bool) bool {
	if !written {
		return false
	}
	if c.Comment != nil && commentReady != nil {
		return false
	}
	if c.Body != nil && bodyReady != nil {
		return false
	}
	if (c.Comment == nil || c.Body == nil) && left() {
		return false
	}
	return true
}

// earlierOutputsLeft reports whether the pull request still carries a comment or a body report
// of r that an earlier run wrote while this config does not write it, answering true where
// that cannot be told, since a page deleted under a link is worse than one left to its
// retention.
func earlierOutputsLeft(ctx context.Context, c *config.Config, r *report.Report) bool {
	n, ok := pullRequestNumber(r)
	if !ok {
		return true
	}
	repo, err := gh.Parse(os.Getenv("GITHUB_REPOSITORY"))
	if err != nil {
		return true
	}
	g, err := gh.New()
	if err != nil {
		return true
	}
	if c.Comment == nil {
		if found, err := g.HasCommentReport(ctx, repo.Owner, repo.Repo, n, r.Key()); err != nil || found {
			return true
		}
	}
	if c.Body == nil {
		if found, err := g.HasBodyReport(ctx, repo.Owner, repo.Repo, n, r.Key()); err != nil || found {
			return true
		}
	}
	return false
}

// pageArtifactName returns the name the page is uploaded as on the given attempt of the run.
// A re-run keeps its run id, and a run cannot hold two artifacts of one name, so each attempt
// uploads under a name of its own rather than deleting the one before it first, which would
// take the link the outputs of that attempt still carry away with it. The first attempt keeps
// the plain name, which is the one nearly every run uploads under.
func pageArtifactName(base string, attempt int) string {
	if attempt <= 1 {
		return base + ".html"
	}
	return fmt.Sprintf("%s@attempt_%d.html", base, attempt)
}

// isPageArtifactOf reports whether name is the name of a page pageArtifactName builds on base,
// on any attempt.
func isPageArtifactOf(base, name string) bool {
	if name == pageArtifactName(base, 1) {
		return true
	}
	rest, ok := strings.CutPrefix(name, base+"@attempt_")
	if !ok {
		return false
	}
	n, ok := strings.CutSuffix(rest, ".html")
	if !ok {
		return false
	}
	attempt, err := strconv.Atoi(n)
	return err == nil && attempt > 1
}

// runAttempt returns which attempt of the workflow run this is, counted from 1.
func runAttempt() int {
	n, err := strconv.Atoi(os.Getenv("GITHUB_RUN_ATTEMPT"))
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// pullRequestNumber returns the number of the pull request the report is of. It is read
// off the ref key rather than off PullRequest, since the key falls back to the ref of a
// pull request whose number could not be detected, and the page is named after that pull
// request all the same.
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
	name, ok := datastore.MetadataArtifactName(c.Report.Datastores, r)
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
