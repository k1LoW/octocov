package config

import (
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	envCache := os.Environ()

	m.Run()

	if err := revertEnv(envCache); err != nil {
		panic(err)
	}
}

func TestDatasourceGithubPath(t *testing.T) {
	if err := clearEnv(); err != nil {
		t.Fatal(err)
	}
	os.Setenv("GITHUB_REPOSITORY", "foo/bar")

	c := New()
	c.Datastore.Github.Repository = "report/dest"
	c.Build()
	if got := c.DatastoreConfigReady(); got != true {
		t.Errorf("got %v\nwant %v", got, true)
	}
	if err := c.BuildDatastoreConfig(); err != nil {
		t.Fatal(err)
	}
	want := "reports/foo/bar.json"
	if got := c.Datastore.Github.Path; got != want {
		t.Errorf("got %v\nwant %v", got, want)
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
