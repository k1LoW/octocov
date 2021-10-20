package config

import (
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
