package artifact

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"testing/fstest"

	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

const defaultArtifactName = "octocov-report"
const reportFilename = "report.json"

type Artifact struct {
	gh         *gh.Gh
	repository string
	name       string
}

func New(gh *gh.Gh, r, name string) (*Artifact, error) {
	if name == "" {
		name = defaultArtifactName
	}
	return &Artifact{
		gh:         gh,
		repository: r,
		name:       name,
	}, nil
}

func (a *Artifact) StoreReport(ctx context.Context, r *report.Report) error {
	if a.repository != r.Repository {
		return errors.New("reporting to the artifact can only be sent from the GitHub Actions of the same repository")
	}
	return a.Put(ctx, reportFilename, r.Bytes())
}

func (a *Artifact) Put(ctx context.Context, path string, content []byte) error {
	if a.repository != os.Getenv("GITHUB_REPOSITORY") {
		return errors.New("reporting to the artifact can only be sent from the GitHub Actions of the same repository")
	}
	return a.gh.PutArtifact(ctx, a.name, path, content)
}

func (a *Artifact) FS() (fs.FS, error) {
	ctx := context.Background()
	r, err := gh.Parse(a.repository)
	if err != nil {
		return nil, err
	}
	fsys := fstest.MapFS{}

	path := fmt.Sprintf("%s/%s/%s", r.Owner, r.Repo, reportFilename)
	af, err := a.gh.GetLatestArtifact(ctx, r.Owner, r.Repo, a.name, reportFilename)
	if err == nil {
		fsys[path] = &fstest.MapFile{
			Data:    af.Content,
			Mode:    fs.ModePerm,
			ModTime: af.CreatedAt,
		}
	}
	return &fsys, nil
}
