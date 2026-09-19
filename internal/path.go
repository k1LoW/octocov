package internal

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
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
	".git":         {},
	"node_modules": {},
	"vendor":       {},
	".bundle":      {},
	"__pycache__":  {},
	".tox":         {},
	".venv":        {},
}

// CollectFiles walks from root and returns absolute paths of all files,
// skipping directories in defaultSkipDirs and everything .gitignore excludes.
func CollectFiles(root string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var (
		files    []string
		patterns []gitignore.Pattern
	)
	// The patterns of a directory are read as the walk enters it, so they are in place before
	// its children are visited and a deeper .gitignore outranks a shallower one, which is the
	// order gitignore.Matcher reads them in. gitignore.ReadPatterns would collect them in one
	// call, but it walks the tree itself without honoring defaultSkipDirs, so a committed
	// vendor/ would be descended into twice. .git/info/exclude is not read: it is local to one
	// checkout and says nothing about the repository a report is being resolved against.
	m := gitignore.NewMatcher(nil)
	err = filepath.Walk(absRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		var segments []string
		if rel != "." {
			segments = strings.Split(filepath.ToSlash(rel), "/")
		}
		if info.IsDir() {
			if len(segments) > 0 {
				if _, skip := defaultSkipDirs[info.Name()]; skip {
					return filepath.SkipDir
				}
				if m.Match(segments, true) {
					return filepath.SkipDir
				}
			}
			if ps := readIgnorePatterns(path, segments); len(ps) > 0 {
				patterns = append(patterns, ps...)
				m = gitignore.NewMatcher(patterns)
			}
			return nil
		}
		// Skip .git file (present in git worktrees instead of .git directory)
		if info.Name() == ".git" {
			return nil
		}
		if m.Match(segments, false) {
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

// readIgnorePatterns reads the .gitignore of dir, if it has one, as patterns scoped to domain.
func readIgnorePatterns(dir string, domain []string) []gitignore.Pattern {
	f, err := os.Open(filepath.Join(dir, ".gitignore"))
	if err != nil {
		return nil
	}
	defer f.Close() //nostyle:handlerrors
	var ps []gitignore.Pattern
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasPrefix(s, "#") || strings.TrimSpace(s) == "" {
			continue
		}
		ps = append(ps, gitignore.ParsePattern(s, domain))
	}
	return ps
}

func DetectPrefix(root, wd string, files, cfiles []string) string {
	var rcfiles [][]string
	for _, f := range cfiles {
		s := strings.Split(filepath.FromSlash(f), string(filepath.Separator))
		reverse(s)
		rcfiles = append(rcfiles, s)
	}

	var rfiles [][]string
	for _, f := range files {
		s := strings.Split(filepath.FromSlash(f), string(filepath.Separator))
		reverse(s)
		rfiles = append(rfiles, s)
	}

	j := 0
	prefix := ""
	for i := 0; i < len(rcfiles); i++ {
	L:
		for j < len(rfiles) {
			if rcfiles[i][0] != rfiles[j][0] {
				j += 1
				continue
			}
			if i < len(rcfiles)-1 && rcfiles[i][0] == rcfiles[i+1][0] {
				// if the same file name continues, exclude it from sampling.
				i += 2
				continue L
			}

			detect := func(s []string, i, j int) string {
				// reverse slice
				reverse(s)
				suffix := join(s...)
				cfile := cfiles[i]
				cfp := strings.TrimSuffix(cfile, suffix)
				file := files[j]
				fp := strings.TrimSuffix(file, suffix)

				// fmt.Printf("root: %s\nwd: %s\n", root, wd)
				// fmt.Printf("file: %s\ncfile: %s\n", file, cfile)
				// fmt.Printf("suffix: %s\n", suffix)
				// fmt.Printf("file_prefix: %s\ncfile_prefix: %s\n", fp, cfp)
				// fmt.Printf("---\n")

				if len(fp) < len(root) {
					cfp = filepath.Join(cfp, strings.TrimPrefix(root, fp))
				}

				prefix := filepath.Join(cfp, strings.TrimPrefix(wd, root))
				if prefix == "." {
					return ""
				}
				return prefix
			}

			for k := range rcfiles[i] {
				if len(rcfiles[i]) <= k || len(rfiles[j]) <= k || rcfiles[i][k] != rfiles[j][k] {
					return detect(rcfiles[i][:k], i, j)
				}
			}

			for k := range rfiles[j] {
				if len(rfiles[j]) <= k || len(rcfiles[i]) <= k || rcfiles[i][k] != rfiles[j][k] {
					return detect(rcfiles[i][:k], i, j)
				}
			}

			if len(rcfiles[i]) == len(rfiles[j]) && rcfiles[i][len(rcfiles[i])-1] == rfiles[j][len(rfiles[j])-1] {
				return detect(rcfiles[i], i, j)
			}

			j += 1
		}
	}
	return prefix
}

func reverse(s []string) {
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
}

func join(elem ...string) string {
	if runtime.GOOS == "windows" && elem[0][len(elem[0])-1] == ':' {
		// Allow filepath.join to be an absolute path
		elem[0] += "\\"
	}
	return filepath.Join(elem...)
}
