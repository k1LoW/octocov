package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var ConfigPaths = []string{
	".octocov.yml",
	".octocov.yaml",
	"octocov.yml",
	"octocov.yaml",
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

		// Check for .git/config file
		gitConfig := filepath.Join(p, ".git", "config")
		if fi, err := os.Stat(gitConfig); err == nil && !fi.IsDir() {
			return p, nil
		}

		if filepath.Dir(p) == p {
			// root directory
			break
		}
		p = filepath.Dir(p)
	}

	// Build error message with all checked paths
	allPaths := []string{".git/config"}
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

		// Check for .git/config file
		gitConfig := filepath.Join(p, ".git", "config")
		if fi, err := os.Stat(gitConfig); err == nil && !fi.IsDir() {
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
	allPaths := append([]string{".git/config"}, ConfigPaths...)
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
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if _, skip := defaultSkipDirs[info.Name()]; skip {
				return filepath.SkipDir
			}
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
