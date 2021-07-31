package central

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/k1LoW/octocov/config"
)

func TestCollectReports(t *testing.T) {
	c := config.New()
	c.Central = &config.ConfigCentral{
		Enable:  true,
		Reports: filepath.Join(testdataDir(t), "reports"),
	}

	ctr := New(&CentralConfig{
		Repository:             c.Repository,
		Index:                  c.Central.Root,
		Wd:                     c.Getwd(),
		Badges:                 c.Central.Badges,
		Reports:                c.Central.Reports,
		CoverageColor:          c.CoverageColor,
		CodeToTestRatioColor:   c.CodeToTestRatioColor,
		TestExecutionTimeColor: c.TestExecutionTimeColor,
	})

	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	got := ctr.reports
	if want := 5; len(got) != want {
		t.Errorf("got %v\nwant %v", len(got), want)
	}
}

func TestGenerateBadges(t *testing.T) {
	bd := t.TempDir()
	c := config.New()
	c.Central = &config.ConfigCentral{
		Enable:  true,
		Reports: filepath.Join(testdataDir(t), "reports"),
		Badges:  bd,
	}

	ctr := New(&CentralConfig{
		Repository:             c.Repository,
		Index:                  c.Central.Root,
		Wd:                     c.Getwd(),
		Badges:                 c.Central.Badges,
		Reports:                c.Central.Reports,
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
	if want := 10; len(paths) != want {
		t.Errorf("got %v\nwant %v", len(paths), want)
	}

	got := []string{}
	if err := filepath.Walk(bd, func(path string, fi os.FileInfo, err error) error {
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

	if want := 10; len(got) != want {
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
	c.Central = &config.ConfigCentral{
		Enable:  true,
		Reports: filepath.Join(testdataDir(t), "reports"),
		Badges:  "badges",
	}
	if err := c.BuildCentralConfig(); err != nil {
		t.Fatal(err)
	}

	ctr := New(&CentralConfig{
		Repository:             c.Repository,
		Index:                  c.Central.Root,
		Wd:                     c.Getwd(),
		Badges:                 c.Central.Badges,
		Reports:                c.Central.Reports,
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
	b, err := ioutil.ReadFile(filepath.Join(testdataDir(t), "central_README.md.golden"))
	if err != nil {
		t.Fatal(err)
	}
	want := string(b)

	if got != want {
		t.Errorf("got %v\nwant %v", got, want)
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
