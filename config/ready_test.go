package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/k1LoW/octocov/internal"
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

func TestPushConfigReady(t *testing.T) {
	os.Setenv("GITHUB_REPOSITORY", "owner/repo")
	os.Setenv("GITHUB_EVENT_NAME", "pull_request")
	os.Setenv("GITHUB_EVENT_PATH", filepath.Join(testdataDir(t), "config", "event_pull_request_opened.json"))
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{},
			"push: is not set",
		},
		{
			&Config{
				Push: &ConfigPush{},
			},
			"failed to traverse the Git root path",
		},
		{
			&Config{
				Push: &ConfigPush{
					Enable: internal.Bool(true),
				},
			},
			"failed to traverse the Git root path",
		},
		{
			&Config{
				GitRoot: "/path/to",
				Push: &ConfigPush{
					Enable: internal.Bool(true),
				},
			},
			"",
		},
		{
			&Config{
				GitRoot: "/path/to",
				Push: &ConfigPush{
					Enable: internal.Bool(true),
					If:     "false",
				},
			},
			"the condition in the `if` section is not met (false)",
		},
	}
	for _, tt := range tests {
		err := tt.c.PushConfigReady()
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

func TestCommentConfigReady(t *testing.T) {
	os.Setenv("GITHUB_REPOSITORY", "owner/repo")
	os.Setenv("GITHUB_EVENT_NAME", "pull_request")
	os.Setenv("GITHUB_EVENT_PATH", filepath.Join(testdataDir(t), "config", "event_pull_request_opened.json"))
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{},
			"comment: is not set",
		},
		{
			&Config{
				Comment: &ConfigComment{},
			},
			"",
		},
		{
			&Config{
				Comment: &ConfigComment{
					Enable: internal.Bool(true),
				},
			},
			"",
		},
		{
			&Config{
				Comment: &ConfigComment{
					If: "false",
				},
			},
			"the condition in the `if` section is not met (false)",
		},
	}
	for _, tt := range tests {
		err := tt.c.CommentConfigReady()
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

func TestCoverageBadgeConfigReady(t *testing.T) {
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{
				Coverage: &ConfigCoverage{
					Path: "path/to/coverage.xml",
				},
			},
			"coverage.badge.path: is not set",
		},
		{
			&Config{
				Coverage: &ConfigCoverage{
					Path: "path/to/coverage.xml",
					Badge: ConfigCoverageBadge{
						Path: "path/to/coverage.svg",
					},
				},
			},
			"",
		},
	}
	for _, tt := range tests {
		err := tt.c.CoverageBadgeConfigReady()
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

func TestCodeToTestRatioBadgeConfigReady(t *testing.T) {
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{
				CodeToTestRatio: &ConfigCodeToTestRatio{
					Test: []string{
						"**_test.go",
					},
				},
			},
			"codeToTestRatio.badge.path: is not set",
		},
		{
			&Config{
				CodeToTestRatio: &ConfigCodeToTestRatio{
					Test: []string{
						"**_test.go",
					},
					Badge: ConfigCodeToTestRatioBadge{
						Path: "path/to/ratio.svg",
					},
				},
			},
			"",
		},
	}
	for _, tt := range tests {
		err := tt.c.CodeToTestRatioBadgeConfigReady()
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

func TestTestExecutionTimeBadgeConfigReady(t *testing.T) {
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{
				TestExecutionTime: &ConfigTestExecutionTime{
					Steps: []string{
						"Run tests",
					},
				},
			},
			"testExecutionTime.badge.path: is not set",
		},
		{
			&Config{
				TestExecutionTime: &ConfigTestExecutionTime{
					Steps: []string{
						"Run tests",
					},
					Badge: ConfigTestExecutionTimeBadge{
						Path: "path/to/time.svg",
					},
				},
			},
			"",
		},
	}
	for _, tt := range tests {
		err := tt.c.TestExecutionTimeBadgeConfigReady()
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

func TestCentralConfigReady(t *testing.T) {
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{},
			"central: is not set",
		},
		{
			&Config{
				Central: &ConfigCentral{},
			},
			"repository: not set (or env GITHUB_REPOSITORY is not set)",
		},
		{
			&Config{
				Central: &ConfigCentral{
					Enable: internal.Bool(false),
				},
			},
			"central.enable: is false",
		},
		{
			&Config{
				Central: &ConfigCentral{
					Enable: internal.Bool(true),
				},
			},
			"repository: not set (or env GITHUB_REPOSITORY is not set)",
		},
		{
			&Config{
				Repository: "owner/repo",
				Central: &ConfigCentral{
					Enable: internal.Bool(true),
				},
			},
			"central.reports.datastores is not set",
		},
		{
			&Config{
				Repository: "owner/repo",
				Central: &ConfigCentral{
					Enable: internal.Bool(true),
					Reports: ConfigCentralReports{
						Datastores: []string{
							"s3://bucket/reports",
						},
					},
				},
			},
			"",
		},
		{
			&Config{
				Repository: "owner/repo",
				Central: &ConfigCentral{
					Enable: internal.Bool(true),
					Reports: ConfigCentralReports{
						Datastores: []string{
							"s3://bucket/reports",
						},
					},
					If: "false",
				},
			},
			"the condition in the `if` section is not met (false)",
		},
	}
	for _, tt := range tests {
		err := tt.c.CentralConfigReady()
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

func TestCentralPushConfigReady(t *testing.T) {
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{
				Repository: "owner/repo",
				Central: &ConfigCentral{
					Enable: internal.Bool(true),
					Reports: ConfigCentralReports{
						Datastores: []string{
							"s3://bucket/reports",
						},
					},
				},
			},
			"central.push: is not set",
		},
		{
			&Config{
				Repository: "owner/repo",
				Central: &ConfigCentral{
					Enable: internal.Bool(true),
					Reports: ConfigCentralReports{
						Datastores: []string{
							"s3://bucket/reports",
						},
					},
					Push: &ConfigPush{
						Enable: internal.Bool(true),
					},
				},
			},
			"failed to traverse the Git root path",
		},
		{
			&Config{
				Repository: "owner/repo",
				Central: &ConfigCentral{
					Enable: internal.Bool(true),
					Reports: ConfigCentralReports{
						Datastores: []string{
							"s3://bucket/reports",
						},
					},
					Push: &ConfigPush{
						Enable: internal.Bool(true),
					},
				},
				GitRoot: "/path/to",
			},
			"",
		},
		{
			&Config{
				Repository: "owner/repo",
				Central: &ConfigCentral{
					Enable: internal.Bool(true),
					Reports: ConfigCentralReports{
						Datastores: []string{
							"s3://bucket/reports",
						},
					},
					Push: &ConfigPush{
						Enable: internal.Bool(true),
						If:     "false",
					},
				},
				GitRoot: "/path/to",
			},
			"the condition in the `if` section is not met (false)",
		},
	}
	for _, tt := range tests {
		err := tt.c.CentralPushConfigReady()
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
	os.Setenv("GITHUB_REPOSITORY", "owner/repo")
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

func TestReportConfigReady(t *testing.T) {
	os.Setenv("GITHUB_REPOSITORY", "owner/repo")
	os.Setenv("GITHUB_EVENT_NAME", "pull_request")
	os.Setenv("GITHUB_EVENT_PATH", filepath.Join(testdataDir(t), "config", "event_pull_request_opened.json"))
	tests := []struct {
		c    *Config
		want string
	}{
		{
			&Config{},
			"report: is not set",
		},
		{
			&Config{
				Report: &ConfigReport{},
			},
			"report.datastores: and report.path: are not set",
		},
		{
			&Config{
				Report: &ConfigReport{
					Datastores: []string{
						"s3://bucket/reports",
					},
				},
			},
			"",
		},
		{
			&Config{
				Report: &ConfigReport{
					Datastores: []string{
						"s3://bucket/reports",
					},
					If: "false",
				},
			},
			"the condition in the `if` section is not met (false)",
		},
	}
	for _, tt := range tests {
		err := tt.c.ReportConfigReady()
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
