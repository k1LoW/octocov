package report

import (
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/goccy/go-json"
	"github.com/k1LoW/octocov/pkg/coverage"
)

type Report struct {
	Repository string             `json:"repository"`
	Ref        string             `json:"ref"`
	Commit     string             `json:"commit"`
	Coverage   *coverage.Coverage `json:"coverage"`
}

func New() *Report {
	repo := os.Getenv("GITHUB_REPOSITORY")
	ref := os.Getenv("GITHUB_REF")
	if ref == "" {
		b, err := ioutil.ReadFile(".git/HEAD")
		if err == nil {
			splitted := strings.Split(strings.TrimSuffix(string(b), "\n"), " ")
			ref = splitted[1]
		}
	}
	commit := os.Getenv("GITHUB_SHA")
	if commit == "" {
		cmd := exec.Command("git", "rev-parse", "HEAD")
		b, err := cmd.Output()
		if err == nil {
			commit = strings.TrimSuffix(string(b), "\n")
		}
	}

	return &Report{
		Repository: repo,
		Ref:        ref,
		Commit:     commit,
	}
}

func (r *Report) String() string {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func (r *Report) MeasureCoverage(path string) error {
	cov, cerr := coverage.Measure(path)
	if cerr != nil {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, r); err != nil {
			return cerr
		}
		return nil
	}
	r.Coverage = cov
	return nil
}
