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

// maxSourceBytes is the largest file whose text is handed to the page. The page draws no
// larger one either, so reading it would only be memory spent on nothing.
const maxSourceBytes = 1024 * 1024

// resolveViewers returns the viewers the tables of the comment, the job summary and the
// pull request body link through. cur links the report of this run and prev the one it is
// compared against. A failure to work out where to link is not a reason to hold the
// report back, so it is said on stderr and the values are left unlinked.
func resolveViewers(ctx context.Context, stderr io.Writer, c *config.Config, r, rPrev *report.Report, comparedArtifact string, files func() ([]*gh.PullRequestFile, error)) (cur, prev *report.Viewer) {
	switch c.ResolveViewer(ctx) {
	case config.ViewerOctocovDev:
		return viewersFor(storedArtifactViewer(c, r), comparedArtifact)
	case config.ViewerArtifact:
		u, err := uploadChangesPage(ctx, c, r, rPrev, files)
		if err != nil {
			fmt.Fprintf(stderr, "Skip linking to the page of the report: %v\n", err) //nostyle:handlerrors
			return nil, nil
		}
		// The page shows the compared report beside this one, so the compared column has
		// nowhere of its own to link to.
		return report.NewArtifactViewer(u), nil
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
// artifact and returns the URL it opens at.
//
// It is uploaded before the comment, the job summary and the body are written, since the
// URL carries the artifact ID, which is known only once the upload is finalized.
func uploadChangesPage(ctx context.Context, c *config.Config, r, rPrev *report.Report, fetchFiles func() ([]*gh.PullRequestFile, error)) (string, error) {
	// A branch has no changes to draw, and a page of every file with its source is too
	// large and too slow to render on every push.
	if !strings.HasPrefix(r.RefKey(), "refs/pull/") || r.PullRequest == 0 {
		return "", errors.New("the page is rendered only for a pull request")
	}
	if !r.IsMeasuredCoverage() {
		return "", errors.New("coverage is not measured")
	}
	if rPrev == nil || !rPrev.IsMeasuredCoverage() {
		return "", errors.New("there is no previous report to compare the coverage against")
	}
	repo, err := gh.Parse(os.Getenv("GITHUB_REPOSITORY"))
	if err != nil {
		return "", err
	}
	runID, err := strconv.ParseInt(os.Getenv("GITHUB_RUN_ID"), 10, 64)
	if err != nil {
		return "", fmt.Errorf("failed to parse GITHUB_RUN_ID: %w", err)
	}
	g, err := gh.New()
	if err != nil {
		return "", err
	}
	files, err := fetchFiles()
	if err != nil {
		return "", err
	}
	aligned := false
	if mb, err := g.FetchMergeBase(ctx, repo.Owner, repo.Repo, r.PullRequest); err == nil {
		aligned = mb != "" && mb == rPrev.Commit
	}
	in := &page.ChangesInput{
		Report:   r,
		RootPath: c.GitRoot,
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
	title := fmt.Sprintf("Coverage of %s#%d", r.Repository, r.PullRequest)
	html, err := page.RenderChanges(ctx, title, in)
	if err != nil {
		return "", err
	}

	var datastores []string
	if c.Report != nil {
		datastores = c.Report.Datastores
	}
	name, err := datastore.PageArtifactName(datastores, r)
	if err != nil {
		return "", err
	}
	id, err := g.PutUnarchivedArtifact(ctx, repo.Owner, repo.Repo, runID, name, html)
	if err != nil {
		return "", err
	}
	if err := g.DeleteArtifactsBeforeRun(ctx, repo.Owner, repo.Repo, name, runID); err != nil {
		// Deleting needs actions: write, which a workflow may not grant and a pull request
		// from a fork never has. The page is uploaded by now, so it is linked all the same.
		fmt.Fprintf(os.Stderr, "Skip deleting the previous pages of %s: %v\n", name, err) //nostyle:handlerrors
	}
	return gh.ArtifactURL(repo.Owner, repo.Repo, runID, id), nil
}

func changedFiles(files []*gh.PullRequestFile) []*page.ChangedFile {
	out := make([]*page.ChangedFile, 0, len(files))
	for _, f := range files {
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
		sources[fc.FileCoverageA.File] = string(b)
	}
	return sources
}

// readSource reads the file at path under root, up to maxSourceBytes.
func readSource(root *os.Root, path string) ([]byte, error) {
	f, err := root.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.IsDir() || fi.Size() > maxSourceBytes {
		return nil, fmt.Errorf("%s is not a file to draw", path)
	}
	return io.ReadAll(f)
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
