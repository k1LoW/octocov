package internal

import (
	"fmt"
	"os"
	"path/filepath"
)

func TraverseGitPath(base string) (string, error) {
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
