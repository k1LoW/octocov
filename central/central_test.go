package central

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
	"testing/fstest"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/datastore/local"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/report"
)

// testBadge is the badge configuration of a central mode repository that customizes nothing.
var testBadge = &config.Badge{}

func TestCollectReports(t *testing.T) {
	c := config.New()
	rd, err := local.New(filepath.Join(testdataDir(t), "reports"))
	if err != nil {
		t.Fatal(err)
	}
	bd, err := local.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctr := New(&Config{
		Repository:             "owner/repo",
		Index:                  ".",
		Wd:                     c.Wd(),
		Badges:                 []datastore.Datastore{bd},
		Reports:                []ReportDatastore{{URL: "local://reports", Datastore: rd}},
		CoverageBadge:          testBadge.RenderCoverage,
		CodeToTestRatioBadge:   testBadge.RenderCodeToTestRatio,
		TestExecutionTimeBadge: testBadge.RenderTestExecutionTime,
	})

	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	got := ctr.reports
	if want := 6; len(got) != want {
		t.Errorf("got %v\nwant %v", len(got), want)
	}
}

func TestCollectReportsSkipsReportsOfOtherRefs(t *testing.T) {
	c := config.New()
	root := t.TempDir()
	for path, r := range map[string]*report.Report{
		filepath.Join("owner", "repo", "report.json"): {
			Repository: "owner/repo",
			Ref:        "refs/heads/main",
			BaseRef:    "refs/heads/main",
			Coverage:   &coverage.Coverage{Total: 100, Covered: 50},
		},
		filepath.Join("owner", "repo", "refs", "pull", "123", "report.json"): {
			Repository:  "owner/repo",
			Ref:         "refs/pull/123/merge",
			BaseRef:     "refs/heads/main",
			PullRequest: 123,
			Coverage:    &coverage.Coverage{Total: 100, Covered: 99},
		},
	} {
		p := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, r.Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rd, err := local.New(root)
	if err != nil {
		t.Fatal(err)
	}
	bd, err := local.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctr := New(&Config{
		Repository:             "owner/repo",
		Index:                  ".",
		Wd:                     c.Wd(),
		Badges:                 []datastore.Datastore{bd},
		Reports:                []ReportDatastore{{URL: "local://reports", Datastore: rd}},
		CoverageBadge:          testBadge.RenderCoverage,
		CodeToTestRatioBadge:   testBadge.RenderCodeToTestRatio,
		TestExecutionTimeBadge: testBadge.RenderTestExecutionTime,
	})

	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	if want := 1; len(ctr.reports) != want {
		t.Fatalf("got %v\nwant %v", len(ctr.reports), want)
	}
	if got, want := ctr.reports[0].Ref, "refs/heads/main"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestGenerateBadges(t *testing.T) {
	c := config.New()
	rd, err := local.New(filepath.Join(testdataDir(t), "reports"))
	if err != nil {
		t.Fatal(err)
	}
	td := t.TempDir()
	bd, err := local.New(td)
	if err != nil {
		t.Fatal(err)
	}
	ctr := New(&Config{
		Repository:             "owner/repo",
		Index:                  ".",
		Wd:                     c.Wd(),
		Badges:                 []datastore.Datastore{bd},
		Reports:                []ReportDatastore{{URL: "local://reports", Datastore: rd}},
		CoverageBadge:          testBadge.RenderCoverage,
		CodeToTestRatioBadge:   testBadge.RenderCodeToTestRatio,
		TestExecutionTimeBadge: testBadge.RenderTestExecutionTime,
	})
	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	paths, err := ctr.generateBadges()
	if err != nil {
		t.Fatal(err)
	}
	if want := 11; len(paths) != want {
		t.Errorf("got %v\nwant %v", len(paths), want)
	}

	var got []string
	if err := filepath.Walk(td, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if fi.IsDir() {
			return nil
		}
		got = append(got, fi.Name())
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	if want := 11; len(got) != want {
		t.Errorf("got %v\nwant %v", len(got), want)
	}
}

// The badges of every collected repository are rendered from the central repository's own
// configuration, since a report carries none of the configuration of the repository it
// describes.
func TestGenerateBadgesUsesTheConfiguredBadge(t *testing.T) {
	c := config.New()
	rd, err := local.New(filepath.Join(testdataDir(t), "reports"))
	if err != nil {
		t.Fatal(err)
	}
	td := t.TempDir()
	bd, err := local.New(td)
	if err != nil {
		t.Fatal(err)
	}
	b := &config.Badge{
		Label: "cov",
		Icon:  "none",
		Colors: []config.BadgeColor{
			{Color: "#123456"},
		},
	}
	ctr := New(&Config{
		Repository:             "owner/repo",
		Index:                  ".",
		Wd:                     c.Wd(),
		Badges:                 []datastore.Datastore{bd},
		Reports:                []ReportDatastore{{URL: "local://reports", Datastore: rd}},
		CoverageBadge:          b.RenderCoverage,
		CodeToTestRatioBadge:   b.RenderCodeToTestRatio,
		TestExecutionTimeBadge: b.RenderTestExecutionTime,
	})
	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}
	if _, err := ctr.generateBadges(); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(filepath.Join(td, ctr.reports[0].Repository, "coverage.svg"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{">cov<", "#123456"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("want to contain %v", want)
		}
	}
	for _, notWant := range []string{">coverage<", "<image"} {
		if strings.Contains(string(got), notWant) {
			t.Errorf("want not to contain %v", notWant)
		}
	}
}

func TestRenderIndex(t *testing.T) {
	// Both the repository column and the badge links are shaped from this, so an ambient
	// value naming another server renders something the golden file cannot match.
	t.Setenv("GITHUB_SERVER_URL", gh.DefaultGithubServerURL)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	c := config.New()
	c.Setwd(filepath.Dir(wd))
	c.Repository = "k1LoW/octocov"
	c.Central = &config.Central{
		Reports: config.CentralReports{
			Datastores: []string{"reports"},
		},
		Badges: config.CentralBadges{
			Datastores: []string{"badges"},
		},
	}
	c.Build()
	rd, err := local.New(filepath.Join(testdataDir(t), "reports"))
	if err != nil {
		t.Fatal(err)
	}
	bd, err := local.New(filepath.Join(c.Wd(), "example/central/badges"))
	if err != nil {
		t.Fatal(err)
	}
	ctr := New(&Config{
		Repository:             c.Repository,
		Index:                  c.Central.Root,
		Wd:                     c.Wd(),
		Badges:                 []datastore.Datastore{bd},
		Reports:                []ReportDatastore{{URL: "local://reports", Datastore: rd}},
		CoverageBadge:          testBadge.RenderCoverage,
		CodeToTestRatioBadge:   testBadge.RenderCodeToTestRatio,
		TestExecutionTimeBadge: testBadge.RenderTestExecutionTime,
	})
	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	if err := ctr.renderIndex(buf); err != nil {
		t.Fatal(err)
	}

	got := buf.String()
	b, err := os.ReadFile(filepath.Join(testdataDir(t), "central_README.md.golden"))
	if err != nil {
		t.Fatal(err)
	}
	want := string(b)

	if got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

// artifactStub stands in for the artifact datastore. The real one lists the artifacts of a
// repository through the GitHub API, which a test cannot reach, and what the collection needs
// of it is only the reports it hands over and the fact that they came from artifacts.
type artifactStub struct {
	fsys fs.FS
}

func (s *artifactStub) Put(_ context.Context, _ string, _ []byte) error { return nil }

func (s *artifactStub) StoreReport(_ context.Context, _ *report.Report) error { return nil }

func (s *artifactStub) FS() (fs.FS, error) { return s.fsys, nil }

func (s *artifactStub) IsArtifact() bool { return true }

func centralReport(repo string, ts time.Time) *report.Report {
	return &report.Report{
		Repository: repo,
		Ref:        "refs/heads/main",
		BaseRef:    "refs/heads/main",
		Coverage:   &coverage.Coverage{Total: 100, Covered: 50},
		Timestamp:  ts,
	}
}

// The index may link only the reports that came from an artifact, so the collection has to
// say which datastore supplied the report it kept, and say it again whenever a later
// datastore supplies a newer one for the same repository.
func TestCollectReportsTracksWhichDatastoreSuppliedTheReport(t *testing.T) {
	c := config.New()
	base := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	root := t.TempDir()
	for repo, ts := range map[string]time.Time{
		"owner/local-only":    base,
		"owner/local-wins":    base.Add(time.Hour),
		"owner/artifact-wins": base,
	} {
		p := filepath.Join(root, repo, "report.json")
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, centralReport(repo, ts).Bytes(), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	rd, err := local.New(root)
	if err != nil {
		t.Fatal(err)
	}
	fsys := fstest.MapFS{}
	for repo, ts := range map[string]time.Time{
		"owner/artifact-only": base,
		"owner/local-wins":    base,
		"owner/artifact-wins": base.Add(time.Hour),
	} {
		fsys[repo+"/report.json"] = &fstest.MapFile{Data: centralReport(repo, ts).Bytes()}
	}
	bd, err := local.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctr := New(&Config{
		Repository:             "owner/repo",
		Index:                  ".",
		Wd:                     c.Wd(),
		Badges:                 []datastore.Datastore{bd},
		Reports:                []ReportDatastore{{URL: "local://reports", Datastore: rd}, {URL: "artifact://owner/repo", Datastore: &artifactStub{fsys: fsys}}},
		CoverageBadge:          testBadge.RenderCoverage,
		CodeToTestRatioBadge:   testBadge.RenderCodeToTestRatio,
		TestExecutionTimeBadge: testBadge.RenderTestExecutionTime,
		BadgeViewer:            report.NewOctocovDevRefViewer(),
	})

	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	want := map[string]bool{
		"owner/local-only":    false,
		"owner/artifact-only": true,
		"owner/local-wins":    false,
		"owner/artifact-wins": true,
	}
	if diff := cmp.Diff(ctr.artifactBacked, want, nil); diff != "" {
		t.Error(diff)
	}
}

// The index links a badge only where the collected report came from an artifact, since the
// pages read a report out of the artifacts of the repository it describes and collecting
// from anywhere else says nothing about whether one is there.
func TestRenderIndexLinksOnlyArtifactBackedReports(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", gh.DefaultGithubServerURL)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	c := config.New()
	c.Setwd(filepath.Dir(wd))
	c.Repository = "k1LoW/octocov"
	c.Central = &config.Central{
		Reports: config.CentralReports{
			Datastores: []string{"reports"},
		},
		Badges: config.CentralBadges{
			Datastores: []string{"badges"},
		},
	}
	c.Build()
	rd, err := local.New(filepath.Join(testdataDir(t), "reports"))
	if err != nil {
		t.Fatal(err)
	}
	bd, err := local.New(filepath.Join(c.Wd(), "example/central/badges"))
	if err != nil {
		t.Fatal(err)
	}
	// The same shape as testdata/octocov_central.yml, where one repository is read out of an
	// artifact while the rest come from a local datastore.
	b, err := os.ReadFile(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json"))
	if err != nil {
		t.Fatal(err)
	}
	tbls := &report.Report{}
	if err := json.Unmarshal(b, tbls); err != nil {
		t.Fatal(err)
	}
	tbls.Timestamp = time.Now()
	fsys := fstest.MapFS{"k1LoW/tbls/report.json": &fstest.MapFile{Data: tbls.Bytes()}}
	ctr := New(&Config{
		Repository:             c.Repository,
		Index:                  c.Central.Root,
		Wd:                     c.Wd(),
		Badges:                 []datastore.Datastore{bd},
		Reports:                []ReportDatastore{{URL: "local://reports", Datastore: rd}, {URL: "artifact://owner/repo", Datastore: &artifactStub{fsys: fsys}}},
		CoverageBadge:          testBadge.RenderCoverage,
		CodeToTestRatioBadge:   testBadge.RenderCodeToTestRatio,
		TestExecutionTimeBadge: testBadge.RenderTestExecutionTime,
		BadgeViewer:            report.NewOctocovDevRefViewer(),
	})
	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}
	if err := ctr.renderIndex(buf); err != nil {
		t.Fatal(err)
	}
	got := buf.String()

	// The three badges of the row and the three the copy snippet offers.
	if n, want := strings.Count(got, "](https://octocov.dev/k1LoW/tbls)"), 6; n != want {
		t.Errorf("got %v\nwant %v", n, want)
	}
	if n, want := strings.Count(got, "octocov.dev"), 6; n != want {
		t.Errorf("got %v\nwant %v", n, want)
	}
}

func testdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(wd), "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

// failingStub stands in for a datastore that cannot be read, which is what a missing
// permission, an expired artifact or an unreachable bucket amounts to here.
type failingStub struct{ err error }

func (s *failingStub) Put(_ context.Context, _ string, _ []byte) error { return nil }

func (s *failingStub) StoreReport(_ context.Context, _ *report.Report) error { return nil }

func (s *failingStub) FS() (fs.FS, error) { return nil, s.err }

// One datastore that cannot be read must not cost the index the repositories the others
// describe, and the warning has to name the datastore, since an index silently short of a
// repository is what this replaced.
func TestCollectReportsWarnsAndContinuesWhenADatastoreCannotBeRead(t *testing.T) {
	c := config.New()
	fsys := fstest.MapFS{
		"owner/readable/report.json": &fstest.MapFile{Data: centralReport("owner/readable", time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)).Bytes()},
	}
	bd, err := local.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctr := New(&Config{
		Repository: "owner/repo",
		Index:      ".",
		Wd:         c.Wd(),
		Badges:     []datastore.Datastore{bd},
		Reports: []ReportDatastore{
			{URL: "artifact://owner/unreachable", Datastore: &failingStub{err: errors.New("artifact not found")}},
			{URL: "artifact://owner/readable", Datastore: &artifactStub{fsys: fsys}},
		},
		CoverageBadge:          testBadge.RenderCoverage,
		CodeToTestRatioBadge:   testBadge.RenderCodeToTestRatio,
		TestExecutionTimeBadge: testBadge.RenderTestExecutionTime,
	})
	warned := new(bytes.Buffer)
	ctr.stderr = warned

	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(ctr.reports))
	for _, r := range ctr.reports {
		got = append(got, r.Repository)
	}
	if diff := cmp.Diff(got, []string{"owner/readable"}, nil); diff != "" {
		t.Error(diff)
	}
	if want := "Skip collecting reports from artifact://owner/unreachable: artifact not found"; !strings.Contains(warned.String(), want) {
		t.Errorf("got %v\nwant to contain %v", warned.String(), want)
	}
}

// The index is rewritten from what was collected, so every datastore failing would empty it
// of every repository at once. That is the outcome the warnings exist to prevent going
// unnoticed, so it is an error rather than one more warning.
func TestCollectReportsFailsWhenNoDatastoreCanBeRead(t *testing.T) {
	c := config.New()
	bd, err := local.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctr := New(&Config{
		Repository: "owner/repo",
		Index:      ".",
		Wd:         c.Wd(),
		Badges:     []datastore.Datastore{bd},
		Reports: []ReportDatastore{
			{URL: "artifact://owner/repo-1", Datastore: &failingStub{err: errors.New("artifact not found")}},
			{URL: "artifact://owner/repo-2", Datastore: &failingStub{err: errors.New("403 Forbidden")}},
		},
		CoverageBadge:          testBadge.RenderCoverage,
		CodeToTestRatioBadge:   testBadge.RenderCodeToTestRatio,
		TestExecutionTimeBadge: testBadge.RenderTestExecutionTime,
	})
	ctr.stderr = new(bytes.Buffer)

	if err := ctr.collectReports(); err == nil {
		t.Error("want error when no datastore could be read")
	}
}

// walkErrorFS hands over the reports it holds until the walk reaches failDir, which it
// refuses. A datastore listing a bucket or a branch fails this way rather than at FS(),
// after some of what it holds has already been read.
type walkErrorFS struct {
	fs.FS
	failDir string
	err     error
}

func (f *walkErrorFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == f.failDir {
		return nil, f.err
	}
	return fs.ReadDir(f.FS, name)
}

// A walk that stops partway still reached some repositories, and those belong in the index.
// Dropping them would be the blank index this whole change is about, arrived at from the
// other side, so the run carries on with what it has and only says what it missed.
func TestCollectReportsKeepsWhatAWalkReachedBeforeItFailed(t *testing.T) {
	c := config.New()
	ts := time.Date(2026, 9, 13, 0, 0, 0, 0, time.UTC)
	// Walked in name order, so the readable directory is reached before the one that fails.
	base := fstest.MapFS{
		"owner/collected/report.json":  &fstest.MapFile{Data: centralReport("owner/collected", ts).Bytes()},
		"owner/unreadable/report.json": &fstest.MapFile{Data: centralReport("owner/unreadable", ts).Bytes()},
	}
	bd, err := local.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	ctr := New(&Config{
		Repository: "owner/repo",
		Index:      ".",
		Wd:         c.Wd(),
		Badges:     []datastore.Datastore{bd},
		Reports: []ReportDatastore{
			{URL: "s3://bucket/reports", Datastore: &artifactStub{fsys: &walkErrorFS{FS: base, failDir: "owner/unreadable", err: errors.New("AccessDenied")}}},
		},
		CoverageBadge:          testBadge.RenderCoverage,
		CodeToTestRatioBadge:   testBadge.RenderCodeToTestRatio,
		TestExecutionTimeBadge: testBadge.RenderTestExecutionTime,
	})
	warned := new(bytes.Buffer)
	ctr.stderr = warned

	// The only datastore configured is the one that failed, and a report was still collected
	// from it, so the index has something to be written from and the run is not an error.
	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	got := make([]string, 0, len(ctr.reports))
	for _, r := range ctr.reports {
		got = append(got, r.Repository)
	}
	if diff := cmp.Diff(got, []string{"owner/collected"}, nil); diff != "" {
		t.Error(diff)
	}
	if want := "Skip collecting reports from s3://bucket/reports: AccessDenied"; !strings.Contains(warned.String(), want) {
		t.Errorf("got %v\nwant to contain %v", warned.String(), want)
	}
}
