package artifact

import (
	"strings"
	"testing"

	"github.com/k1LoW/octocov/report"
)

func TestStoreName(t *testing.T) {
	tests := []struct {
		name        string
		artifact    string
		repository  string
		ref         string
		baseRef     string
		pullRequest int
		want        string
		wantErr     bool
	}{
		{"the default branch keeps the configured name", "", "owner/repo", "refs/heads/main", "refs/heads/main", 0, "octocov-report", false},
		{"a pull request is marked off by its ref", "", "owner/repo", "refs/pull/123/merge", "refs/heads/main", 123, "octocov-report@refs_pull_123", false},
		{"a branch keeps the separators of its name out of the ref", "", "owner/repo", "refs/heads/feat/x", "refs/heads/main", 0, "octocov-report@refs_heads_feat_x", false},
		{"a configured name is marked off the same way", "mine", "owner/repo", "refs/pull/123/merge", "refs/heads/main", 123, "mine@refs_pull_123", false},
		{"a configured name already holding the separator is split on the last one", "mine@v2", "owner/repo", "refs/pull/123/merge", "refs/heads/main", 123, "mine@v2@refs_pull_123", false},
		{"a sub directory keeps its key ahead of the ref", "", "owner/repo/sub", "refs/pull/123/merge", "refs/heads/main", 123, "octocov-report-sub@refs_pull_123", false},
		{"a sub directory on the default branch keeps its key alone", "", "owner/repo/sub", "refs/heads/main", "refs/heads/main", 0, "octocov-report-sub", false},
		{"a report of another repository is refused", "", "other/repo", "refs/heads/main", "refs/heads/main", 0, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_REPOSITORY", "owner/repo")
			a, err := New(nil, "owner/repo", tt.artifact, nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := a.StoreName(&report.Report{
				Repository:  tt.repository,
				Ref:         tt.ref,
				BaseRef:     tt.baseRef,
				PullRequest: tt.pullRequest,
			})
			if err != nil {
				if !tt.wantErr {
					t.Errorf("got err: %v", err)
				}
				return
			}
			if tt.wantErr {
				t.Error("want err")
				return
			}
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestStoreNameHasNoInvalidCharacter(t *testing.T) {
	t.Setenv("GITHUB_REPOSITORY", "owner/repo")
	a, err := New(nil, "owner/repo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := a.StoreName(&report.Report{
		Repository: "owner/repo",
		Ref:        `refs/heads/feat/"a:b<c>d|e*f?g\h`,
		BaseRef:    "refs/heads/main",
	})
	if err != nil {
		t.Fatal(err)
	}
	// The characters GitHub refuses in an artifact name, which a ref may hold.
	for _, c := range []string{`"`, ":", "<", ">", "|", "*", "?", "\r", "\n", `\`, "/"} {
		if strings.Contains(strings.TrimPrefix(got, "octocov-report"), c) {
			t.Errorf("got %v\nwant no %q", got, c)
		}
	}
}
