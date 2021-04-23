package report

import (
	"fmt"
	"os"

	"github.com/k1LoW/octocov/pkg/coverage"
)

type Report struct {
	Repository string             `json:"repository"`
	Ref        string             `json:"ref"`
	Commit     string             `json:"commit"`
	Coverage   *coverage.Coverage `json:"coverage"`
}

func New() *Report {
	return &Report{
		Repository: os.Getenv("GITHUB_REPOSITORY"),
		Ref:        os.Getenv("GITHUB_REF"),
		Commit:     os.Getenv("GITHUB_SHA"),
	}
}

func (r *Report) MeasureCoverage(path string) error {
	// gocover
	if cov, err := coverage.NewGocover().ParseReport(path); err == nil {
		r.Coverage = cov
		return nil
	}
	// lcov
	if cov, err := coverage.NewLcov().ParseReport(path); err == nil {
		r.Coverage = cov
		return nil
	}
	// simplecov
	if cov, err := coverage.NewSimplecov().ParseReport(path); err == nil {
		r.Coverage = cov
		return nil
	}
	return fmt.Errorf("coverage report not found: %s", path)
}
