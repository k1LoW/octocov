package artifact

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
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
	return a.gh.PutArtifact(ctx, a.name, path, content)
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
		name = fmt.Sprintf("%s-%s", a.name, keyRep.Replace(a.r.Key()))
	}
	log.Printf("artifact name: %s", name)
	af, err := a.gh.GetLatestArtifact(ctx, r.Owner, r.Repo, name, reportFilename)
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
