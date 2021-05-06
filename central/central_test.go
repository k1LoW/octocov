package central

import (
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

	ctr := New(c)
	if err := ctr.collectReports(); err != nil {
		t.Fatal(err)
	}

	got := ctr.reports
	if want := 4; len(got) != want {
		t.Errorf("got %v\nwant %v", len(got), want)
	}
}

func TestGenerate(t *testing.T) {
	bd := t.TempDir()
	c := config.New()
	c.Central = &config.ConfigCentral{
		Enable:  true,
		Reports: filepath.Join(testdataDir(t), "reports"),
		Badges:  bd,
	}

	ctr := New(c)
	if err := ctr.Generate(); err != nil {
		t.Fatal(err)
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

	if want := 4; len(got) != want {
		t.Errorf("got %v\nwant %v", len(got), want)
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
