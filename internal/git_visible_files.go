package internal

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

type gitVisibleCollector struct {
	gitRoot      string
	trackedFiles map[string]struct{}
	trackedDirs  map[string]struct{}
	files        []string
}

// CollectGitVisibleFiles returns visible worktree files under wd using .gitignore rules.
// It keeps tracked files even if they are later ignored, and falls back to a raw
// filesystem walk when wd is not inside a git repository.
func CollectGitVisibleFiles(wd string) ([]string, error) {
	absWd, err := filepath.Abs(wd)
	if err != nil {
		return nil, err
	}

	gitRoot, err := GitRoot(absWd)
	if err != nil {
		return CollectFiles(absWd)
	}

	collector, err := newGitVisibleCollector(gitRoot)
	if err != nil {
		return nil, err
	}
	return collector.collect(absWd)
}

func newGitVisibleCollector(gitRoot string) (*gitVisibleCollector, error) {
	repo, err := git.PlainOpen(gitRoot)
	if err != nil {
		return nil, err
	}
	index, err := repo.Storer.Index()
	if err != nil {
		return nil, err
	}

	c := &gitVisibleCollector{
		gitRoot:      gitRoot,
		trackedFiles: make(map[string]struct{}, len(index.Entries)),
		trackedDirs:  map[string]struct{}{gitRoot: {}},
		files:        make([]string, 0, len(index.Entries)),
	}
	for _, entry := range index.Entries {
		absPath := filepath.Join(gitRoot, filepath.FromSlash(entry.Name))
		c.trackedFiles[absPath] = struct{}{}
		for dir := filepath.Dir(absPath); ; dir = filepath.Dir(dir) {
			if _, exists := c.trackedDirs[dir]; exists {
				break
			}
			c.trackedDirs[dir] = struct{}{}
		}
	}

	return c, nil
}

func (c *gitVisibleCollector) collect(wd string) ([]string, error) {
	patterns, err := c.rootPatternStack(wd)
	if err != nil {
		return nil, err
	}
	c.files = c.files[:0]
	if err := c.walk(wd, patterns); err != nil {
		return nil, err
	}
	return c.files, nil
}

func (c *gitVisibleCollector) walk(dir string, patterns []gitignore.Pattern) error {
	matcher := gitignore.NewMatcher(patterns)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		path := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			if _, skip := defaultSkipDirs[entry.Name()]; skip {
				continue
			}
			ignored, err := c.isIgnored(matcher, path, true)
			if err != nil {
				return err
			}
			if ignored {
				if _, tracked := c.trackedDirs[path]; !tracked {
					continue
				}
			}
			childPatterns, err := c.patternStack(path, patterns)
			if err != nil {
				return err
			}
			if err := c.walk(path, childPatterns); err != nil {
				return err
			}
			continue
		}

		if entry.Name() == ".git" {
			continue
		}
		if _, tracked := c.trackedFiles[path]; tracked {
			c.files = append(c.files, path)
			continue
		}
		ignored, err := c.isIgnored(matcher, path, false)
		if err != nil {
			return err
		}
		if ignored {
			continue
		}
		c.files = append(c.files, path)
	}

	return nil
}

func (c *gitVisibleCollector) rootPatternStack(dir string) ([]gitignore.Pattern, error) {
	// The root call bootstraps repo-level excludes and ancestor .gitignore files.
	stack, err := readGitIgnoreFile(filepath.Join(c.gitRoot, ".git", "info", "exclude"), nil)
	if err != nil {
		return nil, err
	}

	dirParts, err := relPathComponents(c.gitRoot, dir)
	if err != nil {
		return nil, err
	}
	for i := range dirParts {
		ancestorParts := dirParts[:i]
		ancestorDir := c.gitRoot
		if len(ancestorParts) > 0 {
			ancestorDir = filepath.Join(c.gitRoot, filepath.Join(ancestorParts...))
		}
		stack, err = appendIgnorePatterns(stack, filepath.Join(ancestorDir, ".gitignore"), ancestorParts)
		if err != nil {
			return nil, err
		}
	}

	return c.patternStack(dir, stack)
}

func (c *gitVisibleCollector) patternStack(dir string, parent []gitignore.Pattern) ([]gitignore.Pattern, error) {
	domain, err := relPathComponents(c.gitRoot, dir)
	if err != nil {
		return nil, err
	}
	return appendIgnorePatterns(parent, filepath.Join(dir, ".gitignore"), domain)
}

func (c *gitVisibleCollector) isIgnored(matcher gitignore.Matcher, path string, isDir bool) (bool, error) {
	parts, err := relPathComponents(c.gitRoot, path)
	if err != nil {
		return false, err
	}
	return matcher.Match(parts, isDir), nil
}

func appendIgnorePatterns(stack []gitignore.Pattern, path string, domain []string) ([]gitignore.Pattern, error) {
	ps, err := readGitIgnoreFile(path, domain)
	if err != nil {
		return nil, err
	}
	if len(ps) == 0 {
		return stack, nil
	}
	combined := make([]gitignore.Pattern, len(stack)+len(ps))
	copy(combined, stack)
	copy(combined[len(stack):], ps)
	return combined, nil
}

func readGitIgnoreFile(path string, domain []string) (_ []gitignore.Pattern, err error) {
	patterns := make([]gitignore.Pattern, 0)

	var f *os.File
	f, err = os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return patterns, nil
		}
		return nil, err
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = errors.Join(err, closeErr)
		}
	}()

	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}
		patterns = append(patterns, gitignore.ParsePattern(line, domain))
	}
	if err = s.Err(); err != nil {
		return nil, err
	}
	return patterns, nil
}

func relPathComponents(root, path string) ([]string, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return nil, err
	}
	if rel == "." {
		return []string{}, nil
	}
	return strings.Split(filepath.ToSlash(rel), "/"), nil
}
