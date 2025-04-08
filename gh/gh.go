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
			Content:  github.String(content),
			Encoding: github.String("utf-8"),
			Size:     github.Int(len(content)),
		}

		resB, _, err := srv.CreateBlob(ctx, owner, repo, blob)
		if err != nil {
			return err
		}

		entry := &github.TreeEntry{
			Path: github.String(cp),
			Mode: github.String("100644"),
			Type: github.String("blob"),
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
		Message: github.String(message),
		Tree:    tree,
		Parents: []*github.Commit{parent},
	}
	resC, _, err := srv.CreateCommit(ctx, owner, repo, commit, &github.CreateCommitOptions{})
	if err != nil {
		return err
	}

	nref := &github.Reference{
		Ref: github.String(path.Join("refs", "heads", branch)),
		Object: &github.GitObject{
			Type: github.String("commit"),
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
		jobs, _, err := g.client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, &github.ListWorkflowJobsOptions{})
		if err != nil {
			return 0, err
		}
		if len(jobs.Jobs) == 1 {
			return jobs.Jobs[0].GetID(), nil
		}
		for _, j := range jobs.Jobs {
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
	splitted := strings.SplitN(os.Getenv("GITHUB_REF"), "/", 3) // refs/pull/8/head or refs/heads/branch/branch/name
	if len(splitted) < 3 {
		return "", fmt.Errorf("env %s is not set", "GITHUB_REF")
	}
	if strings.Contains(os.Getenv("GITHUB_REF"), "refs/heads/") {
		return splitted[2], nil
	}
	if os.Getenv("GITHUB_HEAD_REF") == "" {
		return "", fmt.Errorf("env %s is not set", "GITHUB_HEAD_REF")
	}
	return os.Getenv("GITHUB_HEAD_REF"), nil
}

func (g *Gh) DetectCurrentPullRequestNumber(ctx context.Context, owner, repo string) (int, error) {
	if os.Getenv("GITHUB_PULL_REQUEST_NUMBER") != "" {
		return strconv.Atoi(os.Getenv("GITHUB_PULL_REQUEST_NUMBER"))
	}
	splitted := strings.Split(os.Getenv("GITHUB_REF"), "/") // refs/pull/8/head or refs/heads/branch/branch/name
	if len(splitted) < 3 {
		return 0, fmt.Errorf("env %s is not set", "GITHUB_REF")
	}
	if strings.Contains(os.Getenv("GITHUB_REF"), "refs/pull/") {
		prNumber := splitted[2]
		return strconv.Atoi(prNumber)
	}
	b := strings.Join(splitted[2:], "/")
	l, _, err := g.client.PullRequests.List(ctx, owner, repo, &github.PullRequestListOptions{
		State: "open",
	})
	if err != nil {
		return 0, err
	}
	var d *github.PullRequest
	for _, pr := range l {
		if pr.GetHead().GetRef() == b {
			if d != nil {
				return 0, errors.New("could not detect number of pull request")
			}
			d = pr
		}
	}
	if d != nil {
		return d.GetNumber(), nil
	}
	return 0, errors.New("could not detect number of pull request")
}

func (g *Gh) ReplaceInsertToBody(ctx context.Context, owner, repo string, number int, content, key string) error {
	sig := generateSig(key)
	pr, _, err := g.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return err
	}
	current := pr.GetBody()
	var rep string
	if strings.Count(current, sig) < 2 {
		rep = fmt.Sprintf("%s\n%s\n%s\n%s\n", current, sig, content, sig)
	} else {
		buf := new(bytes.Buffer)
		if !strings.HasSuffix(current, "\n") {
			current += "\n"
		}
		if _, err := repin.Replace(strings.NewReader(current), strings.NewReader(content), sig, sig, false, buf); err != nil {
			return err
		}
		rep = buf.String()
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
	Filename string
	BlobURL  string
	Status   string
}

func (g *Gh) FetchPullRequestFiles(ctx context.Context, owner, repo string, number int) ([]*PullRequestFile, error) {
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
				Filename: f.GetFilename(),
				BlobURL:  f.GetBlobURL(),
				Status:   f.GetStatus(),
			})
		}
		page += 1
	}
	return files, nil
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
			Filename: f.GetFilename(),
			BlobURL:  f.GetBlobURL(),
		})
	}
	return files, nil
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
			if s.GetStartedAt().Time.Unix() < t.Unix() && t.Unix() <= s.GetCompletedAt().Time.Unix() {
				log.Print("detect step")
				return s.GetCompletedAt().Time.Sub(s.GetStartedAt().Time), nil
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
			if s.GetStartedAt().Time.Unix() < t.Unix() && t.Unix() <= s.GetCompletedAt().Time.Unix() {
				log.Print("detect step")
				return Step{
					Name:        s.GetName(),
					StartedAt:   s.GetStartedAt().Time,
					CompletedAt: s.GetCompletedAt().Time,
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
		jobs, _, err := g.client.Actions.ListWorkflowJobs(ctx, owner, repo, runID, &github.ListWorkflowJobsOptions{})
		if err != nil {
			return nil, err
		}
		for _, j := range jobs.Jobs {
			log.Printf("search job: %d", j.GetID())
			l := len(j.Steps)
			for i, s := range j.Steps {
				if s.GetName() == name {
					max += 1
					if s.StartedAt == nil || s.CompletedAt == nil {
						steps = []Step{}
						continue L
					}
					log.Printf("got job step [%d %d/%d]: %s %v-%v", j.GetID(), i+1, l, s.GetName(), s.StartedAt, s.CompletedAt)
					steps = append(steps, Step{
						Name:        s.GetName(),
						StartedAt:   s.GetStartedAt().Time,
						CompletedAt: s.GetCompletedAt().Time,
					})
				}
			}
		}
		if max == len(steps) {
			return steps, nil
		}
	}
	if max < len(steps) || len(steps) == 0 {
		return nil, fmt.Errorf("could not get step times: %s", name)
	}
	return steps, nil
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

func (g *Gh) PutArtifact(ctx context.Context, name, fp string, content []byte) error {
	return artifact.Upload(ctx, name, fp, bytes.NewReader(content))
}

type ArtifactFile struct {
	Name      string
	Content   []byte
	CreatedAt time.Time
}

func (g *Gh) FetchLatestArtifact(ctx context.Context, owner, repo, name, fp string) (*ArtifactFile, error) {
	const maxRedirect = 5
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
			u, _, err := g.client.Actions.DownloadArtifact(ctx, owner, repo, a.GetID(), maxRedirect)
			if err != nil {
				return nil, err
			}
			resp, err := http.Get(u.String())
			if err != nil {
				return nil, err
			}
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
					Name:      file.Name,
					Content:   out.Bytes(),
					CreatedAt: a.CreatedAt.Time,
				}, nil
			}
		}
		if res.NextPage == 0 {
			break
		}
	}
	return nil, errors.New("artifact not found")
}

func (g *Gh) IsPrivate(ctx context.Context, owner, repo string) (bool, error) {
	r, _, err := g.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return false, err
	}
	return r.GetPrivate(), nil
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
	Name    string
	Number  int
	State   string
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
	b, err := os.ReadFile(filepath.Clean(p))
	if err != nil {
		return i, err
	}
	s := struct {
		PullRequest struct {
			Number int    `json:"number,omitempty"`
			State  string `json:"state,omitempty"`
		} `json:"pull_request,omitempty"`
		Issue struct {
			Number int    `json:"number,omitempty"`
			State  string `json:"state,omitempty"`
		} `json:"issue,omitempty"`
	}{}
	if err := json.Unmarshal(b, &s); err != nil {
		return i, err
	}
	switch {
	case s.PullRequest.Number > 0:
		i.Number = s.PullRequest.Number
		i.State = s.PullRequest.State
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
