package internal

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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

// CollectFiles returns absolute paths of all files under root, skipping
// directories in defaultSkipDirs.
// When root is inside a git work tree, it lists the files git knows about
// (tracked files, including those in submodules, and untracked files that are
// not ignored) instead of walking root, so that ignored directories such as
// build caches are never traversed. Otherwise it walks root.
func CollectFiles(root string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if files, err := collectGitFiles(absRoot); err == nil {
		return files, nil
	}
	return walkFiles(absRoot)
}

// collectGitFiles lists files under root with git ls-files.
// It returns an error when root is not inside a git work tree or git is unavailable.
func collectGitFiles(root string) ([]string, error) {
	// --recurse-submodules cannot be combined with --others, so list them separately.
	tracked, err := gitLsFiles(root, "--cached", "--recurse-submodules")
	if err != nil {
		return nil, err
	}
	untracked, err := gitLsFiles(root, "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var files []string
	for _, rel := range append(tracked, untracked...) {
		if inSkipDir(rel) {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(rel))
		// The index still lists files deleted from the work tree, and an
		// uninitialized submodule appears as a directory.
		fi, err := os.Lstat(path)
		if err != nil || fi.IsDir() {
			continue
		}
		files = append(files, path)
	}
	sort.Strings(files)
	return files, nil
}

func gitLsFiles(root string, args ...string) ([]string, error) {
	cmd := exec.Command("git", append([]string{"ls-files", "-z"}, args...)...) // #nosec G204
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var paths []string
	for p := range strings.SplitSeq(string(out), "\x00") {
		if p != "" {
			paths = append(paths, p)
		}
	}
	return paths, nil
}

// inSkipDir reports whether any directory in the slash-separated path rel is in defaultSkipDirs.
func inSkipDir(rel string) bool {
	dirs := strings.Split(rel, "/")
	for _, d := range dirs[:len(dirs)-1] {
		if _, skip := defaultSkipDirs[d]; skip {
			return true
		}
	}
	return false
}

func walkFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, skip := defaultSkipDirs[d.Name()]; skip {
				return filepath.SkipDir
			}
			return nil
		}
		// Skip .git file (present in git worktrees instead of .git directory)
		if d.Name() == ".git" {
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
