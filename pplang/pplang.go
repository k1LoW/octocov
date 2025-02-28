package pplang

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	giturls "github.com/chainguard-dev/git-urls"
	"github.com/google/go-github/v67/github"
	"github.com/k1LoW/go-github-client/v67/factory"
	"gopkg.in/ini.v1"
)

func Detect(dir string) (string, error) {
	fsys := os.DirFS(dir)

	if lang, err := DetectFS(fsys); err == nil {
		return lang, nil
	}

	if client, err := factory.NewGithubClient(); err == nil {
		if lang, err := DetectUsingAPI(client, fsys); err == nil {
			return lang, nil
		}
	}

	return "", errors.New("can not detect project programming language")
}

func DetectFS(fsys fs.FS) (string, error) {
	if fi, err := fs.Stat(fsys, "go.mod"); err == nil && !fi.IsDir() {
		return "Go", nil
	}
	return "", errors.New("can not detect project programming language")
}

func DetectUsingAPI(client *github.Client, fsys fs.FS) (string, error) {
	owner, repo, err := ownerRepo(fsys)
	if err != nil {
		return "", err
	}
	langs, _, err := client.Repositories.ListLanguages(context.Background(), owner, repo)
	if err != nil {
		return "", err
	}
	lang := ""
	c := 0
	for l := range langs {
		if langs[l] > c {
			lang = l
			c = langs[l]
		}
	}
	if lang != "" {
		return lang, nil
	}

	return "", errors.New("can not detect project programming language")
}

func ownerRepo(fsys fs.FS) (string, string, error) {
	if os.Getenv("GITHUB_REPOSITORY") != "" {
		splitted := strings.Split(os.Getenv("GITHUB_REPOSITORY"), "/")
		if len(splitted) == 2 {
			return splitted[0], splitted[1], nil
		}
	}
	if f, err := fs.ReadFile(fsys, ".git/config"); err == nil {
		c, err := ini.Load(f)
		if err != nil {
			return "", "", err
		}
		for _, s := range c.Sections() {
			if !strings.Contains(s.Name(), "remote") || s.Key("url").String() == "" {
				continue
			}
			u, err := giturls.Parse(s.Key("url").String())
			if err != nil {
				continue
			}
			splitted := strings.Split(u.Path, "/")
			switch len(splitted) {
			case 3:
				return splitted[1], strings.TrimSuffix(splitted[2], filepath.Ext(splitted[2])), nil
			case 2:
				return splitted[0], strings.TrimSuffix(splitted[1], filepath.Ext(splitted[1])), nil
			}
		}
	}

	return "", "", errors.New("can not get owner/repo")
}
