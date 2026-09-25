package cmd

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

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
			if tt.want == "" {
				if strings.Contains(table, "octocov.dev") {
					t.Errorf("got\n%v\nwant no link", table)
				}
				return
			}
			if !strings.Contains(table, tt.want) {
				t.Errorf("got\n%v\nwant it to contain\n%v", table, tt.want)
			}
		})
	}
}

func TestViewersFor(t *testing.T) {
	stored := report.NewOctocovDevViewer("octocov-metadata-octocov-report@refs_pull_722")
	tests := []struct {
		name             string
		stored           *report.Viewer
		comparedArtifact string
		wantCur          bool
		wantPrev         bool
	}{
		{"both sides when both came from an artifact", stored, "octocov-metadata-octocov-report@refs_heads_main", true, true},
		{"the compared side alone stays unlinked", stored, "", true, false},
		// A run storing no artifact of its own links nothing, so that every link in a
		// comment belongs to a run that wrote one.
		{"a run storing no artifact links neither side", nil, "octocov-metadata-octocov-report@refs_heads_main", false, false},
		{"neither side when there is nothing either way", nil, "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cur, prev := viewersFor(tt.stored, tt.comparedArtifact)
			if (cur != nil) != tt.wantCur {
				t.Errorf("got cur %v\nwant one: %v", cur, tt.wantCur)
			}
			if (prev != nil) != tt.wantPrev {
				t.Errorf("got prev %v\nwant one: %v", prev, tt.wantPrev)
			}
		})
	}
}

func TestReadBaseReport(t *testing.T) {
	stored := func(commit string, ts int) *fstest.MapFile {
		return &fstest.MapFile{Data: fmt.Appendf(nil, `{"commit":%q,"timestamp":"2026-09-%02dT00:00:00Z"}`, commit, ts)}
	}
	tests := []struct {
		name             string
		stores           []comparedDatastore
		keys             []string
		want             string
		wantMetadataRead string
	}{
		{
			"the report of the base branch is read",
			[]comparedDatastore{{name: "s3", fsys: fstest.MapFS{
				"owner/repo/report.json":                    stored("main", 2),
				"owner/repo/refs/heads/develop/report.json": stored("develop", 1),
			}}},
			[]string{"refs/heads/develop", ""},
			"develop",
			"",
		},
		{
			"the default branch is read where the base branch has no report",
			[]comparedDatastore{{name: "s3", fsys: fstest.MapFS{
				"owner/repo/report.json": stored("main", 1),
			}}},
			[]string{"refs/heads/develop", ""},
			"main",
			"",
		},
		{
			"the base branch of one datastore wins over a newer default branch of another",
			[]comparedDatastore{
				{name: "s3", fsys: fstest.MapFS{"owner/repo/report.json": stored("main", 2)}},
				{name: "artifact", fsys: fstest.MapFS{"owner/repo/refs/heads/develop/report.json": stored("develop", 1)}, metadataRead: "octocov-metadata-octocov-report@refs_heads_develop"},
			},
			[]string{"refs/heads/develop", ""},
			"develop",
			"octocov-metadata-octocov-report@refs_heads_develop",
		},
		{
			"the newest of the datastores holding the key is read",
			[]comparedDatastore{
				{name: "artifact", fsys: fstest.MapFS{"owner/repo/report.json": stored("older", 1)}, metadataRead: "octocov-metadata-octocov-report@refs_heads_main"},
				{name: "s3", fsys: fstest.MapFS{"owner/repo/report.json": stored("newer", 2)}},
			},
			[]string{""},
			"newer",
			"",
		},
		{
			"nothing is read where no datastore holds a report",
			[]comparedDatastore{{name: "s3", fsys: fstest.MapFS{}}},
			[]string{"refs/heads/develop", ""},
			"",
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, metadataRead := readBaseReport(tt.stores, "owner/repo", tt.keys)
			var commit string
			if got != nil {
				commit = got.Commit
			}
			if commit != tt.want {
				t.Errorf("got %v\nwant %v", commit, tt.want)
			}
			if metadataRead != tt.wantMetadataRead {
				t.Errorf("got %v\nwant %v", metadataRead, tt.wantMetadataRead)
			}
		})
	}
}
