package datastore

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/report"
)

type Github struct {
	config *config.Config
	client *github.Client
}

// NewGithub ...
func NewGithub(c *config.Config) (*Github, error) {
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

	return &Github{
		config: c,
		client: v3c,
	}, nil
}

func (g *Github) Push(ctx context.Context, r *report.Report) error {
	srv := g.client.Git
	branch := g.config.Push.Github.Branch
	content := r.String()
	rPath := g.config.Push.Github.Path
	from := r.Repository
	if g.config.Report.Repository != "" {
		from = g.config.Report.Repository
	}
	message := fmt.Sprintf("Push coverage report of %s", from)
	splitted := strings.Split(g.config.Push.Github.Repository, "/")
	owner := splitted[0]
	repo := splitted[1]

	dRef, _, err := srv.GetRef(ctx, owner, repo, path.Join("heads", branch))
	if err != nil {
		return err
	}

	parent, _, err := srv.GetCommit(ctx, owner, repo, *dRef.Object.SHA)
	if err != nil {
		return err
	}

	var tree *github.Tree

	if rPath != "" {
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
			Path: github.String(rPath),
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
