package config

import (
	"os"
	"testing"
)

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

func TestDatasourceS3Path(t *testing.T) {
	if err := clearEnv(); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GITHUB_REPOSITORY", "foo/bar")

	c := New()
	c.Datastore = &ConfigDatastore{
		S3: &ConfigDatastoreS3{
			Bucket: "test-bucket",
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
	if got := c.Datastore.S3.Path; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestDatasourceGCSPath(t *testing.T) {
	if err := clearEnv(); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GITHUB_REPOSITORY", "foo/bar")

	c := New()
	c.Datastore = &ConfigDatastore{
		GCS: &ConfigDatastoreGCS{
			Bucket: "test-bucket",
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
	if got := c.Datastore.GCS.Path; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}

func TestDatasourceBQTable(t *testing.T) {
	if err := clearEnv(); err != nil {
		t.Fatal(err)
	}

	os.Setenv("GITHUB_REPOSITORY", "foo/bar")

	c := New()
	c.Datastore = &ConfigDatastore{
		BQ: &ConfigDatastoreBQ{
			Project: "test-project-id",
			Dataset: "test_dataset_id",
		},
	}

	c.Build()
	if got := c.DatastoreConfigReady(); got != true {
		t.Errorf("got %v\nwant %v", got, true)
	}
	if err := c.BuildDatastoreConfig(); err != nil {
		t.Fatal(err)
	}
	want := "reports"
	if got := c.Datastore.BQ.Table; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}
