package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/k1LoW/octocov/internal"
)

func (c *Config) Build() {
	// Repository
	if c.Repository == "" {
		c.Repository = os.Getenv("GITHUB_REPOSITORY")
	}

	// Coverage
	if c.Coverage == nil {
		c.Coverage = &Coverage{}
	}
	if c.Coverage.Path != "" {
		_, _ = fmt.Fprintln(os.Stderr, "Deprecated: coverage.path: has been deprecated. please use coverage.paths: instead.") //nostyle:handlerrors
		c.Coverage.Paths = append(c.Coverage.Paths, c.Coverage.Path)
	}
	if len(c.Coverage.Paths) == 0 {
		c.Coverage.Paths = append(c.Coverage.Paths, filepath.Dir(c.path))
	} else {
		var paths []string
		for _, p := range c.Coverage.Paths {
			p = filepath.FromSlash(p)
			paths = append(paths, filepath.Join(filepath.Dir(c.path), p))
		}
		c.Coverage.Paths = paths
	}

	// TestExecutionTime
	if c.TestExecutionTime == nil {
		c.TestExecutionTime = &TestExecutionTime{}
	}

	// Badges
	c.Coverage.Badge.build("coverage.badge", c.Root())
	if c.CodeToTestRatio != nil {
		c.CodeToTestRatio.Badge.build("codeToTestRatio.badge", c.Root())
	}
	c.TestExecutionTime.Badge.build("testExecutionTime.badge", c.Root())

	// Report

	// Central
	if c.Central != nil {
		if c.Central.Root == "" {
			c.Central.Root = "."
		}
		if !filepath.IsAbs(c.Central.Root) {
			c.Central.Root = filepath.Clean(filepath.Join(c.Root(), c.Central.Root))
		}
		if len(c.Central.Reports.Datastores) == 0 {
			c.Central.Reports.Datastores = append(c.Central.Reports.Datastores, defaultReportsDatastore)
		}
		if len(c.Central.Badges.Datastores) == 0 {
			c.Central.Badges.Datastores = append(c.Central.Badges.Datastores, defaultBadgesDatastore)
		}
		c.Central.Badges.Coverage.build("central.badges.coverage", c.Root())
		c.Central.Badges.CodeToTestRatio.build("central.badges.codeToTestRatio", c.Root())
		c.Central.Badges.TestExecutionTime.build("central.badges.testExecutionTime", c.Root())
	}

	// Push

	// Comment

	// Diff

	// Viewer
	for _, removed := range []struct {
		key string
		set bool
	}{
		{"comment", c.Comment != nil && c.Comment.HideCoverageLink},
		{"summary", c.Summary != nil && c.Summary.HideCoverageLink},
		{"body", c.Body != nil && c.Body.HideCoverageLink},
	} {
		if removed.set {
			_, _ = fmt.Fprintf(os.Stderr, "Removed: %s.hideCoverageLink: has been removed. please use viewer: none instead.\n", removed.key) //nostyle:handlerrors
		}
	}

	// GitRoot
	gitRoot, _ := internal.GitRoot(c.Root()) //nostyle:handlerrors
	c.GitRoot = gitRoot
}
