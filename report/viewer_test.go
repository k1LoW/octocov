package report

import "testing"

func TestViewerReportURL(t *testing.T) {
	tests := []struct {
		name         string
		artifactName string
		serverURL    string
		repository   string
		ref          string
		baseRef      string
		pullRequest  int
		want         string
	}{
		{"a pull request has a page of its own", "octocov-report@refs_pull_722", "", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
		{"the default branch is the repository itself", "octocov-report", "", "k1LoW/octocov", "refs/heads/main", "refs/heads/main", 0, "https://octocov.dev/k1LoW/octocov"},
		{"a branch has a tree page", "octocov-report@refs_heads_feat_x", "", "k1LoW/octocov", "refs/heads/feat/x", "refs/heads/main", 0, "https://octocov.dev/k1LoW/octocov/tree/feat/x"},
		{"a branch name is escaped segment by segment", "octocov-report", "", "k1LoW/octocov", "refs/heads/feat/a b", "refs/heads/main", 0, "https://octocov.dev/k1LoW/octocov/tree/feat/a%20b"},
		{"a tag has no page of its own", "octocov-report", "", "k1LoW/octocov", "refs/tags/v1.0.0", "refs/heads/main", 0, ""},
		{"a report predating the base ref is read as the repository", "octocov-report", "", "k1LoW/octocov", "refs/heads/whatever", "", 0, "https://octocov.dev/k1LoW/octocov"},
		{"a report of a sub directory is served under its repository", "octocov-report-sub@refs_pull_722", "", "k1LoW/octocov/sub", "refs/pull/722/merge", "refs/heads/main", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
		{"a report stored in no artifact links nowhere", "", "", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, ""},
		{"a GitHub Enterprise Server run links nowhere", "octocov-report@refs_pull_722", "https://github.example.com", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, ""},
		{"github.com stated explicitly is served", "octocov-report@refs_pull_722", "https://github.com", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
		// The same server either way, so the slash must not read as another host.
		{"github.com with a trailing slash is served", "octocov-report@refs_pull_722", "https://github.com/", "k1LoW/octocov", "refs/pull/722/merge", "refs/heads/main", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			v := NewViewer(tt.artifactName)
			got := v.reportURL(&Report{Repository: tt.repository, Ref: tt.ref, BaseRef: tt.baseRef, PullRequest: tt.pullRequest, Commit: "0123456789abcdef"})
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestViewerFileURL(t *testing.T) {
	tests := []struct {
		name         string
		artifactName string
		serverURL    string
		path         string
		want         string
	}{
		{"a file is named by the artifact holding its report", "octocov-report@refs_pull_722", "", "report/report.go", "https://octocov.dev/k1LoW/octocov/file/report/report.go?artifact_name=octocov-report%40refs_pull_722"},
		{"the default branch is addressed the same way", "octocov-report", "", "report/report.go", "https://octocov.dev/k1LoW/octocov/file/report/report.go?artifact_name=octocov-report"},
		{"a separator inside a name does not end the path", "octocov-report", "", "some dir/a#b.go", "https://octocov.dev/k1LoW/octocov/file/some%20dir/a%23b.go?artifact_name=octocov-report"},
		{"an empty path links nowhere", "octocov-report", "", "", ""},
		{"a report stored in no artifact links nowhere", "", "", "report/report.go", ""},
		{"a GitHub Enterprise Server run links nowhere", "octocov-report", "https://github.example.com", "report/report.go", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			v := NewViewer(tt.artifactName)
			got := v.fileURL(&Report{Repository: "k1LoW/octocov", Commit: "0123456789abcdef"}, tt.path)
			if got != tt.want {
				t.Errorf("got %v\nwant %v", got, tt.want)
			}
		})
	}
}

func TestLinkCell(t *testing.T) {
	if got, want := linkCell("82.3%", ""), "82.3%"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
	if got, want := linkCell("82.3%", "https://octocov.dev/o/r/pull/1"), "[82.3%](https://octocov.dev/o/r/pull/1)"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}
