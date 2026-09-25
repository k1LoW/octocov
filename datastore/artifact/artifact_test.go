package artifact

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

func TestStoreName(t *testing.T) {
	tests := []struct {
		name        string
		artifact    string
		repository  string
		ref         string
		pullRequest int
		want        string
		wantErr     bool
	}{
		{"the default branch keeps the configured name", "", "owner/repo", "refs/heads/main", 0, "octocov-report", false},
		{"a pull request keeps the configured name too", "", "owner/repo", "refs/pull/123/merge", 123, "octocov-report", false},
		{"a branch keeps the configured name too", "", "owner/repo", "refs/heads/feat/x", 0, "octocov-report", false},
		{"a configured name is kept as it is", "mine", "owner/repo", "refs/pull/123/merge", 123, "mine", false},
		{"a sub directory carries its key", "", "owner/repo/sub", "refs/pull/123/merge", 123, "octocov-report-sub", false},
		{"a report of another repository is refused", "", "other/repo", "refs/heads/main", 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_REPOSITORY", "owner/repo")
			a, err := New(nil, "owner/repo", tt.artifact, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := a.StoreName(&report.Report{
				Repository:  tt.repository,
				Ref:         tt.ref,
				BaseRef:     "refs/heads/main",
				PullRequest: tt.pullRequest,
			})
			if err != nil {
				if !tt.wantErr {
					t.Errorf("got err: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want err")
				return
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestMetadataName(t *testing.T) {
	tests := []struct {
		name string
		ref  string
		want string
	}{
		{"octocov-report", "refs/pull/123", "octocov-metadata-octocov-report@refs_pull_123"},
		{"octocov-report", "refs/heads/feat/x", "octocov-metadata-octocov-report@refs_heads_feat_x"},
		{"octocov-report-sub", "refs/heads/main", "octocov-metadata-octocov-report-sub@refs_heads_main"},
		// Split on the last separator, the ref never carrying one.
		{"mine@v2", "refs/heads/main", "octocov-metadata-mine@v2@refs_heads_main"},
	}
	for _, tt := range tests {
		if got := MetadataName(tt.name, tt.ref); got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestMetadataNameHasNoInvalidCharacter(t *testing.T) {
	got := MetadataName("octocov-report", report.NormalizeRef(`refs/heads/feat/"a:b<c>d|e*f?g\h`))
	// The characters GitHub refuses in an artifact name, which a ref may hold.
	for _, c := range []string{`"`, ":", "<", ">", "|", "*", "?", "\r", "\n", `\`, "/"} {
		if strings.Contains(got, c) {
			t.Errorf("got %v\nwant no %q", got, c)
		}
	}
}

// fakeClient stands in for the GitHub API, holding the artifacts uploaded to it by name.
type fakeClient struct {
	nextID        int64
	uploaded      map[string]*gh.ArtifactFile
	byID          map[int64]*gh.ArtifactFile
	branches      map[int64]string
	defaultBranch string
	askedBranch   string
}

func newFakeClient() *fakeClient {
	return &fakeClient{
		uploaded:      map[string]*gh.ArtifactFile{},
		byID:          map[int64]*gh.ArtifactFile{},
		branches:      map[int64]string{},
		defaultBranch: "main",
	}
}

func (c *fakeClient) PutArtifact(ctx context.Context, owner, repo string, runID int64, name, fp string, content []byte) error {
	c.add(name, fp, "", content)
	return nil
}

func (c *fakeClient) FetchRunArtifactID(ctx context.Context, owner, repo string, runID int64, name string) (int64, error) {
	af, ok := c.uploaded[name]
	if !ok {
		return 0, gh.ErrArtifactNotFound
	}
	return af.ID, nil
}

func (c *fakeClient) FetchLatestArtifact(ctx context.Context, owner, repo, name, fp string) (*gh.ArtifactFile, error) {
	return c.FetchLatestArtifactOfBranch(ctx, owner, repo, name, fp, "")
}

func (c *fakeClient) FetchLatestArtifactOfBranch(ctx context.Context, owner, repo, name, fp, branch string) (*gh.ArtifactFile, error) {
	c.askedBranch = branch
	af, ok := c.uploaded[name]
	if !ok || af.Name != fp || (branch != "" && c.branches[af.ID] != branch) {
		return nil, gh.ErrArtifactNotFound
	}
	return af, nil
}

func (c *fakeClient) FetchArtifact(ctx context.Context, owner, repo string, id int64, fp string) (*gh.ArtifactFile, error) {
	af, ok := c.byID[id]
	if !ok || af.Name != fp {
		return nil, gh.ErrArtifactNotFound
	}
	return af, nil
}

func (c *fakeClient) FetchDefaultBranch(ctx context.Context, owner, repo string) (string, error) {
	return c.defaultBranch, nil
}

func (c *fakeClient) add(name, fp, branch string, content []byte) int64 {
	c.nextID++
	af := &gh.ArtifactFile{ID: c.nextID, Name: fp, Content: content}
	c.uploaded[name] = af
	c.byID[c.nextID] = af
	c.branches[c.nextID] = branch
	return c.nextID
}

func TestStoreReportPutsTheMetadataOfTheRef(t *testing.T) {
	tests := []struct {
		name         string
		report       *report.Report
		event        string
		wantMetadata string
		want         Metadata
	}{
		{
			"a pull request points at the head of the pull request, not at the merge commit measured",
			&report.Report{Repository: "owner/repo", Ref: "refs/pull/123/merge", BaseRef: "refs/heads/main", PullRequest: 123, Commit: "merge"},
			`{"pull_request":{"number":123,"head":{"sha":"head"}}}`,
			"octocov-metadata-octocov-report@refs_pull_123",
			Metadata{Ref: "refs/pull/123", Commit: "merge", HeadCommit: "head"},
		},
		{
			"a pull request whose event says nothing of it leaves the head out",
			&report.Report{Repository: "owner/repo", Ref: "refs/pull/123/merge", BaseRef: "refs/heads/main", PullRequest: 123, Commit: "merge"},
			`{}`,
			"octocov-metadata-octocov-report@refs_pull_123",
			Metadata{Ref: "refs/pull/123", Commit: "merge"},
		},
		{
			"a branch is at the commit measured",
			&report.Report{Repository: "owner/repo", Ref: "refs/heads/main", BaseRef: "refs/heads/main", Commit: "abc"},
			`{}`,
			"octocov-metadata-octocov-report@refs_heads_main",
			Metadata{Ref: "refs/heads/main", Commit: "abc", HeadCommit: "abc"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_REPOSITORY", "owner/repo")
			t.Setenv("GITHUB_RUN_ID", "10")
			p := filepath.Join(t.TempDir(), "event.json")
			if err := os.WriteFile(p, []byte(tt.event), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("GITHUB_EVENT_NAME", "pull_request")
			t.Setenv("GITHUB_EVENT_PATH", p)
			c := newFakeClient()
			a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName}
			if err := a.StoreReport(t.Context(), tt.report); err != nil {
				t.Fatal(err)
			}
			stored, ok := c.uploaded["octocov-report"]
			if !ok {
				t.Fatal("want the report stored under the configured name")
			}
			mf, ok := c.uploaded[tt.wantMetadata]
			if !ok {
				t.Fatalf("want the metadata stored as %s", tt.wantMetadata)
			}
			got := Metadata{}
			if err := json.Unmarshal(mf.Content, &got); err != nil {
				t.Fatal(err)
			}
			want := tt.want
			want.Report.ArtifactName = "octocov-report"
			want.Report.ArtifactID = stored.ID
			if diff := cmp.Diff(got, want); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestStoreReportSkipsTheMetadataOfARefTooLongToName(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	t.Setenv("GITHUB_RUN_ID", "10")
	c := newFakeClient()
	stderr := new(bytes.Buffer)
	a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName, stderr: stderr}
	ref := "refs/heads/" + strings.Repeat("a", 230)
	if err := a.StoreReport(t.Context(), &report.Report{Repository: "owner/repo", Ref: ref, BaseRef: "refs/heads/main", Commit: "abc"}); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.uploaded["octocov-report"]; !ok {
		t.Error("want the report stored all the same")
	}
	if len(c.uploaded) != 1 {
		t.Errorf("got %d artifacts\nwant the report alone", len(c.uploaded))
	}
	if want := "Skip storing the metadata of " + ref; !strings.Contains(stderr.String(), want) {
		t.Errorf("got %q\nwant it to contain %q", stderr.String(), want)
	}
}

func TestFSReadsTheReportTheMetadataOfTheBaseRefPointsAt(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	c := newFakeClient()
	base := c.add("octocov-report", reportFilename, "main", []byte(`{"commit":"base"}`))
	// A pull request stored the newer report under the same name.
	c.add("octocov-report", reportFilename, "feat", []byte(`{"commit":"pull"}`))
	m := &Metadata{Ref: "refs/heads/main", Commit: "base"}
	m.Report.ArtifactName = "octocov-report"
	m.Report.ArtifactID = base
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	c.add("octocov-metadata-octocov-report@refs_heads_main", metadataFilename, "main", b)

	a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName, r: &report.Report{Repository: "owner/repo", BaseRef: "refs/heads/main"}}
	got := readReport(t, a)
	if want := `{"commit":"base"}`; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	if want := "octocov-metadata-octocov-report@refs_heads_main"; a.MetadataRead() != want {
		t.Errorf("got %v\nwant %v", a.MetadataRead(), want)
	}
}

func TestFSFallsBackToTheNewestReportOfTheBaseBranch(t *testing.T) {
	// A ref an older octocov stored has no metadata, so the report is looked for among the
	// artifacts the runs on its branch uploaded.
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	c := newFakeClient()
	c.add("octocov-report", reportFilename, "develop", []byte(`{"commit":"base"}`))
	a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName, r: &report.Report{Repository: "owner/repo", BaseRef: "refs/heads/develop"}}
	got := readReport(t, a)
	if want := `{"commit":"base"}`; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	if want := "develop"; c.askedBranch != want {
		t.Errorf("got %v\nwant %v", c.askedBranch, want)
	}
	// Linking the metadata of the ref would open the report it points at, not this one.
	if got := a.MetadataRead(); got != "" {
		t.Errorf("got %v\nwant no metadata", got)
	}
}

func TestFSFallsBackWhereTheMetadataPointsAtAReportGone(t *testing.T) {
	// A re-run deletes the report the earlier attempt stored, and an attempt failing before its
	// metadata is stored leaves the earlier metadata pointing at it.
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	c := newFakeClient()
	c.add("octocov-report", reportFilename, "main", []byte(`{"commit":"base"}`))
	m := &Metadata{Ref: "refs/heads/main", Commit: "gone"}
	m.Report.ArtifactName = "octocov-report"
	m.Report.ArtifactID = 999
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	c.add("octocov-metadata-octocov-report@refs_heads_main", metadataFilename, "main", b)

	a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName, r: &report.Report{Repository: "owner/repo", BaseRef: "refs/heads/main"}}
	got := readReport(t, a)
	if want := `{"commit":"base"}`; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	if want := "main"; c.askedBranch != want {
		t.Errorf("got %v\nwant %v", c.askedBranch, want)
	}
	// Linking the metadata of the ref would open the report it points at, not this one.
	if got := a.MetadataRead(); got != "" {
		t.Errorf("got %v\nwant no metadata", got)
	}
}

func TestFSFallsBackWhereTheMetadataIsOfAnotherRefSharingTheName(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	c := newFakeClient()
	// The fake holds only the newest artifact of a name, which the fallback has to find.
	other := c.add("octocov-report", reportFilename, "release_v1", []byte(`{"commit":"other"}`))
	c.add("octocov-report", reportFilename, "release/v1", []byte(`{"commit":"base"}`))
	m := &Metadata{Ref: "refs/heads/release_v1", Commit: "other"}
	m.Report.ArtifactName = "octocov-report"
	m.Report.ArtifactID = other
	b, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	// refs/heads/release/v1 and refs/heads/release_v1 name the same metadata artifact.
	c.add("octocov-metadata-octocov-report@refs_heads_release_v1", metadataFilename, "release_v1", b)

	a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName, r: &report.Report{Repository: "owner/repo", BaseRef: "refs/heads/release/v1"}}
	got := readReport(t, a)
	if want := `{"commit":"base"}`; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	if got := a.MetadataRead(); got != "" {
		t.Errorf("got %v\nwant no metadata", got)
	}
}

func TestFSReadsTheDefaultBranchWithoutAReport(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	// As the central mode does, which has no report of its own to be compared.
	c := newFakeClient()
	c.defaultBranch = "develop"
	c.add("octocov-report", reportFilename, "develop", []byte(`{"commit":"base"}`))
	a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName}
	readReport(t, a)
	if want := "develop"; c.askedBranch != want {
		t.Errorf("got %v\nwant %v", c.askedBranch, want)
	}
}

func TestFSReadsTheBaseBranchOfAPullRequest(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	c := newFakeClient()
	c.add("octocov-report", reportFilename, "develop", []byte(`{"commit":"develop"}`))
	a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName, r: &report.Report{Repository: "owner/repo", Ref: "refs/pull/123/merge", BaseRef: "refs/heads/develop", PullRequest: 123}}
	// Under the key of the base branch, as the datastores storing by path hold it.
	got := readReportAt(t, a, "owner/repo/refs/heads/develop/"+reportFilename)
	if want := `{"commit":"develop"}`; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestFSFallsBackToTheDefaultBranchWhereTheBaseBranchHasNoReport(t *testing.T) {
	// As in a repository reporting only from the default branch.
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	c := newFakeClient()
	c.add("octocov-report", reportFilename, "main", []byte(`{"commit":"main"}`))
	a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName, r: &report.Report{Repository: "owner/repo", Ref: "refs/pull/123/merge", BaseRef: "refs/heads/develop", PullRequest: 123}}
	got := readReport(t, a)
	if want := `{"commit":"main"}`; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	if want := "main"; c.askedBranch != want {
		t.Errorf("got %v\nwant %v", c.askedBranch, want)
	}
}

func TestFSReportsNothingWhereNeitherBranchHasAReport(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	c := newFakeClient()
	a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName, r: &report.Report{Repository: "owner/repo", Ref: "refs/pull/123/merge", BaseRef: "refs/heads/main", PullRequest: 123}}
	if _, err := a.FS(); !errors.Is(err, gh.ErrArtifactNotFound) {
		t.Errorf("got %v\nwant %v", err, gh.ErrArtifactNotFound)
	}
}

func readReport(t *testing.T, a *Artifact) string {
	t.Helper()
	return readReportAt(t, a, "owner/repo/"+reportFilename)
}

func readReportAt(t *testing.T, a *Artifact, path string) string {
	t.Helper()
	fsys, err := a.FS()
	if err != nil {
		t.Fatal(err)
	}
	b, err := fs.ReadFile(fsys, path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
