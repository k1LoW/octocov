package internal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func GetRootPath(base string) (string, error) {
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

func DetectPrefix(gitRoot, wd string, files, cfiles []string) string {
	rcfiles := [][]string{}
	for _, f := range cfiles {
		s := strings.Split(f, "/")
		reverse(s)
		rcfiles = append(rcfiles, s)
	}

	rfiles := [][]string{}
	for _, f := range files {
		s := strings.Split(f, "/")
		// reverse slice
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
				suffix := filepath.Join(s...)
				cfile := cfiles[i]
				cfp := strings.TrimSuffix(cfile, suffix)
				file := files[j]
				fp := strings.TrimSuffix(file, suffix)

				// fmt.Printf("gitRoot: %s\nwd: %s\n", gitRoot, wd)
				// fmt.Printf("file: %s\ncfile: %s\n", file, cfile)
				// fmt.Printf("suffix: %s\n", suffix)
				// fmt.Printf("file_prefix: %s\ncfile_prefix: %s\n", fp, cfp)
				// fmt.Printf("---\n")

				if len(fp) < len(gitRoot) {
					cfp = filepath.Join(cfp, strings.TrimPrefix(gitRoot, fp))
				}

				prefix := filepath.Join(cfp, strings.TrimPrefix(wd, gitRoot))
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
