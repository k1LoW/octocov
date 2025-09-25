package artifact

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"
	"testing/fstest"

	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

const defaultArtifactName = "octocov-report"
const reportFilename = "report.json"

var keyRep = strings.NewReplacer(`"`, "_", ":", "_", "<", "_", ">", "_", "|", "_", "*", "_", "?", "_", "\r", "_", "\n", "_", "\\", "_", "/", "_")

type Artifact struct {
	gh         *gh.Gh
	repository string
	name       string
	r          *report.Report
}

func New(gh *gh.Gh, repo, name string, r *report.Report) (*Artifact, error) {
	if name == "" {
		name = defaultArtifactName
	}
	return &Artifact{
		gh:         gh,
		repository: repo,
		name:       name,
		r:          r,
	}, nil
}

func (a *Artifact) StoreReport(ctx context.Context, r *report.Report) error {
	switch {
	case a.repository == r.Repository:
		return a.Put(ctx, reportFilename, r.Bytes())
	case strings.HasPrefix(r.Repository, fmt.Sprintf("%s/", a.repository)):
		a.name = fmt.Sprintf("%s-%s", a.name, keyRep.Replace(r.Key()))
		return a.Put(ctx, reportFilename, r.Bytes())
	default:
		return errors.New("reporting to the artifact can only be sent from the GitHub Actions of the same repository")
	}
}

func (a *Artifact) Put(ctx context.Context, path string, content []byte) error {
	// i think this fails
	r, err := gh.Parse(a.repository)
	if err != nil {
		return err
	}
	s := os.Getenv("GITHUB_RUN_ID")
	if s == "" {
		return errors.New("env GITHUB_RUN_ID is not set")
	}
	runID, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to parse GITHUB_RUN_ID: %w", err)
	}
	return a.gh.PutArtifact(ctx, r.Owner, r.Repo, runID, a.name, path, content)
}

func (a *Artifact) FS() (fs.FS, error) {
	ctx := context.Background()
	var (
		path, name string
		r          *gh.Repository
		err        error
	)
	if a.r == nil {
		r, err = gh.Parse(a.repository)
		if err != nil {
			return nil, err
		}
		path = fmt.Sprintf("%s/%s/%s", r.Owner, r.Repo, reportFilename)
		name = a.name
	} else {
		r, err = gh.Parse(a.r.Repository)
		if err != nil {
			return nil, err
		}
		path = fmt.Sprintf("%s/%s/%s", r.Owner, r.Reponame(), reportFilename)
		key := keyRep.Replace(a.r.Key())
		if key == "" {
			name = a.name
		} else {
			name = fmt.Sprintf("%s-%s", a.name, keyRep.Replace(a.r.Key()))
		}
	}
	log.Printf("artifact name: %s", name)
	af, err := a.gh.FetchLatestArtifact(ctx, r.Owner, r.Repo, name, reportFilename)
	fsys := fstest.MapFS{}
	if err == nil {
		fsys[path] = &fstest.MapFile{
			Data:    af.Content,
			Mode:    fs.ModePerm,
			ModTime: af.CreatedAt,
		}
	}
	return &fsys, nil
}
