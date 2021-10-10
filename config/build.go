package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/k1LoW/octocov/internal"
)

func (c *Config) Build() {
	// Repository
	if c.Repository == "" {
		c.Repository = os.Getenv("GITHUB_REPOSITORY")
	}

	// Coverage
	if c.Coverage == nil {
		c.Coverage = &ConfigCoverage{}
	}
	if c.Coverage.Path == "" {
		c.Coverage.Path = filepath.Dir(c.path)
	}

	// CodeToTestRatio
	if c.CodeToTestRatio != nil {
		if c.CodeToTestRatio.Code == nil {
			c.CodeToTestRatio.Code = []string{}
		}
		if c.CodeToTestRatio.Test == nil {
			c.CodeToTestRatio.Test = []string{}
		}
	}

	// TestExecutionTime
	if c.TestExecutionTime == nil {
		c.TestExecutionTime = &ConfigTestExecutionTime{}
	}

	// Report

	// Central
	if c.Central != nil {
		if c.Central.Root == "" {
			c.Central.Root = "."
		}
		if !strings.HasPrefix(c.Central.Root, "/") {
			c.Central.Root = filepath.Clean(filepath.Join(c.Root(), c.Central.Root))
		}
		if len(c.Central.Reports.Datastores) == 0 {
			c.Central.Reports.Datastores = append(c.Central.Reports.Datastores, defaultReportsDatastore)
		}
		if c.Central.Badges == "" {
			c.Central.Badges = defaultBadgesDir
		}
		if !strings.HasPrefix(c.Central.Badges, "/") {
			c.Central.Badges = filepath.Clean(filepath.Join(c.Root(), c.Central.Badges))
		}
	}

	// Push

	// Comment

	// Diff

	// GitRoot
	gitRoot, _ := internal.TraverseGitPath(c.Root())
	c.GitRoot = gitRoot
}
