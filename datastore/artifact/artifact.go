package artifact

import (
	"context"
	"errors"
	"fmt"
	"io"
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

// client is the part of *gh.Gh this datastore uses. It is an interface so a test can stand
// in for the upload, which otherwise needs the runtime of a GitHub Actions job.
type client interface {
	PutArtifact(ctx context.Context, owner, repo string, runID int64, name, fp string, content []byte) error
	DeleteArtifactsBeforeRun(ctx context.Context, owner, repo, name string, runID int64) error
	FetchLatestArtifact(ctx context.Context, owner, repo, name, fp string) (*gh.ArtifactFile, error)
}

type Artifact struct {
	gh         client
	repository string
	name       string
	r          *report.Report
	// stderr is where a failure to delete the previous reports of a pull request goes. It
	// is a field so a test can read it back.
	stderr io.Writer
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
		stderr:     os.Stderr,
	}, nil
}

// IsArtifact reports that this datastore reads its reports out of GitHub Actions artifacts,
// which is the same place the pages that browse a report read it from. It is stated as a
// method so a caller can ask it of any datastore.Datastore without naming this type, which
// nothing standing in for it while reading the API is out of reach could satisfy.
func (a *Artifact) IsArtifact() bool {
	return true
}

func (a *Artifact) StoreReport(ctx context.Context, r *report.Report) error {
	name, err := a.StoreName(r)
	if err != nil {
		return err
	}
	if err := a.put(ctx, name, reportFilename, r.Bytes()); err != nil {
		return err
	}
	if r.PullRequest == 0 {
		// The report of a branch can be the base a pull request is compared against, which
		// is picked by the merge base commit rather than by being the newest, so the older
		// ones are still read.
		return nil
	}
	if err := a.deletePrevious(ctx, name); err != nil {
		// Deleting needs actions: write, which a workflow may not grant and a pull request
		// from a fork never has. The report is stored by now, so failing the run over the
		// artifacts it leaves behind would be worse than leaving them.
		fmt.Fprintf(a.stderr, "Skip deleting the previous reports of %s: %v\n", name, err) //nostyle:handlerrors
	}
	return nil
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
	if err != nil {
		// An empty filesystem used to stand in for every failure here, which left a missing
		// permission and an expired artifact looking exactly like a repository that had
		// never reported. Saying so is what lets the caller decide, and the central mode
		// turns it into a warning rather than stopping.
		return nil, fmt.Errorf("failed to fetch artifact %s of %s/%s: %w", name, r.Owner, r.Repo, err)
	}
	fsys := fstest.MapFS{
		path: &fstest.MapFile{
			Data:    af.Content,
			Mode:    fs.ModePerm,
			ModTime: af.CreatedAt,
		},
	}
	return &fsys, nil
}

// StoreName returns the artifact name the report is stored under. The report of the
// default branch keeps the configured name, which is the one FS() looks up, so a
// comparison and the central mode keep reading the branch they are meant to describe.
func (a *Artifact) StoreName(r *report.Report) (string, error) {
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
	runID, err := currentRunID()
	if err != nil {
		return err
	}
	return a.gh.PutArtifact(ctx, r.Owner, r.Repo, runID, name, path, content)
}

// deletePrevious deletes the artifacts of the name that earlier runs uploaded. Nothing reads
// any but the newest, and unlike a datastore writing to a path fixed by the ref, a later
// upload does not replace them.
func (a *Artifact) deletePrevious(ctx context.Context, name string) error {
	r, err := gh.Parse(a.repository)
	if err != nil {
		return err
	}
	runID, err := currentRunID()
	if err != nil {
		return err
	}
	return a.gh.DeleteArtifactsBeforeRun(ctx, r.Owner, r.Repo, name, runID)
}

func currentRunID() (int64, error) {
	s := os.Getenv("GITHUB_RUN_ID")
	if s == "" {
		return 0, errors.New("env GITHUB_RUN_ID is not set")
	}
	runID, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse GITHUB_RUN_ID: %w", err)
	}
	return runID, nil
}
