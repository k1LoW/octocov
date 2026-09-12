package central

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/datastore"
	"github.com/k1LoW/octocov/datastore/artifact"
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
	ctr.artifactBacked = map[string]bool{"k1LoW/tbls": true}

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

func TestIsArtifact(t *testing.T) {
	if !isArtifact(&artifact.Artifact{}) {
		t.Error("an artifact datastore is what the pages read too")
	}
	d, err := local.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if isArtifact(d) {
		t.Error("a local datastore says nothing about the artifacts of the repositories it holds")
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
