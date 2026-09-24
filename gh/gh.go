package gh

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	ghttp "github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/google/go-github/v67/github"
	"github.com/k1LoW/go-github-actions/artifact"
	"github.com/k1LoW/go-github-client/v67/factory"
	"github.com/k1LoW/repin"
	"github.com/lestrrat-go/backoff/v2"
	"github.com/shurcooL/githubv4"
	"golang.org/x/oauth2"
)

const DefaultGithubServerURL = "https://github.com"
const maxCopySize = 1073741824 // 1GB

var octocovNameRe = regexp.MustCompile(`(?i)(octocov|coverage)`)

type Gh struct {
	client   *github.Client
	v4Client *githubv4.Client
}

func New() (*Gh, error) {
	client, err := factory.NewGithubClient(factory.Timeout(10 * time.Second))
	if err != nil {
		return nil, err
	}

	token, _, _, v4ep := factory.GetTokenAndEndpoints()
	v4c := githubv4.NewEnterpriseClient(v4ep, oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})))

	return &Gh{
		client:   client,
		v4Client: v4c,
	}, nil
}

func (g *Gh) Client() *github.Client {
	return g.client
}

func (g *Gh) SetClient(client *github.Client) {
	g.client = client
}

func (g *Gh) PushContent(ctx context.Context, owner, repo, branch, content, cp, message string) error {
	srv := g.client.Git
	dRef, _, err := srv.GetRef(ctx, owner, repo, path.Join("heads", branch))
	if err != nil {
		return err
	}

	parent, _, err := srv.GetCommit(ctx, owner, repo, *dRef.Object.SHA)
	if err != nil {
		return err
	}

	var tree *github.Tree

	if cp != "" {
		blob := &github.Blob{
			Content:  new(content),
			Encoding: new("utf-8"),
			Size:     new(len(content)),
		}

		resB, _, err := srv.CreateBlob(ctx, owner, repo, blob)
		if err != nil {
			return err
		}

		entry := &github.TreeEntry{
			Path: new(cp),
			Mode: new("100644"),
			Type: new("blob"),
			SHA:  resB.SHA,
		}

		entries := []*github.TreeEntry{entry}

		tree, _, err = srv.CreateTree(ctx, owner, repo, *dRef.Object.SHA, entries)
		if err != nil {
			return err
		}
	} else {
		tree, _, err = srv.GetTree(ctx, owner, repo, *parent.Tree.SHA, false)
		if err != nil {
			return err
		}
	}

	commit := &github.Commit{
		Message: new(message),
		Tree:    tree,
		Parents: []*github.Commit{parent},
	}
	resC, _, err := srv.CreateCommit(ctx, owner, repo, commit, &github.CreateCommitOptions{})
	if err != nil {
		return err
	}

	nref := &github.Reference{
		Ref: new(path.Join("refs", "heads", branch)),
		Object: &github.GitObject{
			Type: new("commit"),
			SHA:  resC.SHA,
		},
	}
	if _, _, err := srv.UpdateRef(ctx, owner, repo, nref, false); err != nil {
		return err
	}

	return nil
}

func (g *Gh) FetchDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	r, _, err := g.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	return r.GetDefaultBranch(), nil
}

func (g *Gh) FetchRawRootURL(ctx context.Context, owner, repo string) (string, error) {
	b, err := g.FetchDefaultBranch(ctx, owner, repo)
	if err != nil {
		return "", err
	}

	if os.Getenv("GITHUB_SERVER_URL") != "" && os.Getenv("GITHUB_SERVER_URL") != DefaultGithubServerURL {
		// GitHub Enterprise Server
		return fmt.Sprintf("%s/%s/%s/raw/%s", os.Getenv("GITHUB_SERVER_URL"), owner, repo, b), nil
	}

	baseRef := fmt.Sprintf("refs/heads/%s", b)
	ref, _, err := g.client.Git.GetRef(ctx, owner, repo, baseRef)
	if err != nil {
		return "", err
	}
	tree, _, err := g.client.Git.GetTree(ctx, owner, repo, ref.GetObject().GetSHA(), false)
	if err != nil {
		return "", err
	}
	for _, e := range tree.Entries {
		if e.GetType() != "blob" {
			continue
		}
		path := e.GetPath()
		fc, _, _, err := g.client.Repositories.GetContents(ctx, owner, repo, path, &github.RepositoryContentGetOptions{})
		if err != nil {
			return "", err
		}
		return trimContentURL(fc.GetDownloadURL(), path)
	}
	return "", fmt.Errorf("not found files. please commit file to root directory and push: %s/%s", owner, repo)
}

func (g *Gh) DetectCurrentJobID(ctx context.Context, owner, repo string) (int64, error) {
	if os.Getenv("GITHUB_RUN_ID") == "" {
		return 0, fmt.Errorf("env %s is not set", "GITHUB_RUN_ID")
	}
	runID, err := strconv.ParseInt(os.Getenv("GITHUB_RUN_ID"), 10, 64)
	if err != nil {
		return 0, err
	}

	// Although it would be nice if we could get the job_id from an environment variable,
	// there is no way to get it at this time, so it uses a heuristic.
	p := backoff.Exponential( //nostyle:funcfmt
		backoff.WithMinInterval(time.Second),
		backoff.WithMaxInterval(30*time.Second),
		backoff.WithJitterFactor(0.05),
		backoff.WithMaxRetries(5),
	)
	b := p.Start(ctx)
	for backoff.Continue(b) {
		jobs, err := g.listWorkflowJobs(ctx, owner, repo, runID)
		if err != nil {
			return 0, err
		}
		if len(jobs) == 1 {
			return jobs[0].GetID(), nil
		}
		for _, j := range jobs {
			if j.GetName() == os.Getenv("GITHUB_JOB") {
				return j.GetID(), nil
			}
			for _, s := range j.Steps {
				if s.StartedAt != nil && s.CompletedAt == nil && octocovNameRe.MatchString(s.GetName()) {
					return j.GetID(), nil
				}
			}
		}
	}

	return 0, errors.New("could not detect id of current job")
}

func (g *Gh) DetectCurrentBranch(ctx context.Context) (string, error) {
	// GITHUB_HEAD_REF is set only for the pull request events, and on pull_request_target
	// GITHUB_REF names the base branch rather than the merge ref, so reading GITHUB_REF
	// first would report the base branch as the branch being built.
	if h := os.Getenv("GITHUB_HEAD_REF"); h != "" {
		return h, nil
	}
	splitted := strings.SplitN(os.Getenv("GITHUB_REF"), "/", 3) // refs/pull/8/head or refs/heads/branch/branch/name
	if len(splitted) < 3 {
		return "", fmt.Errorf("env %s is not set", "GITHUB_REF")
	}
	if strings.Contains(os.Getenv("GITHUB_REF"), "refs/heads/") {
		return splitted[2], nil
	}
	return "", fmt.Errorf("env %s is not set", "GITHUB_HEAD_REF")
}

// DetectCurrentBaseRef returns the ref the current run is to be compared against, which
// is the base branch of the pull request when the run is on one, and the default branch
// otherwise.
func (g *Gh) DetectCurrentBaseRef(ctx context.Context, owner, repo string) (string, error) {
	if b := os.Getenv("GITHUB_BASE_REF"); b != "" {
		return fmt.Sprintf("refs/heads/%s", b), nil
	}
	b, err := g.FetchDefaultBranch(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("refs/heads/%s", b), nil
}

// ErrNotPullRequest marks the failures of DetectCurrentPullRequestNumber that only mean the run
// is not against a pull request, as opposed to a lookup that could not be completed. Callers that
// fall back to comparing against the default branch use it to tell the ordinary case from one
// worth reporting, because the two measure different sets of changed lines.
var ErrNotPullRequest = errors.New("not running against a pull request")

func (g *Gh) DetectCurrentPullRequestNumber(ctx context.Context, owner, repo string) (int, error) {
	if os.Getenv("GITHUB_PULL_REQUEST_NUMBER") != "" {
		return strconv.Atoi(os.Getenv("GITHUB_PULL_REQUEST_NUMBER"))
	}
	splitted := strings.Split(os.Getenv("GITHUB_REF"), "/") // refs/pull/8/head or refs/heads/branch/branch/name
	if len(splitted) < 3 {
		// Reached both when the variable is unset and when it holds something that is not a
		// ref path, so report what was seen rather than asserting which of the two it was.
		return 0, fmt.Errorf("env %s does not hold a ref path (%q): %w", "GITHUB_REF", os.Getenv("GITHUB_REF"), ErrNotPullRequest)
	}
	if strings.Contains(os.Getenv("GITHUB_REF"), "refs/pull/") {
		prNumber := splitted[2]
		return strconv.Atoi(prNumber)
	}
	// Not GITHUB_REF, which on pull_request_target holds the base branch and would send
	// the search after whichever pull request has the base branch as its head.
	b, err := g.DetectCurrentBranch(ctx)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrNotPullRequest, err)
	}
	l, _, err := g.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		State: "open",
	})
	if err != nil {
		return 0, err
	}
	var d *github.PullRequest
	for _, pr := range l {
		if pr.GetHead().GetRef() == b && isSameRepo(pr.GetHead().GetRepo(), owner, repo) {
			if d != nil {
				return 0, errors.New("more than one open pull request has the current branch as its head")
			}
			d = pr
		}
	}
	if d != nil {
		return d.GetNumber(), nil
	}
	return 0, ErrNotPullRequest
}

// isSameRepo checks if the PR head repository matches the target repository.
// For fork PRs, pr.GetHead().GetRepo() returns the fork repository.
func isSameRepo(headRepo *github.Repository, owner, repo string) bool {
	if headRepo == nil {
		return false
	}
	return strings.EqualFold(headRepo.GetOwner().GetLogin(), owner) && headRepo.GetName() == repo
}

func (g *Gh) ReplaceInsertToBody(ctx context.Context, owner, repo string, number int, content, key string) error {
	sig := generateSig(key)
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return err
	}
	rep, err := insertToBody(pr.GetBody(), content, sig)
	if err != nil {
		return err
	}
	if _, _, err := g.client.PullRequests.Edit(ctx, owner, repo, number, &github.PullRequest{
		Body: &rep,
	}); err != nil {
		return err
	}
	return nil
}

type PullRequest struct {
	Number  int
	IsDraft bool
	Labels  []string
}

func (g *Gh) FetchPullRequest(ctx context.Context, owner, repo string, number int) (*PullRequest, error) {
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	var labels []string
	for _, l := range pr.Labels {
		labels = append(labels, l.GetName())
	}
	return &PullRequest{
		Number:  pr.GetNumber(),
		IsDraft: pr.GetDraft(),
		Labels:  labels,
	}, nil
}

type PullRequestFile struct {
	Filename         string
	PreviousFilename string
	BlobURL          string
	Status           string
	Additions        int
	Deletions        int
	Patch            string
	ChangedLines     []int // line numbers added or modified in the file (new-file line numbers)
	// UnchangedByMerge says the merge commit changes nothing in the file against its first
	// parent, as when the base branch already carries the same change. Patch is then empty for
	// that reason rather than because the API left a large or binary file's patch out.
	UnchangedByMerge bool
}

// ChangedLinesByFile converts a list of PullRequestFile into a map of filename to changed line numbers.
func ChangedLinesByFile(files []*PullRequestFile) map[string][]int {
	if len(files) == 0 {
		// Distinguish "the changed files could not be fetched at all" from "they were fetched,
		// but none of the changed lines are instrumented", so that the two can be reported
		// differently.
		return nil
	}
	m := make(map[string][]int, len(files))
	for _, f := range files {
		if len(f.ChangedLines) == 0 {
			continue
		}
		m[f.Filename] = f.ChangedLines
	}
	return m
}

// PullRequestFiles is the files a pull request changes, and which commits the two sides of
// their patches are numbered as.
type PullRequestFiles struct {
	Files []*PullRequestFile
	// Parent is the first parent of the merge commit when the patches are its diff against
	// that parent, which is then the commit their old side is numbered as. It is empty where
	// the patches are those of the pull request files API, whose old side is the merge base.
	Parent string
	// Unaligned says why some of the changed lines are left numbered as the pull request
	// head's rather than as the commit the coverage was measured on, and is empty otherwise.
	Unaligned string
}

// FetchPullRequestFiles returns the files a pull request changes, with ChangedLines and Patch
// numbered as the lines of commit, the commit the coverage was measured on. When commit is the
// merge commit GitHub creates for the pull request, they are taken from the diff of that commit
// against its first parent, since the patches of the pull request files API number the lines of
// the pull request head, which differ from those of the merge commit wherever the base branch has
// changed a file above them since the pull request branched.
func (g *Gh) FetchPullRequestFiles(ctx context.Context, owner, repo string, number int, commit string) (*PullRequestFiles, error) {
	files, err := g.listPullRequestFiles(ctx, owner, repo, number)
	if err != nil {
		return nil, err
	}
	out := &PullRequestFiles{Files: files}
	if commit == "" {
		return out, nil
	}
	// A failed lookup below leaves the head's lines rather than failing, because the whole file
	// coverage table and the comment would go down with it over lines that are only wrong for
	// the files the base branch has also changed.
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		out.Unaligned = fmt.Sprintf("could not look up the head of pull request #%d: %v", number, err)
		return out, nil
	}
	head := pr.GetHead().GetSHA()
	if commit == head {
		// The coverage was measured on the head, whose line numbers the patches already carry.
		return out, nil
	}
	merged, parent, ok, err := g.fetchMergeCommitFiles(ctx, owner, repo, commit, head)
	if err != nil {
		out.Unaligned = fmt.Sprintf("could not diff commit %s against its first parent: %v", commit, err)
		return out, nil
	}
	if !ok {
		out.Unaligned = fmt.Sprintf("commit %s is neither the head of pull request #%d nor its merge commit", commit, number)
		return out, nil
	}
	out.Parent = parent
	if n := alignChangedLines(files, merged); n > 0 {
		out.Unaligned = fmt.Sprintf("the diff of merge commit %s stops at %d files and leaves out %d of the changed files", commit, compareFilesLimit, n)
	}
	return out, nil
}

// compareFilesLimit is the most files the compare API returns for one comparison.
const compareFilesLimit = 300

// alignChangedLines replaces the ChangedLines, the Patch and the rest of what describes the
// change of each of files with those of merged, the diff of the merge commit against its first
// parent. The pull
// request files API still decides which files there are, because it returns up to 3000 files
// where the compare API stops at compareFilesLimit. It returns how many files were left with the
// lines of the pull request head because merged stopped at that limit.
func alignChangedLines(files []*PullRequestFile, merged []*github.CommitFile) int {
	byName := make(map[string]*github.CommitFile, len(merged))
	for _, f := range merged {
		byName[f.GetFilename()] = f
	}
	truncated := len(merged) >= compareFilesLimit
	kept := 0
	for _, f := range files {
		if m, ok := byName[f.Filename]; ok {
			// Everything the diff says about the file together with its patch, since the page
			// draws them side by side and reads the base report under the previous name.
			f.PreviousFilename = m.GetPreviousFilename()
			f.Status = m.GetStatus()
			f.Additions = m.GetAdditions()
			f.Deletions = m.GetDeletions()
			f.Patch = m.GetPatch()
			f.ChangedLines = parseChangedLinesFromPatch(f.Patch)
			continue
		}
		if truncated {
			// The file may be past the limit rather than unchanged by the merge. Its lines are
			// the ones of the pull request head, which is what every file got before, rather
			// than none, which would drop it from patch coverage altogether.
			kept++
			continue
		}
		// The merge changes nothing in the file, as when the base branch already carries the
		// same change.
		f.Patch = ""
		f.ChangedLines = nil
		f.UnchangedByMerge = true
	}
	return kept
}

func (g *Gh) FetchChangedFiles(ctx context.Context, owner, repo string) ([]*PullRequestFile, error) {
	base, err := g.FetchDefaultBranch(ctx, owner, repo)
	if err != nil {
		return nil, err
	}
	head, err := g.DetectCurrentBranch(ctx)
	if err != nil {
		return nil, err
	}
	compare, _, err := g.client.Repositories.CompareCommits(ctx, owner, repo, base, head, &github.ListOptions{})
	if err != nil {
		return nil, err
	}
	var files []*PullRequestFile
	for _, f := range compare.Files {
		files = append(files, &PullRequestFile{
			Filename:         f.GetFilename(),
			PreviousFilename: f.GetPreviousFilename(),
			BlobURL:          f.GetBlobURL(),
			Status:           f.GetStatus(),
			Additions:        f.GetAdditions(),
			Deletions:        f.GetDeletions(),
			Patch:            f.GetPatch(),
			ChangedLines:     parseChangedLinesFromPatch(f.GetPatch()),
		})
	}
	return files, nil
}

var patchHunkHeaderRe = regexp.MustCompile(`^@@ -\d+(?:,\d+)? \+(\d+)(?:,\d+)? @@`)

// parseChangedLinesFromPatch parses a unified diff hunk (as returned by the
// GitHub API for a pull request file) and returns the line numbers, in the
// new version of the file, that were added or modified.
func parseChangedLinesFromPatch(patch string) []int {
	var lines []int
	newLine := 0
	for l := range strings.SplitSeq(patch, "\n") {
		if m := patchHunkHeaderRe.FindStringSubmatch(l); len(m) > 1 {
			n, err := strconv.Atoi(m[1])
			if err != nil {
				// The start line is out of range: skip the lines of this hunk.
				newLine = 0
				continue
			}
			newLine = n
			continue
		}
		if newLine == 0 {
			continue
		}
		switch {
		case strings.HasPrefix(l, "+"):
			lines = append(lines, newLine)
			newLine++
		case strings.HasPrefix(l, "-"):
			// Removed line: does not exist in the new file.
		case strings.HasPrefix(l, `\`):
			// e.g. "\ No newline at end of file": not an actual line.
		default:
			// Context line: exists in both old and new files.
			newLine++
		}
	}
	return lines
}

func (g *Gh) FetchStepExecutionTimeByTime(ctx context.Context, owner, repo string, jobID int64, t time.Time) (time.Duration, error) {
	p := backoff.Exponential( //nostyle:funcfmt
		backoff.WithMinInterval(time.Second),
		backoff.WithMaxInterval(30*time.Second),
		backoff.WithJitterFactor(0.05),
		backoff.WithMaxRetries(5),
	)
	b := p.Start(ctx)
	log.Printf("target time: %v", t)
	for backoff.Continue(b) {
		job, _, err := g.client.Actions.GetWorkflowJobByID(ctx, owner, repo, jobID)
		if err != nil {
			return 0, err
		}
		l := len(job.Steps)
		for i, s := range job.Steps {
			log.Printf("job step [%d/%d]: %s %v-%v", i+1, l, s.GetName(), s.StartedAt, s.CompletedAt)
			if s.StartedAt == nil || s.CompletedAt == nil {
				continue
			}
			// Truncate less than a second
			startTime := s.GetStartedAt().Time
			completeTime := s.GetCompletedAt().Time
			if startTime.Unix() < t.Unix() && t.Unix() <= completeTime.Unix() {
				log.Print("detect step")
				return completeTime.Sub(startTime), nil
			}
		}
	}
	return 0, fmt.Errorf("the step that was executed at the relevant time (%v) does not exist in the job (%d)", t, jobID)
}

func (g *Gh) FetchStepByTime(ctx context.Context, owner, repo string, jobID int64, t time.Time) (Step, error) {
	p := backoff.Exponential( //nostyle:funcfmt
		backoff.WithMinInterval(time.Second),
		backoff.WithMaxInterval(30*time.Second),
		backoff.WithJitterFactor(0.05),
		backoff.WithMaxRetries(5),
	)
	b := p.Start(ctx)
	log.Printf("target time: %v", t)
	for backoff.Continue(b) {
		job, _, err := g.client.Actions.GetWorkflowJobByID(ctx, owner, repo, jobID)
		if err != nil {
			return Step{}, err
		}
		l := len(job.Steps)
		for i, s := range job.Steps {
			log.Printf("job step [%d/%d]: %s %v-%v", i+1, l, s.GetName(), s.StartedAt, s.CompletedAt)
			if s.StartedAt == nil || s.CompletedAt == nil {
				continue
			}
			// Truncate less than a second
			startTime := s.GetStartedAt().Time
			completeTime := s.GetCompletedAt().Time
			if startTime.Unix() < t.Unix() && t.Unix() <= completeTime.Unix() {
				log.Print("detect step")
				return Step{
					Name:        s.GetName(),
					StartedAt:   startTime,
					CompletedAt: completeTime,
				}, nil
			}
		}
	}
	return Step{}, fmt.Errorf("the step that was executed at the relevant time (%v) does not exist in the job (%d)", t, jobID)
}

type Step struct {
	Name        string
	StartedAt   time.Time
	CompletedAt time.Time
}

func (g *Gh) FetchStepsByName(ctx context.Context, owner, repo string, name string) ([]Step, error) {
	if os.Getenv("GITHUB_RUN_ID") == "" {
		return nil, fmt.Errorf("env %s is not set", "GITHUB_RUN_ID")
	}
	runID, err := strconv.ParseInt(os.Getenv("GITHUB_RUN_ID"), 10, 64)
	if err != nil {
		return nil, err
	}
	// Although it would be nice if we could get the job_id from an environment variable,
	// there is no way to get it at this time, so it uses a heuristic.
	p := backoff.Exponential( //nostyle:funcfmt
		backoff.WithMinInterval(time.Second),
		backoff.WithMaxInterval(30*time.Second),
		backoff.WithJitterFactor(0.05),
		backoff.WithMaxRetries(5),
	)
	b := p.Start(ctx)
	var steps []Step
	max := 0
L:
	for backoff.Continue(b) {
		max = 0
		jobs, err := g.listWorkflowJobs(ctx, owner, repo, runID)
		if err != nil {
			return nil, err
		}
		for _, j := range jobs {
			log.Printf("search job: %d", j.GetID()) //nolint:gosec // format string uses %d, no injection risk
			l := len(j.Steps)
			for i, s := range j.Steps {
				if s.GetName() == name {
					max += 1
					if s.StartedAt == nil || s.CompletedAt == nil {
						steps = []Step{}
						continue L
					}
					log.Printf("got job step [%d %d/%d]: %s %v-%v", j.GetID(), i+1, l, s.GetName(), s.StartedAt, s.CompletedAt) //nolint:gosec // values from GitHub API response, not user input
					steps = append(steps, Step{
						Name:        s.GetName(),
						StartedAt:   s.GetStartedAt().Time,
						CompletedAt: s.GetCompletedAt().Time,
					})
				}
			}
		}
		// Keep retrying while nothing matched. The step may belong to a job that
		// has not started yet, and returning the empty result as a success would
		// silently measure a test execution time of zero.
		if max > 0 && max == len(steps) {
			return steps, nil
		}
	}
	if max == 0 {
		return nil, fmt.Errorf("could not find any step named %q in the workflow run", name)
	}
	// A pass that collected every match returns inside the loop, so getting here
	// means the backoff ran out while a matching step was still incomplete.
	return nil, fmt.Errorf("step named %q did not complete in time", name)
}

func (g *Gh) PutComment(ctx context.Context, owner, repo string, n int, comment, key string) error {
	sig := generateSig(key)
	if err := g.minimizePreviousComments(ctx, owner, repo, n, sig); err != nil {
		return err
	}
	c := strings.Join([]string{comment, sig}, "\n")
	if _, _, err := g.client.Issues.CreateComment(ctx, owner, repo, n, &github.IssueComment{Body: &c}); err != nil {
		return err
	}
	return nil
}

func (g *Gh) PutCommentWithUpdate(ctx context.Context, owner, repo string, n int, comment, key string) error {
	sig := generateSig(key)
	commentId, err := g.findPreviousComment(ctx, owner, repo, n, sig)
	if err != nil {
		return err
	}

	if commentId == 0 {
		return g.PutComment(ctx, owner, repo, n, comment, key)
	} else {
		c := strings.Join([]string{comment, sig}, "\n")
		if _, _, err := g.client.Issues.EditComment(ctx, owner, repo, commentId, &github.IssueComment{Body: &c}); err != nil {
			return err
		}
		return nil
	}
}

func (g *Gh) PutCommentWithDeletion(ctx context.Context, owner, repo string, n int, comment, key string) error {
	sig := generateSig(key)
	if err := g.deletePreviousComments(ctx, owner, repo, n, sig); err != nil {
		return err
	}
	c := strings.Join([]string{comment, sig}, "\n")
	if _, _, err := g.client.Issues.CreateComment(ctx, owner, repo, n, &github.IssueComment{Body: &c}); err != nil {
		return err
	}
	return nil
}

func (g *Gh) PutArtifact(ctx context.Context, owner, repo string, runID int64, name, fp string, content []byte) error {
	current, _, err := g.client.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, runID, &github.ListOptions{})
	if err != nil {
		return err
	}
	for _, a := range current.Artifacts {
		if a.GetName() == name {
			if _, err := g.client.Actions.DeleteArtifact(ctx, owner, repo, a.GetID()); err != nil {
				return err
			}
			break
		}
	}
	return artifact.Upload(ctx, name, fp, bytes.NewReader(content))
}

// PutUnarchivedArtifact uploads content as an artifact of a single file named name, which
// is served as the file itself rather than as a zip of it, and returns the ID of the
// artifact. Where only a zipped artifact can be uploaded, which is on GitHub Enterprise
// Server, it is uploaded zipped instead.
func (g *Gh) PutUnarchivedArtifact(ctx context.Context, owner, repo string, runID int64, name string, content []byte) (int64, error) {
	if err := g.deleteRunArtifact(ctx, owner, repo, runID, name); err != nil {
		return 0, err
	}
	id, err := artifact.UploadUnarchived(ctx, name, bytes.NewReader(content))
	if !errors.Is(err, artifact.ErrUnarchivedUploadNotSupported) {
		return id, err
	}
	if err := artifact.Upload(ctx, name, name, bytes.NewReader(content)); err != nil {
		return 0, err
	}
	// The legacy upload does not answer with the ID, so it is looked up by the name, which
	// is unique within the run once the earlier one has been deleted.
	a, err := g.findRunArtifact(ctx, owner, repo, runID, name)
	if err != nil {
		return 0, err
	}
	if a == nil {
		return 0, fmt.Errorf("the uploaded artifact %s is not found", name)
	}
	return a.GetID(), nil
}

// ArtifactURL returns the URL an artifact of a workflow run is opened at.
func ArtifactURL(owner, repo string, runID, artifactID int64) string {
	server := strings.TrimRight(os.Getenv("GITHUB_SERVER_URL"), "/")
	if server == "" {
		server = DefaultGithubServerURL
	}
	return fmt.Sprintf("%s/%s/%s/actions/runs/%d/artifacts/%d", server, owner, repo, runID, artifactID)
}

// FetchMergeBase returns the commit the changes of the pull request are counted from.
func (g *Gh) FetchMergeBase(ctx context.Context, owner, repo string, number int) (string, error) {
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return "", err
	}
	compare, _, err := g.client.Repositories.CompareCommits(ctx, owner, repo, pr.GetBase().GetSHA(), pr.GetHead().GetSHA(), &github.ListOptions{PerPage: 1})
	if err != nil {
		return "", err
	}
	return compare.GetMergeBaseCommit().GetSHA(), nil
}

// DeleteArtifactsBeforeRun deletes the artifacts of the name that were uploaded by a workflow
// run with an id lower than runID. Two runs can finish out of order, so an artifact of a
// later run is kept even when it was uploaded first.
func (g *Gh) DeleteArtifactsBeforeRun(ctx context.Context, owner, repo, name string, runID int64) error {
	artifacts, err := g.artifactsBeforeRun(ctx, owner, repo, name, runID)
	if err != nil {
		return err
	}
	return g.deleteArtifacts(ctx, owner, repo, artifacts)
}

// DeleteCompletedArtifactsBeforeRun is DeleteArtifactsBeforeRun for the artifacts of the runs
// that have completed. A run still in progress may yet write a link to the artifact it
// uploaded, and deleting it first would leave that link pointing at nothing.
func (g *Gh) DeleteCompletedArtifactsBeforeRun(ctx context.Context, owner, repo, name string, runID int64) error {
	artifacts, err := g.artifactsBeforeRun(ctx, owner, repo, name, runID)
	if err != nil {
		return err
	}
	completed := map[int64]bool{}
	var done []*github.Artifact
	for _, a := range artifacts {
		id := a.GetWorkflowRun().GetID()
		ok, seen := completed[id]
		if !seen {
			run, _, err := g.client.Actions.GetWorkflowRunByID(ctx, owner, repo, id)
			if err != nil {
				return err
			}
			ok = run.GetStatus() == "completed"
			completed[id] = ok
		}
		if ok {
			done = append(done, a)
		}
	}
	return g.deleteArtifacts(ctx, owner, repo, done)
}

type ArtifactFile struct {
	ID        int64
	Name      string
	Content   []byte
	CreatedAt time.Time
}

// ErrArtifactNotFound is returned when no artifact holds the file asked for.
var ErrArtifactNotFound = errors.New("artifact not found")

func (g *Gh) FetchLatestArtifact(ctx context.Context, owner, repo, name, fp string) (*ArtifactFile, error) {
	return g.FetchLatestArtifactOfBranch(ctx, owner, repo, name, fp, "")
}

// FetchLatestArtifactOfBranch is FetchLatestArtifact for the artifacts that a workflow run on
// branch uploaded, or for all of them when branch is empty. A run of a pull request is left out,
// since its head branch is the branch it merges from: a release pull request from the default
// branch runs on it as a push does, and measures the merge onto another branch.
func (g *Gh) FetchLatestArtifactOfBranch(ctx context.Context, owner, repo, name, fp, branch string) (*ArtifactFile, error) {
	pullRequestRun := map[int64]bool{}
	page := 1
	for {
		l, res, err := g.client.Actions.ListArtifacts(ctx, owner, repo, &github.ListArtifactsOptions{
			Name: &name,
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		})
		if err != nil {
			return nil, err
		}
		page += 1
		for _, a := range l.Artifacts {
			if branch != "" {
				if a.GetWorkflowRun().GetHeadBranch() != branch {
					continue
				}
				id := a.GetWorkflowRun().GetID()
				pr, seen := pullRequestRun[id]
				if !seen {
					run, _, err := g.client.Actions.GetWorkflowRunByID(ctx, owner, repo, id)
					if err != nil {
						return nil, err
					}
					pr = run.GetEvent() == "pull_request" || run.GetEvent() == "pull_request_target"
					pullRequestRun[id] = pr
				}
				if pr {
					continue
				}
			}
			af, err := g.downloadArtifactFile(ctx, owner, repo, a, fp)
			if err != nil {
				return nil, err
			}
			if af != nil {
				return af, nil
			}
		}
		if res.NextPage == 0 {
			break
		}
	}
	return nil, ErrArtifactNotFound
}

// FetchArtifact returns the file fp of the artifact of the id. An artifact deleted or expired
// is reported as ErrArtifactNotFound, as one that does not hold the file is.
func (g *Gh) FetchArtifact(ctx context.Context, owner, repo string, id int64, fp string) (*ArtifactFile, error) {
	a, res, err := g.client.Actions.GetArtifact(ctx, owner, repo, id)
	if err != nil {
		if res != nil && res.StatusCode == http.StatusNotFound {
			return nil, fmt.Errorf("%w: artifact %d", ErrArtifactNotFound, id)
		}
		return nil, err
	}
	if a.GetExpired() {
		return nil, fmt.Errorf("%w: artifact %d has expired", ErrArtifactNotFound, id)
	}
	af, err := g.downloadArtifactFile(ctx, owner, repo, a, fp)
	if err != nil {
		return nil, err
	}
	if af == nil {
		return nil, fmt.Errorf("%w: %s in artifact %d", ErrArtifactNotFound, fp, id)
	}
	return af, nil
}

// FetchRunArtifactID returns the ID of the artifact of the name the workflow run uploaded.
func (g *Gh) FetchRunArtifactID(ctx context.Context, owner, repo string, runID int64, name string) (int64, error) {
	a, err := g.findRunArtifact(ctx, owner, repo, runID, name)
	if err != nil {
		return 0, err
	}
	if a == nil {
		return 0, fmt.Errorf("%w: %s in run %d", ErrArtifactNotFound, name, runID)
	}
	return a.GetID(), nil
}

func (g *Gh) IsPrivate(ctx context.Context, owner, repo string) (bool, error) {
	r, _, err := g.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return false, err
	}
	return r.GetPrivate(), nil
}

// HasCommentReport reports whether the pull request has a comment octocov wrote a report of key
// into.
func (g *Gh) HasCommentReport(ctx context.Context, owner, repo string, number int, key string) (bool, error) {
	id, err := g.findPreviousComment(ctx, owner, repo, number, generateSig(key))
	if err != nil {
		return false, err
	}
	return id != 0, nil
}

// HasBodyReport reports whether the body of the pull request carries a report of key octocov
// inserted into it.
func (g *Gh) HasBodyReport(ctx context.Context, owner, repo string, number int, key string) (bool, error) {
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return false, err
	}
	return strings.Contains(pr.GetBody(), generateSig(key)), nil
}

// DeleteRunArtifacts deletes the artifacts of the workflow run whose name match reports true
// for, as the ones earlier attempts of a re-run uploaded.
func (g *Gh) DeleteRunArtifacts(ctx context.Context, owner, repo string, runID int64, match func(name string) bool) error {
	// Every page is read before anything is deleted, since deleting while paging shifts the
	// artifacts that follow onto a page that has already been read.
	var ids []int64
	opts := &github.ListOptions{PerPage: 100}
	for {
		l, res, err := g.client.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, runID, opts)
		if err != nil {
			return err
		}
		for _, a := range l.Artifacts {
			if match(a.GetName()) {
				ids = append(ids, a.GetID())
			}
		}
		if res.NextPage == 0 {
			break
		}
		opts.Page = res.NextPage
	}
	for _, id := range ids {
		if _, err := g.client.Actions.DeleteArtifact(ctx, owner, repo, id); err != nil {
			return fmt.Errorf("failed to delete artifact %d: %w", id, err)
		}
	}
	return nil
}

// artifactsBeforeRun returns the artifacts of the name that were uploaded by a workflow run with
// an id lower than runID.
func (g *Gh) artifactsBeforeRun(ctx context.Context, owner, repo, name string, runID int64) ([]*github.Artifact, error) {
	// Every page is read before anything is deleted, since deleting while paging shifts
	// the artifacts that follow onto a page that has already been read.
	var artifacts []*github.Artifact
	opts := &github.ListArtifactsOptions{
		Name:        &name,
		ListOptions: github.ListOptions{PerPage: 100},
	}
	for {
		l, res, err := g.client.Actions.ListArtifacts(ctx, owner, repo, opts)
		if err != nil {
			return nil, err
		}
		for _, a := range l.Artifacts {
			if a.GetName() != name {
				continue
			}
			// An artifact that does not say which run uploaded it cannot be shown to be
			// older, so it is left alone.
			if id := a.GetWorkflowRun().GetID(); id != 0 && id < runID {
				artifacts = append(artifacts, a)
			}
		}
		if res.NextPage == 0 {
			break
		}
		opts.Page = res.NextPage
	}
	return artifacts, nil
}

func (g *Gh) deleteArtifacts(ctx context.Context, owner, repo string, artifacts []*github.Artifact) error {
	for _, a := range artifacts {
		// Stopping at the first failure rather than trying the rest, since the usual cause
		// is a token without actions: write, which every other delete would fail on too.
		if _, err := g.client.Actions.DeleteArtifact(ctx, owner, repo, a.GetID()); err != nil {
			return fmt.Errorf("failed to delete artifact %d: %w", a.GetID(), err)
		}
	}
	return nil
}

// deleteRunArtifact deletes the artifact of the name the run has already uploaded, since a name
// can be used only once within a run.
func (g *Gh) deleteRunArtifact(ctx context.Context, owner, repo string, runID int64, name string) error {
	a, err := g.findRunArtifact(ctx, owner, repo, runID, name)
	if err != nil || a == nil {
		return err
	}
	_, err = g.client.Actions.DeleteArtifact(ctx, owner, repo, a.GetID())
	return err
}

// findRunArtifact returns the artifact of the name the workflow run uploaded, or nil when
// there is none. Every page is read, since a run with a large matrix can hold more
// artifacts than one page does.
func (g *Gh) findRunArtifact(ctx context.Context, owner, repo string, runID int64, name string) (*github.Artifact, error) {
	opts := &github.ListOptions{PerPage: 100}
	for {
		l, res, err := g.client.Actions.ListWorkflowRunArtifacts(ctx, owner, repo, runID, opts)
		if err != nil {
			return nil, err
		}
		for _, a := range l.Artifacts {
			if a.GetName() == name {
				return a, nil
			}
		}
		if res.NextPage == 0 {
			return nil, nil
		}
		opts.Page = res.NextPage
	}
}

// downloadArtifactFile returns the file fp of the artifact, or nil when the artifact does not
// hold it.
func (g *Gh) downloadArtifactFile(ctx context.Context, owner, repo string, a *github.Artifact, fp string) (*ArtifactFile, error) {
	const maxRedirect = 5
	u, _, err := g.client.Actions.DownloadArtifact(ctx, owner, repo, a.GetID(), maxRedirect)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	buf := new(bytes.Buffer)
	size, err := io.CopyN(buf, resp.Body, maxCopySize)
	if !errors.Is(err, io.EOF) {
		return nil, err
	}
	if size >= maxCopySize {
		return nil, fmt.Errorf("too large file size to copy: %d >= %d", size, maxCopySize)
	}
	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return nil, err
	}
	for _, file := range reader.File {
		if file.Name != fp {
			continue
		}
		in, err := file.Open()
		if err != nil {
			return nil, err
		}
		out := new(bytes.Buffer)
		size, err := io.CopyN(out, in, maxCopySize)
		if !errors.Is(err, io.EOF) {
			_ = in.Close() //nostyle:handlerrors
			return nil, err
		}
		if size >= maxCopySize {
			_ = in.Close() //nostyle:handlerrors
			return nil, fmt.Errorf("too large file size to copy: %d >= %d", size, maxCopySize)
		}
		if err := in.Close(); err != nil {
			return nil, err
		}
		return &ArtifactFile{
			ID:        a.GetID(),
			Name:      file.Name,
			Content:   out.Bytes(),
			CreatedAt: a.GetCreatedAt().Time,
		}, nil
	}
	return nil, nil
}

// fetchMergeCommitFiles returns the files commit changes against its first parent, and that
// parent, when commit is the merge of head onto the base branch of a pull request, and false
// otherwise.
func (g *Gh) fetchMergeCommitFiles(ctx context.Context, owner, repo, commit, head string) ([]*github.CommitFile, string, bool, error) {
	// The git data API rather than the commits API, which would also send the files of the
	// commit only to have them thrown away. The parents cannot be read from the local checkout
	// either, since a shallow checkout of the merge commit does not carry them.
	c, _, err := g.client.Git.GetCommit(ctx, owner, repo, commit)
	if err != nil {
		return nil, "", false, err
	}
	if len(c.Parents) != 2 || c.Parents[1].GetSHA() != head {
		// As on pull_request_target, or on a run the head has moved past since.
		return nil, "", false, nil
	}
	// Not paginated, because pages split the commits only. The files come on the first page,
	// up to compareFilesLimit of them for the whole comparison.
	comparison, _, err := g.client.Repositories.CompareCommits(ctx, owner, repo, c.Parents[0].GetSHA(), commit, &github.ListOptions{})
	if err != nil {
		return nil, "", false, err
	}
	return comparison.Files, c.Parents[0].GetSHA(), true, nil
}

func (g *Gh) listPullRequestFiles(ctx context.Context, owner, repo string, number int) ([]*PullRequestFile, error) {
	var files []*PullRequestFile
	page := 1
	for {
		commitFiles, _, err := g.client.PullRequests.ListFiles(ctx, owner, repo, number, &github.ListOptions{
			Page:    page,
			PerPage: 100,
		})
		if err != nil {
			return nil, err
		}
		if len(commitFiles) == 0 {
			break
		}
		for _, f := range commitFiles {
			files = append(files, &PullRequestFile{
				Filename:         f.GetFilename(),
				PreviousFilename: f.GetPreviousFilename(),
				BlobURL:          f.GetBlobURL(),
				Status:           f.GetStatus(),
				Additions:        f.GetAdditions(),
				Deletions:        f.GetDeletions(),
				Patch:            f.GetPatch(),
				ChangedLines:     parseChangedLinesFromPatch(f.GetPatch()),
			})
		}
		page += 1
	}
	return files, nil
}

// listWorkflowJobs returns every job of the workflow run, following pagination.
// The API returns 30 jobs per page by default, so a run with a large matrix
// would otherwise be silently truncated.
func (g *Gh) listWorkflowJobs(ctx context.Context, owner, repo string, runID int64) ([]*github.WorkflowJob, error) {
	opts := &github.ListWorkflowJobsOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var jobs []*github.WorkflowJob
	for {
		js, res, err := g.client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, opts)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, js.Jobs...)
		if res.NextPage == 0 {
			return jobs, nil
		}
		opts.Page = res.NextPage
	}
}

type minimizeCommentMutation struct {
	MinimizeComment struct {
		MinimizedComment struct {
			IsMinimized bool
		}
	} `graphql:"minimizeComment(input: $input)"`
}

func (g *Gh) minimizePreviousComments(ctx context.Context, owner, repo string, n int, sig string) error {
	page := 1
	for {
		opts := &github.IssueListCommentsOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		}
		comments, res, err := g.client.Issues.ListComments(ctx, owner, repo, n, opts)
		if err != nil {
			return err
		}
		for _, c := range comments {
			if strings.Contains(*c.Body, sig) {
				var m minimizeCommentMutation
				input := githubv4.MinimizeCommentInput{
					SubjectID:        githubv4.ID(c.GetNodeID()),
					Classifier:       githubv4.ReportedContentClassifiers("OUTDATED"),
					ClientMutationID: nil,
				}
				if err := g.v4Client.Mutate(ctx, &m, input, nil); err != nil {
					return err
				}
			}
		}
		if res.NextPage == 0 {
			break
		}
		page = res.NextPage
	}
	return nil
}

func (g *Gh) findPreviousComment(ctx context.Context, owner, repo string, n int, sig string) (int64, error) {
	page := 1
	for {
		opts := &github.IssueListCommentsOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		}
		comments, res, err := g.client.Issues.ListComments(ctx, owner, repo, n, opts)
		if err != nil {
			return 0, err
		}
		for _, c := range comments {
			if strings.Contains(*c.Body, sig) {
				return *c.ID, nil
			}
		}
		if res.NextPage == 0 {
			break
		}
		page = res.NextPage
	}
	return 0, nil
}

func (g *Gh) deletePreviousComments(ctx context.Context, owner, repo string, n int, sig string) error {
	page := 1
	for {
		opts := &github.IssueListCommentsOptions{
			ListOptions: github.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		}
		comments, res, err := g.client.Issues.ListComments(ctx, owner, repo, n, opts)
		if err != nil {
			return err
		}
		for _, c := range comments {
			if strings.Contains(*c.Body, sig) {
				_, err = g.client.Issues.DeleteComment(ctx, owner, repo, *c.ID)
				if err != nil {
					return err
				}
			}
		}
		if res.NextPage == 0 {
			break
		}
		page = res.NextPage
	}
	return nil
}

func PushUsingLocalGit(ctx context.Context, gitRoot string, addPaths []string, message string) (int, error) {
	r, err := git.PlainOpen(gitRoot)
	if err != nil {
		return 0, err
	}
	w, err := r.Worktree()
	if err != nil {
		return 0, err
	}
	status, err := w.Status()
	if err != nil {
		return 0, err
	}

	c := 0
	for _, p := range addPaths {
		rel, err := filepath.Rel(gitRoot, p)
		if err != nil {
			return 0, err
		}
		if _, ok := status[rel]; ok {
			c += 1
			_, err := w.Add(rel)
			if err != nil {
				return 0, err
			}
		}
	}

	if c == 0 {
		return c, nil
	}

	opts := &git.CommitOptions{}
	switch {
	case os.Getenv("GITHUB_SERVER_URL") == DefaultGithubServerURL:
		opts.Author = &object.Signature{
			Name:  "github-actions",
			Email: "41898282+github-actions[bot]@users.noreply.github.com",
			When:  time.Now(),
		}
	case os.Getenv("GITHUB_ACTOR") != "":
		opts.Author = &object.Signature{
			Name:  os.Getenv("GITHUB_ACTOR"),
			Email: fmt.Sprintf("%s@users.noreply.github.com", os.Getenv("GITHUB_ACTOR")),
			When:  time.Now(),
		}
	}
	if _, err := w.Commit(message, opts); err != nil {
		return c, err
	}

	if err := r.PushContext(ctx, &git.PushOptions{
		Auth: &ghttp.BasicAuth{
			Username: "octocov",
			Password: os.Getenv("GITHUB_TOKEN"),
		},
	}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return c, err
	}

	return c, nil
}

type GitHubEvent struct {
	Name   string
	Number int
	State  string
	// HeadSHA is the commit the head of the pull request is at, which on a pull request
	// event is not the commit the run checks out, that being the merge of it onto the base.
	HeadSHA string
	Payload any
}

func DecodeGitHubEvent() (*GitHubEvent, error) {
	i := &GitHubEvent{}
	n := os.Getenv("GITHUB_EVENT_NAME")
	if n == "" {
		return i, fmt.Errorf("env %s is not set", "GITHUB_EVENT_NAME")
	}
	i.Name = n
	p := os.Getenv("GITHUB_EVENT_PATH")
	if p == "" {
		return i, fmt.Errorf("env %s is not set", "GITHUB_EVENT_PATH")
	}
	b, err := os.ReadFile(filepath.Clean(p)) //nolint:gosec // p is from trusted GITHUB_EVENT_PATH env var
	if err != nil {
		return i, err
	}
	s := struct {
		PullRequest struct {
			Number int    `json:"number,omitempty"`
			State  string `json:"state,omitempty"`
			Head   struct {
				SHA string `json:"sha,omitempty"`
			} `json:"head,omitzero"`
		} `json:"pull_request,omitzero"`
		Issue struct {
			Number int    `json:"number,omitempty"`
			State  string `json:"state,omitempty"`
		} `json:"issue,omitzero"`
	}{}
	if err := json.Unmarshal(b, &s); err != nil {
		return i, err
	}
	switch {
	case s.PullRequest.Number > 0:
		i.Number = s.PullRequest.Number
		i.State = s.PullRequest.State
		i.HeadSHA = s.PullRequest.Head.SHA
	case s.Issue.Number > 0:
		i.Number = s.Issue.Number
		i.State = s.Issue.State
	}

	var payload any

	if err := json.Unmarshal(b, &payload); err != nil {
		return i, err
	}

	i.Payload = payload

	return i, nil
}

type Repository struct {
	Owner string
	Repo  string
	Path  string
}

func (r *Repository) Reponame() string {
	if r.Path == "" {
		return r.Repo
	}
	return fmt.Sprintf("%s/%s", r.Repo, r.Path)
}

func Parse(raw string) (*Repository, error) {
	splitted := strings.Split(raw, "/")
	if len(splitted) < 2 {
		return nil, fmt.Errorf("could not parse: %s", raw)
	}
	for _, p := range splitted {
		if p == "" {
			return nil, fmt.Errorf("invalid repository path: %s", raw)
		}
		if strings.Trim(p, ".") == "" {
			return nil, fmt.Errorf("invalid repository path: %s", raw)
		}
	}

	r := &Repository{
		Owner: splitted[0],
		Repo:  splitted[1],
	}
	if len(splitted) > 2 {
		r.Path = strings.Join(splitted[2:], "/")
	}

	return r, nil
}

// trimContentURL trim suffix path and private token.
func trimContentURL(u, p string) (string, error) {
	parsed, err := url.Parse(u) //nostyle:handlerrors
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(parsed.String(), fmt.Sprintf("?%s", parsed.RawQuery)), p), "/"), nil
}

func generateSig(key string) string {
	if key == "" {
		return "<!-- octocov -->"
	}
	return fmt.Sprintf("<!-- octocov:%s -->", key)
}

// insertToBody embeds content delimited by sig into current, replacing the previously
// embedded content when current already holds a pair of sig.
func insertToBody(current, content, sig string) (string, error) {
	if strings.Count(current, sig) < 2 {
		return fmt.Sprintf("%s%s\n%s\n%s\n", terminateBeforeSig(current), sig, content, sig), nil
	}
	// repin.Replace copies everything preceding the opening sig verbatim, so bodies embedded
	// by earlier versions keep their missing blank line unless it is restored here.
	if i := strings.Index(current, sig); i > 0 {
		current = terminateBeforeSig(current[:i]) + current[i:]
	}
	if !strings.HasSuffix(current, "\n") {
		current += "\n"
	}
	buf := new(bytes.Buffer)
	if _, err := repin.Replace(strings.NewReader(current), strings.NewReader(content), sig, sig, false, buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// terminateBeforeSig closes before with a blank line so that the sig following it starts a
// fresh Markdown block. GitHub parses an unterminated HTML block, such as a badge <a> tag
// appended by another GitHub App, as raw HTML up to the next blank line, which would
// otherwise swallow the report heading and table and render them as literal text.
func terminateBeforeSig(before string) string {
	before = strings.TrimRight(before, "\r\n")
	if before == "" {
		return ""
	}
	return before + "\n\n"
}
