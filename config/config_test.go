package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k1LoW/octocov/pkg/coverage"
	"github.com/k1LoW/octocov/report"
)

func TestMain(m *testing.M) {
	envCache := os.Environ()

	m.Run()

	if err := revertEnv(envCache); err != nil {
		panic(err)
	}
}

func TestLoad(t *testing.T) {
	tests := []struct {
		wd      string
		path    string
		wantErr bool
	}{
		{testdataDir(t), "", false},
		{filepath.Join(testdataDir(t), "config"), "", false},
		{filepath.Join(testdataDir(t), "config"), ".octocov.yml", false},
		{filepath.Join(testdataDir(t), "config"), "no.yml", true},
	}
	for _, tt := range tests {
		c := New()
		c.wd = tt.wd
		if err := c.Load(tt.path); err != nil {
			if !tt.wantErr {
				t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
			}
		} else {
			if tt.wantErr {
				t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
			}
		}
	}
}

func TestDatasourceGithubPath(t *testing.T) {
	if err := clearEnv(); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GITHUB_REPOSITORY", "foo/bar")

	c := New()
	c.Datastore = &ConfigDatastore{
		Github: &ConfigDatastoreGithub{
			Repository: "report/dest",
		},
	}

	c.Build()
	if got := c.DatastoreConfigReady(); got != true {
		t.Errorf("got %v\nwant %v", got, true)
	}
	if err := c.BuildDatastoreConfig(); err != nil {
		t.Fatal(err)
	}
	want := "reports/foo/bar/report.json"
	if got := c.Datastore.Github.Path; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestAcceptable(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"60%", true},
		{"50%", false},
		{"49.9%", false},
	}
	for _, tt := range tests {
		c := New()
		c.Coverage.Acceptable = tt.in
		c.Build()

		r := report.New()
		r.Coverage = &coverage.Coverage{
			Covered: 50,
			Total:   100,
		}
		if err := c.Accepptable(r); err != nil {
			if !tt.wantErr {
				t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
			}
		} else {
			if tt.wantErr {
				t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
			}
		}
	}
}

func revertEnv(envCache []string) error {
	if err := clearEnv(); err != nil {
		return err
	}
	for _, e := range envCache {
		splitted := strings.Split(e, "=")
		if err := os.Setenv(splitted[0], splitted[1]); err != nil {
			return err
		}
	}
	return nil
}

func clearEnv() error {
	for _, e := range os.Environ() {
		splitted := strings.Split(e, "=")
		if err := os.Unsetenv(splitted[0]); err != nil {
			return err
		}
	}
	return nil
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
