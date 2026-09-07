package cmd

import (
	"strings"
	"testing"

	"github.com/k1LoW/octocov/config"
	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/report"
)

func TestStoredArtifactViewer(t *testing.T) {
	tests := []struct {
		name       string
		repository string
		reportCfg  *config.Report
		want       string
	}{
		{"an artifact datastore is linked", "k1LoW/octocov", &config.Report{Datastores: []string{"artifact://k1LoW/octocov"}}, "https://octocov.dev/k1LoW/octocov/pull/722"},
		{"a report: that is not set links nowhere", "k1LoW/octocov", nil, ""},
		{"a report stored anywhere but an artifact links nowhere", "k1LoW/octocov", &config.Report{Datastores: []string{"s3://bucket/prefix"}}, ""},
		{"a report stored only in a file links nowhere", "k1LoW/octocov", &config.Report{Path: "report.json"}, ""},
		// The condition is what decides whether this run stores the report at all, so a
		// report.if: that does not hold has to leave the artifact unlinked. Here it cannot
		// be evaluated, which is one of the ways it does not hold.
		{"a report.if: that does not hold links nowhere", "", &config.Report{If: "is_default_branch", Datastores: []string{"artifact://k1LoW/octocov"}}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_REPOSITORY", "k1LoW/octocov")
			t.Setenv("GITHUB_SERVER_URL", "https://github.com")
			c := config.New()
			c.Repository = tt.repository
			c.Report = tt.reportCfg
			r := &report.Report{
				Repository:  "k1LoW/octocov",
				Ref:         "refs/pull/722/merge",
				BaseRef:     "refs/heads/main",
				PullRequest: 722,
				Commit:      "0123456789abcdef",
				Coverage:    &coverage.Coverage{Total: 100, Covered: 50},
			}

			v := storedArtifactViewer(c, r)
			if (v == nil) != (tt.want == "") {
				t.Fatalf("got viewer %v\nwant one only when there is a link", v)
			}
			// The viewer is observable only through what it renders, which is the thing the
			// gating is actually about.
			table := r.Table(v)
			switch {
			case tt.want == "":
				if strings.Contains(table, "octocov.dev") {
					t.Errorf("got\n%v\nwant no link", table)
				}
			default:
				if !strings.Contains(table, tt.want) {
					t.Errorf("got\n%v\nwant it to contain\n%v", table, tt.want)
				}
			}
		})
	}
}
