package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

type ConfigDatastore struct {
	If     string                 `yaml:"if,omitempty"`
	Github *ConfigDatastoreGithub `yaml:"github,omitempty"`
	S3     *ConfigDatastoreS3     `yaml:"s3,omitempty"`
	GCS    *ConfigDatastoreGCS    `yaml:"gcs,omitempty"`
	BQ     *ConfigDatastoreBQ     `yaml:"bq,omitempty"`
}

type ConfigDatastoreGithub struct {
	Repository string `yaml:"repository"`
	Branch     string `yaml:"branch"`
	Path       string `yaml:"path"`
}

type ConfigDatastoreS3 struct {
	Bucket string `yaml:"bucket"`
	Path   string `yaml:"path"`
}

type ConfigDatastoreGCS struct {
	Bucket string `yaml:"bucket"`
	Path   string `yaml:"path"`
}

type ConfigDatastoreBQ struct {
	Project string `yaml:"project"`
	Dataset string `yaml:"dataset"`
	Table   string `yaml:"table"`
}

func (c *Config) DatastoreConfigReady() bool {
	if c.Datastore == nil {
		return false
	}
	ok, err := CheckIf(c.Datastore.If)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Skip storing the report: %v\n", err)
		return false
	}
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Skip storing the report: the condition in the `if` section is not met (%s)\n", c.Datastore.If)
		return false
	}
	return true
}

func (c *Config) BuildDatastoreConfig() error {
	if c.Datastore.Github == nil && c.Datastore.S3 == nil && c.Datastore.GCS == nil && c.Datastore.BQ == nil {
		return errors.New("datastore not set")
	}
	if c.Datastore.Github != nil {
		// GitHub
		if err := c.buildDatastoreGithubConfig(); err != nil {
			return err
		}
	}
	if c.Datastore.S3 != nil {
		// S3
		if err := c.buildDatastoreS3Config(); err != nil {
			return err
		}
	}
	if c.Datastore.GCS != nil {
		// GCS
		if err := c.buildDatastoreGCSConfig(); err != nil {
			return err
		}
	}
	if c.Datastore.BQ != nil {
		// BigQuery
		if err := c.buildDatastoreBQConfig(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Config) buildDatastoreGithubConfig() error {
	if c.Datastore.Github == nil {
		return errors.New("datastore.github not set")
	}
	if c.Datastore.Github.Branch == "" {
		c.Datastore.Github.Branch = defaultBranch
	}
	if c.Datastore.Github.Path == "" && c.Repository != "" {
		c.Datastore.Github.Path = fmt.Sprintf("%s/%s/report.json", defaultReportsDir, c.Repository)
	}
	if c.Datastore.Github.Repository == "" {
		return errors.New("datastore.github.repository not set")
	}
	if strings.Count(c.Datastore.Github.Repository, "/") != 1 {
		return errors.New("datastore.github.repository should be 'owner/repo'")
	}
	if c.Datastore.Github.Branch == "" {
		return errors.New("datastore.github.branch not set")
	}
	if c.Datastore.Github.Path == "" {
		return errors.New("datastore.github.path not set")
	}
	return nil
}

func (c *Config) buildDatastoreS3Config() error {
	if c.Datastore.S3 == nil {
		return errors.New("datastore.s3 not set")
	}
	if c.Datastore.S3.Bucket == "" {
		return errors.New("datastore.s3.bucket not set")
	}
	if c.Datastore.S3.Path == "" && c.Repository != "" {
		c.Datastore.S3.Path = fmt.Sprintf("%s/%s/report.json", defaultReportsDir, c.Repository)
	}
	if c.Datastore.S3.Path == "" {
		return errors.New("datastore.s3.path not set")
	}
	return nil
}

func (c *Config) buildDatastoreGCSConfig() error {
	if c.Datastore.GCS == nil {
		return errors.New("datastore.gcs not set")
	}
	if c.Datastore.GCS.Bucket == "" {
		return errors.New("datastore.gcs.bucket not set")
	}
	if c.Datastore.GCS.Path == "" && c.Repository != "" {
		c.Datastore.GCS.Path = fmt.Sprintf("%s/%s/report.json", defaultReportsDir, c.Repository)
	}
	if c.Datastore.GCS.Path == "" {
		return errors.New("datastore.gcs.path not set")
	}
	return nil
}

func (c *Config) buildDatastoreBQConfig() error {
	if c.Datastore.BQ == nil {
		return errors.New("datastore.bq not set")
	}
	if c.Datastore.BQ.Project == "" {
		return errors.New("datastore.bq.project not set")
	}
	if c.Datastore.BQ.Dataset == "" {
		return errors.New("datastore.bq.dataset not set")
	}
	if c.Datastore.BQ.Table == "" {
		c.Datastore.BQ.Table = defaultReportsDir
	}
	return nil
}
