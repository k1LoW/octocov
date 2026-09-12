package central

import (
	"bytes"
	"context"
	"encoding/json"
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
	"github.com/k1LoW/octocov/report"
)

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
		Reports:                []datastore.Datastore{rd},
		CoverageColor:          c.CoverageColor,
		CodeToTestRatioColor:   c.CodeToTestRatioColor,
		TestExecutionTimeColor: c.TestExecutionTimeColor,
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
		Reports:                []datastore.Datastore{rd},
		CoverageColor:          c.CoverageColor,
		CodeToTestRatioColor:   c.CodeToTestRatioColor,
		TestExecutionTimeColor: c.TestExecutionTimeColor,
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
		Reports:                []datastore.Datastore{rd},
		CoverageColor:          c.CoverageColor,
		CodeToTestRatioColor:   c.CodeToTestRatioColor,
		TestExecutionTimeColor: c.TestExecutionTimeColor,
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

func TestRenderIndex(t *testing.T) {
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
		Reports:                []datastore.Datastore{rd},
		CoverageColor:          c.CoverageColor,
		CodeToTestRatioColor:   c.CodeToTestRatioColor,
		TestExecutionTimeColor: c.TestExecutionTimeColor,
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

func (d *artifactStub) Put(_ context.Context, _ string, _ []byte) error { return nil }

func (d *artifactStub) StoreReport(_ context.Context, _ *report.Report) error { return nil }

func (d *artifactStub) FS() (fs.FS, error) { return d.fsys, nil }

func (d *artifactStub) IsArtifact() bool { return true }

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
		Reports:                []datastore.Datastore{rd, &artifactStub{fsys: fsys}},
		CoverageColor:          c.CoverageColor,
		CodeToTestRatioColor:   c.CodeToTestRatioColor,
		TestExecutionTimeColor: c.TestExecutionTimeColor,
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
		Reports:                []datastore.Datastore{rd, &artifactStub{fsys: fsys}},
		CoverageColor:          c.CoverageColor,
		CodeToTestRatioColor:   c.CodeToTestRatioColor,
		TestExecutionTimeColor: c.TestExecutionTimeColor,
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
