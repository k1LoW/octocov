package artifact

import (
	"context"
	"encoding/json"
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

// metadataPrefix and metadataFilename name the artifact that says which artifact holds the
// report a ref stored last. The reports keep the name they are configured with, so the
// artifacts of one name hold the reports of every ref, and the metadata is what tells them
// apart without the report having to be stored under a name octocov makes up.
const (
	metadataPrefix   = "octocov-metadata"
	metadataFilename = "metadata.json"
)

// MaxNameLength is the longest name GitHub Actions accepts for an artifact.
const MaxNameLength = 255

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
	FetchRunArtifactID(ctx context.Context, owner, repo string, runID int64, name string) (int64, error)
	FetchLatestArtifact(ctx context.Context, owner, repo, name, fp string) (*gh.ArtifactFile, error)
	FetchLatestArtifactOfBranch(ctx context.Context, owner, repo, name, fp, branch string) (*gh.ArtifactFile, error)
	FetchArtifact(ctx context.Context, owner, repo string, id int64, fp string) (*gh.ArtifactFile, error)
	FetchDefaultBranch(ctx context.Context, owner, repo string) (string, error)
}

// Metadata is what the metadata artifact of a ref holds.
type Metadata struct {
	// Ref is the ref the run that stored the report was on, as Report.RunRef names it.
	Ref    string `json:"ref"`
	Commit string `json:"commit"`
	// HeadCommit is the commit the ref was at, which a reader compares against where the ref
	// is now to tell whether the report is of it. On a pull request it is the head of the pull
	// request, since Commit is the merge commit the run measured, which GitHub makes again
	// whenever the base moves. It is empty where the event does not say.
	HeadCommit string `json:"head_commit,omitempty"`
	Report     struct {
		ArtifactName string `json:"artifact_name"`
		ArtifactID   int64  `json:"artifact_id"`
	} `json:"report"`
}

type Artifact struct {
	gh         client
	repository string
	name       string
	r          *report.Report
	// metadataRead is the name of the metadata artifact the last FS() read the report
	// through, and empty where it read the branch fallback instead.
	metadataRead string
	// stderr is where storing no metadata for a ref whose name is too long is said. It is a
	// field so a test can read it back.
	stderr io.Writer
}

func New(g *gh.Gh, repo, name string, r *report.Report) (*Artifact, error) {
	if name == "" {
		name = defaultArtifactName
	}
	a := &Artifact{
		repository: repo,
		name:       name,
		r:          r,
		stderr:     os.Stderr,
	}
	// Left nil rather than holding a nil *gh.Gh, which as a client would compare unequal
	// to nil and fail only once it is called.
	if g != nil {
		a.gh = g
	}
	return a, nil
}

// MetadataRead returns the name of the metadata artifact the last FS() read the report through,
// or an empty name where no metadata led to it. A link naming metadata is only good for a
// report some metadata points at, and the report the branch fallback reads is not one.
func (a *Artifact) MetadataRead() string {
	return a.metadataRead
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
	ref := r.RunRef()
	if ref == "" {
		return nil
	}
	// A shortened name would be one more rule every reader has to share. Without the metadata
	// the readers take the branch fallback, which reads the same report, so the report stored
	// is worth more than failing the run over it.
	if n := MetadataName(name, ref); len(n) > MaxNameLength {
		fmt.Fprintf(a.stderr, "Skip storing the metadata of %s: the artifact name %s is longer than %d characters\n", ref, n, MaxNameLength) //nostyle:handlerrors
		return nil
	}
	return a.putMetadata(ctx, name, ref, r)
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
	af, key, err := a.fetchBaseReport(ctx, r, name)
	if err != nil {
		// An empty filesystem used to stand in for every failure here, which left a missing
		// permission and an expired artifact looking exactly like a repository that had
		// never reported. Saying so is what lets the caller decide, and the central mode
		// turns it into a warning rather than stopping.
		return nil, fmt.Errorf("failed to fetch artifact %s of %s/%s: %w", name, r.Owner, r.Repo, err)
	}
	// Laid out under the key of the ref it was read for, as the datastores storing by path
	// hold it, so that the comparison can prefer the report of the base branch across every
	// datastore rather than taking this one for the base branch's whichever ref it is of.
	if key != "" {
		path = fmt.Sprintf("%s/%s/%s/%s", r.Owner, r.Reponame(), key, reportFilename)
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

// StoreName returns the artifact name the report is stored under, which is the configured
// one, with the key of a report of a sub directory appended. The reports of every ref share
// it, and the metadata of each ref says which of them is its own.
func (a *Artifact) StoreName(r *report.Report) (string, error) {
	name := a.name
	switch {
	case a.repository == r.Repository:
	case strings.HasPrefix(r.Repository, fmt.Sprintf("%s/", a.repository)):
		name = fmt.Sprintf("%s-%s", name, keyRep.Replace(r.Key()))
	default:
		return "", errors.New("reporting to the artifact can only be sent from the GitHub Actions of the same repository")
	}
	return name, nil
}

// putMetadata stores the metadata of ref, pointing at the artifact of the name this run has
// just stored the report in. It points by ID, since the reports of every ref share the name.
func (a *Artifact) putMetadata(ctx context.Context, name, ref string, r *report.Report) error {
	repo, err := gh.Parse(a.repository)
	if err != nil {
		return err
	}
	runID, err := currentRunID()
	if err != nil {
		return err
	}
	id, err := a.gh.FetchRunArtifactID(ctx, repo.Owner, repo.Repo, runID, name)
	if err != nil {
		return err
	}
	m := &Metadata{Ref: ref, Commit: r.Commit, HeadCommit: headCommit(r)}
	m.Report.ArtifactName = name
	m.Report.ArtifactID = id
	b, err := json.Marshal(m)
	if err != nil {
		return err
	}
	return a.put(ctx, MetadataName(name, ref), metadataFilename, b)
}

// headCommit returns the commit the ref of the run that took r was at.
func headCommit(r *report.Report) string {
	if r.PullRequest == 0 {
		return r.Commit
	}
	e, err := gh.DecodeGitHubEvent()
	if err != nil || e.Number != r.PullRequest {
		return ""
	}
	return e.HeadSHA
}

// fetchReport returns the report of the artifacts of the name that ref stored last, which is
// the one its metadata points at. A ref with no metadata, which is every ref an octocov older
// than the metadata stored, or whose metadata points at a report no longer there, is read from
// the newest artifact of the name a run on its branch uploaded, so the reports of the other
// refs sharing the name stay out.
func (a *Artifact) fetchReport(ctx context.Context, r *gh.Repository, name, ref string) (*gh.ArtifactFile, error) {
	a.metadataRead = ""
	metadataName := MetadataName(name, ref)
	mf, err := a.gh.FetchLatestArtifact(ctx, r.Owner, r.Repo, metadataName, metadataFilename)
	switch {
	case err == nil:
		m := &Metadata{}
		if err := json.Unmarshal(mf.Content, m); err != nil {
			return nil, fmt.Errorf("failed to read the metadata of %s: %w", ref, err)
		}
		// The name carries the ref with its separators replaced, so two refs such as
		// refs/heads/release/v1 and refs/heads/release_v1 share it, and the newest metadata of
		// the name can be the other ref's. The fallback compares the branch unreplaced.
		if m.Ref != ref {
			break
		}
		af, err := a.gh.FetchArtifact(ctx, r.Owner, r.Repo, m.Report.ArtifactID, reportFilename)
		// The report can be gone while its metadata is not. A re-run keeps the run id, and the
		// upload of the report deletes the one the earlier attempt stored first, so an attempt
		// that fails before storing the metadata leaves the earlier metadata pointing at nothing.
		if err == nil {
			a.metadataRead = metadataName
		}
		if !errors.Is(err, gh.ErrArtifactNotFound) {
			return af, err
		}
	case errors.Is(err, gh.ErrArtifactNotFound):
	default:
		return nil, err
	}
	branch, _ := strings.CutPrefix(ref, "refs/heads/")
	return a.gh.FetchLatestArtifactOfBranch(ctx, r.Owner, r.Repo, name, reportFilename, branch)
}

// fetchBaseReport returns the report the report of this run is compared against, and the key
// of the ref it was read for, as report.BaseKeys orders them. A ref with nothing to read gives
// way to the next, and the default branch, the empty key, is also what is read where there is
// no report of this run, as in the central mode.
func (a *Artifact) fetchBaseReport(ctx context.Context, r *gh.Repository, name string) (*gh.ArtifactFile, string, error) {
	keys := []string{""}
	if a.r != nil {
		keys = a.r.BaseKeys()
	}
	var (
		tried    string
		notFound error
	)
	for _, key := range keys {
		ref := key
		if ref == "" {
			var err error
			ref, err = a.defaultRef(ctx, r)
			if err != nil {
				return nil, "", err
			}
		}
		// The base branch is the default branch, already found to have nothing.
		if ref == tried {
			continue
		}
		af, err := a.fetchReport(ctx, r, name, ref)
		if !errors.Is(err, gh.ErrArtifactNotFound) {
			return af, key, err
		}
		tried, notFound = ref, err
	}
	return nil, "", notFound
}

// defaultRef returns the ref of the default branch. Outside a pull request the base ref of
// the run is the default branch, so the API is asked only where the run knows neither.
func (a *Artifact) defaultRef(ctx context.Context, r *gh.Repository) (string, error) {
	if a.r != nil {
		if d := a.r.DefaultBranch(); d != "" {
			return d, nil
		}
		if base := report.NormalizeRef(a.r.BaseRef); base != "" && !strings.HasPrefix(a.r.RunRef(), "refs/pull/") {
			return base, nil
		}
	}
	b, err := a.gh.FetchDefaultBranch(ctx, r.Owner, r.Repo)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("refs/heads/%s", b), nil
}

// RefScopedName returns name marked off by ref, or name itself when ref is empty.
func RefScopedName(name, ref string) string {
	if ref == "" {
		return name
	}
	return fmt.Sprintf("%s%s%s", name, refSeparator, keyRep.Replace(ref))
}

// MetadataName returns the name of the artifact holding the metadata of ref for the reports
// stored under the artifact name. The name is carried whole, since a run can store a report
// under more than one name and a run holds one artifact of a name.
func MetadataName(name, ref string) string {
	return RefScopedName(fmt.Sprintf("%s-%s", metadataPrefix, name), ref)
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
