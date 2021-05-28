package gh

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
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
	"github.com/google/go-github/v35/github"
	"github.com/lestrrat-go/backoff/v2"
)

const DefaultGithubServerURL = "https://github.com"

var octocovNameRe = regexp.MustCompile(`(?i)(octocov|coverage)`)

type Gh struct {
	client *github.Client
}

func New() (*Gh, error) {
	// GITHUB_TOKEN
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("env %s is not set", "GITHUB_TOKEN")
	}
	v3c := github.NewClient(httpClient(token))
	if v3ep := os.Getenv("GITHUB_API_URL"); v3ep != "" {
		baseEndpoint, err := url.Parse(v3ep)
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(baseEndpoint.Path, "/") {
			baseEndpoint.Path += "/"
		}
		v3c.BaseURL = baseEndpoint
	}

	return &Gh{
		client: v3c,
	}, nil
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
	resC, _, err := srv.CreateCommit(ctx, owner, repo, commit)
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

func (g *Gh) GetRawRootURL(ctx context.Context, owner, repo string) (string, error) {
	r, _, err := g.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return "", err
	}
	b := r.GetDefaultBranch()

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
		return strings.TrimSuffix(strings.TrimSuffix(fc.GetDownloadURL(), path), "/"), nil
	}
	return "", fmt.Errorf("not found files. please commit file to root directory and push: %s/%s", owner, repo)
}

func (g *Gh) DetectCurrentJobID(ctx context.Context, owner, repo string, nameRe *regexp.Regexp) (int64, error) {
	if os.Getenv("GITHUB_RUN_ID") == "" {
		return 0, fmt.Errorf("env %s is not set", "GITHUB_RUN_ID")
	}
	runID, err := strconv.ParseInt(os.Getenv("GITHUB_RUN_ID"), 10, 64)
	if err != nil {
		return 0, err
	}

	// Although it would be nice if we could get the job_id from an environment variable,
	// there is no way to get it at this time, so it uses a heuristic.
	p := backoff.Exponential(
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
			if j.GetName() == os.Getenv("GTIHUB_JOB") {
				return j.GetID(), nil
			}
			for _, s := range j.Steps {
				if nameRe != nil {
					if nameRe.MatchString(s.GetName()) {
						return j.GetID(), nil
					}
				}
				if s.StartedAt != nil && s.CompletedAt == nil && octocovNameRe.MatchString(s.GetName()) {
					return j.GetID(), nil
				}
			}
		}
	}

	return 0, errors.New("could not detect id of current job")
}

func (g *Gh) GetStepExecutionTimeByTime(ctx context.Context, owner, repo string, jobID int64, t time.Time) (time.Duration, error) {
	p := backoff.Exponential(
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
	return 0, fmt.Errorf("the step that was executed at the relevant time (%v) does not exist in the job (%d).", t, jobID)
}

func PushUsingLocalGit(ctx context.Context, gitRoot string, addPaths []string, message string) error {
	r, err := git.PlainOpen(gitRoot)
	if err != nil {
		return err
	}
	w, err := r.Worktree()
	if err != nil {
		return err
	}
	status, err := w.Status()
	if err != nil {
		return err
	}
	push := false
	for _, p := range addPaths {
		rel, err := filepath.Rel(gitRoot, p)
		if err != nil {
			return err
		}
		if _, ok := status[rel]; ok {
			push = true
			_, err := w.Add(rel)
			if err != nil {
				return err
			}
		}
	}

	if !push {
		return nil
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
		return err
	}

	if err := r.PushContext(ctx, &git.PushOptions{
		Auth: &ghttp.BasicAuth{
			Username: "octocov",
			Password: os.Getenv("GITHUB_TOKEN"),
		},
	}); err != nil {
		return err
	}

	return nil
}

type roundTripper struct {
	transport   *http.Transport
	accessToken string
}

func (rt roundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	r.Header.Set("Authorization", fmt.Sprintf("token %s", rt.accessToken))
	return rt.transport.RoundTrip(r)
}

func httpClient(token string) *http.Client {
	t := &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	}
	rt := roundTripper{
		transport:   t,
		accessToken: token,
	}
	return &http.Client{
		Timeout:   time.Second * 10,
		Transport: rt,
	}
}
