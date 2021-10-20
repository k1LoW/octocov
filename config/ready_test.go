package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCoverageConfigReady(t *testing.T) {
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{},
			"coverage: is not set",
		},
		{
			&Config{
				Coverage: &ConfigCoverage{},
			},
			"coverage.path: is not set",
		},
		{
			&Config{
				Coverage: &ConfigCoverage{
					Path: "path/to/coverage.svg",
				},
			},
			"",
		},
	}
	for _, tt := range tests {
		err := tt.c.CoverageConfigReady()
		if err == nil && tt.want != "" {
			t.Errorf("got %v\nwant %v", err, tt.want)
			continue
		}
		if err != nil && tt.want == "" {
			t.Errorf("got %v\nwant %v", err, tt.want)
			continue
		}
		if err != nil && tt.want != "" {
			if got := err.Error(); got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		}
	}
}

func TestCodeToTestRatioConfigReady(t *testing.T) {
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{},
			"codeToTestRatio: is not set",
		},
		{
			&Config{
				CodeToTestRatio: &ConfigCodeToTestRatio{},
			},
			"codeToTestRatio.test: is not set",
		},
		{
			&Config{
				CodeToTestRatio: &ConfigCodeToTestRatio{
					Test: []string{"path/to/test/**"},
				},
			},
			"",
		},
	}
	for _, tt := range tests {
		err := tt.c.CodeToTestRatioConfigReady()
		if err == nil && tt.want != "" {
			t.Errorf("got %v\nwant %v", err, tt.want)
			continue
		}
		if err != nil && tt.want == "" {
			t.Errorf("got %v\nwant %v", err, tt.want)
			continue
		}
		if err != nil && tt.want != "" {
			if got := err.Error(); got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		}
	}
}

func TestTestExecutionTimeConfigReady(t *testing.T) {
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{},
			"testExecutionTime: is not set",
		},
		{
			&Config{
				TestExecutionTime: &ConfigTestExecutionTime{
					Steps: []string{},
				},
			},
			"coverage: is not set",
		},
		{
			&Config{
				TestExecutionTime: &ConfigTestExecutionTime{
					Steps: []string{
						"Run tests",
					},
				},
			},
			"",
		},
	}
	for _, tt := range tests {
		err := tt.c.TestExecutionTimeConfigReady()
		if err == nil && tt.want != "" {
			t.Errorf("got %v\nwant %v", err, tt.want)
			continue
		}
		if err != nil && tt.want == "" {
			t.Errorf("got %v\nwant %v", err, tt.want)
			continue
		}
		if err != nil && tt.want != "" {
			if got := err.Error(); got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		}
	}
}

func TestDiffConfigReady(t *testing.T) {
	os.Setenv("GITHUB_EVENT_NAME", "pull_request")
	os.Setenv("GITHUB_EVENT_PATH", filepath.Join(testdataDir(t), "config", "event_pull_request_opened.json"))
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{},
			"diff: is not set",
		},
		{
			&Config{
				Diff: &ConfigDiff{},
			},
			"diff.path: and diff.datastores: are not set",
		},
		{
			&Config{
				Diff: &ConfigDiff{
					Path: "path/to/report.json",
				},
			},
			"",
		},
		{
			&Config{
				Diff: &ConfigDiff{
					Path: "path/to/report.json",
					If:   "false",
				},
			},
			"the condition in the `if` section is not met (false)",
		},
	}
	for _, tt := range tests {
		err := tt.c.DiffConfigReady()
		if err == nil && tt.want != "" {
			t.Errorf("got %v\nwant %v", err, tt.want)
			continue
		}
		if err != nil && tt.want == "" {
			t.Errorf("got %v\nwant %v", err, tt.want)
			continue
		}
		if err != nil && tt.want != "" {
			if got := err.Error(); got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		}
	}
}
