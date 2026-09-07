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
const reportFilename = report.Filename

// refSeparator marks off the ref key an artifact name carries. It is not one of the
// characters an artifact name may not contain, keyRep never produces it, and it needs
// no escaping where the name travels in a query string, so a reader can split a name on
// its last occurrence and take what follows as the ref.
const refSeparator = "@"

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
	name, err := a.storeName(r)
	if err != nil {
		return err
	}
	return a.put(ctx, name, reportFilename, r.Bytes())
}

func (a *Artifact) Put(ctx context.Context, path string, content []byte) error {
	return a.put(ctx, a.name, path, content)
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

// storeName returns the artifact name the report is stored under. The report of the
// default branch keeps the configured name, which is the one FS() looks up, so a
// comparison and the central mode keep reading the branch they are meant to describe.
func (a *Artifact) storeName(r *report.Report) (string, error) {
	name := a.name
	switch {
	case a.repository == r.Repository:
	case strings.HasPrefix(r.Repository, fmt.Sprintf("%s/", a.repository)):
		name = fmt.Sprintf("%s-%s", name, keyRep.Replace(r.Key()))
	default:
		return "", errors.New("reporting to the artifact can only be sent from the GitHub Actions of the same repository")
	}
	if k := r.RefKey(); k != "" {
		name = fmt.Sprintf("%s%s%s", name, refSeparator, keyRep.Replace(k))
	}
	return name, nil
}

func (a *Artifact) put(ctx context.Context, name, path string, content []byte) error {
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
	return a.gh.PutArtifact(ctx, r.Owner, r.Repo, runID, name, path, content)
}
