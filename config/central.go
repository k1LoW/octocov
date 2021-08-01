package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/k1LoW/octocov/datastore"
)

func (c *Config) CentralConfigReady() bool {
	return (c.Central != nil && c.Central.Enable)
}

func (c *Config) CentralPushConfigReady() bool {
	if !c.CentralConfigReady() || !c.Central.Push.Enable || c.GitRoot == "" {
		return false
	}
	ok, err := CheckIf(c.Central.Push.If)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Skip pushing badges: %v\n", err)
		return false
	}
	if !ok {
		_, _ = fmt.Fprintf(os.Stderr, "Skip pushing badges: the condition in the `if` section is not met (%s)\n", c.Push.If)
		return false
	}
	return true
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
	if c.Central.Reports == "" {
		c.Central.Reports = defaultReportsDir
	}
	if c.Central.Badges == "" {
		c.Central.Badges = defaultBadgesDir
	}
	if !strings.HasPrefix(c.Central.Badges, "/") {
		c.Central.Badges = filepath.Clean(filepath.Join(c.Root(), c.Central.Badges))
	}

	return nil
}

func (c *Config) CentralReportsFS() (fs.ReadDirFS, error) {
	switch {
	case strings.HasPrefix(c.Central.Reports, "s3://"):
		splitted := strings.Split(strings.TrimPrefix(c.Central.Reports, "s3://"), "/")
		if len(splitted) == 0 {
			return nil, fmt.Errorf("invalid central.reports: %s", c.Central.Reports)
		}
		bucket := splitted[0]
		sess, err := session.NewSession()
		if err != nil {
			return nil, err
		}
		sc := s3.New(sess)
		s, err := datastore.NewS3(sc, bucket)
		if err != nil {
			return nil, err
		}
		return s.ReadDirFS(strings.Join(splitted[1:], "/"))
	default:
		l, err := datastore.NewLocal(c.Root())
		if err != nil {
			return nil, err
		}
		fsys, err := l.ReadDirDS(c.Central.Reports)
		if err != nil {
			return nil, err
		}
		return fsys, err
	}
}
