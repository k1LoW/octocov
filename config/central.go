package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (c *Config) CentralConfigReady() bool {
	return (c.Central != nil && c.Central.Enable)
}

func (c *Config) BuildCentralConfig() error {
	if c.Repository == "" {
		return errors.New("repository: not set (or env GITHUB_REPOSITORY is not set)")
	}
	if c.Central == nil {
		return errors.New("central: not set")
	}
	if c.Central.Root == "" {
		c.Central.Root = "."
	}
	if !strings.HasPrefix(c.Central.Root, "/") {
		c.Central.Root = filepath.Clean(filepath.Join(c.Root(), c.Central.Root))
	}
	if c.Central.Push.Enable {
		gitRoot, err := traverseGitPath(c.Central.Root)
		if err != nil {
			return err
		}
		c.Central.Push.Root = gitRoot
	}
	if c.Central.Reports == "" {
		c.Central.Reports = defaultReportsDir
	}
	if !strings.HasPrefix(c.Central.Reports, "/") {
		c.Central.Reports = filepath.Clean(filepath.Join(c.Root(), c.Central.Reports))
	}
	if c.Central.Badges == "" {
		c.Central.Badges = defaultBadgesDir
	}
	if !strings.HasPrefix(c.Central.Badges, "/") {
		c.Central.Badges = filepath.Clean(filepath.Join(c.Root(), c.Central.Badges))
	}

	return nil
}

func traverseGitPath(base string) (string, error) {
	p, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	for {
		fi, err := os.Stat(p)
		if err != nil {
			return "", err
		}
		if !fi.IsDir() {
			p = filepath.Dir(p)
			continue
		}
		gitConfig := filepath.Join(p, ".git", "config")
		if fi, err := os.Stat(gitConfig); err == nil && !fi.IsDir() {
			return p, nil
		}
		if p == "/" {
			break
		}
		p = filepath.Dir(p)
	}
	return "", fmt.Errorf("failed to traverse the Git root path: %s", base)
}
