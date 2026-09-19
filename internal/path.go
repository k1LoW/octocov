package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ConfigPaths = []string{
	".octocov.yml",
	".octocov.yaml",
	"octocov.yml",
	"octocov.yaml",
}

func isGitRoot(p string) bool {
	dotGit := filepath.Join(p, ".git")
	fi, err := os.Stat(dotGit)
	if err != nil {
		return false
	}
	if fi.IsDir() {
		// Normal repo: .git/config exists
		gitConfig := filepath.Join(dotGit, "config")
		cfi, err := os.Stat(gitConfig)
		return err == nil && !cfi.IsDir()
	}
	// Worktree: .git is a file starting with "gitdir:"
	content, err := os.ReadFile(dotGit)
	if err != nil {
		return false
	}
	return strings.HasPrefix(string(content), "gitdir:")
}

func GitRoot(base string) (string, error) {
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

		if isGitRoot(p) {
			return p, nil
		}

		if filepath.Dir(p) == p {
			// root directory
			break
		}
		p = filepath.Dir(p)
	}

	// Build error message with all checked paths
	allPaths := []string{".git"}
	return "", fmt.Errorf("failed to traverse the root path (looking for %s): %s", strings.Join(allPaths, " or "), base)
}

func RootPath(base string) (string, error) {
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

		if isGitRoot(p) {
			return p, nil
		}

		// Check for all config files in defaultConfigPaths
		for _, configFile := range ConfigPaths {
			configPath := filepath.Join(p, configFile)
			if fi, err := os.Stat(configPath); err == nil && !fi.IsDir() {
				return p, nil
			}
		}

		if filepath.Dir(p) == p {
			// root directory
			break
		}
		p = filepath.Dir(p)
	}

	// Build error message with all checked paths
	allPaths := append([]string{".git"}, ConfigPaths...)
	return "", fmt.Errorf("failed to traverse the root path (looking for %s): %s", strings.Join(allPaths, " or "), base)
}

var defaultSkipDirs = map[string]struct{}{
	".git":        {},
	"node_modules": {},
	"vendor":      {},
	".bundle":     {},
	"__pycache__": {},
	".tox":        {},
	".venv":       {},
}

// CollectFiles walks from root and returns absolute paths of all files,
// skipping directories in defaultSkipDirs.
func CollectFiles(root string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var files []string
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if _, skip := defaultSkipDirs[info.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip .git file (present in git worktrees instead of .git directory)
		if info.Name() == ".git" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}
