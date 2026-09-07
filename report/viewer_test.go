package report

import "testing"

func TestViewerReportURL(t *testing.T) {
	tests := []struct {
		name         string
		artifactName string
		serverURL    string
		repository   string
		pullRequest  int
		want         string
	}{
		{"a pull request has a page of its own", "octocov-report@refs_pull_722", "", "k1LoW/octocov", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
		{"a run that is not on a pull request has none", "octocov-report", "", "k1LoW/octocov", 0, ""},
		{"a report of a sub directory is served under its repository", "octocov-report-sub@refs_pull_722", "", "k1LoW/octocov/sub", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
		{"a report stored in no artifact links nowhere", "", "", "k1LoW/octocov", 722, ""},
		{"a GitHub Enterprise Server run links nowhere", "octocov-report@refs_pull_722", "https://github.example.com", "k1LoW/octocov", 722, ""},
		{"github.com stated explicitly is served", "octocov-report@refs_pull_722", "https://github.com", "k1LoW/octocov", 722, "https://octocov.dev/k1LoW/octocov/pull/722"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			v := NewViewer(tt.artifactName)
			got := v.reportURL(&Report{Repository: tt.repository, PullRequest: tt.pullRequest, Commit: "0123456789abcdef"})
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
		commit       string
		path         string
		want         string
	}{
		{"a file is named by the artifact holding its report", "octocov-report@refs_pull_722", "", "0123456789abcdef", "report/report.go", "https://octocov.dev/k1LoW/octocov/file/report/report.go?artifact_name=octocov-report%40refs_pull_722&commit=0123456789abcdef"},
		{"the default branch is addressed the same way", "octocov-report", "", "0123456789abcdef", "report/report.go", "https://octocov.dev/k1LoW/octocov/file/report/report.go?artifact_name=octocov-report&commit=0123456789abcdef"},
		{"a separator inside a name does not end the path", "octocov-report", "", "0123456789abcdef", "some dir/a#b.go", "https://octocov.dev/k1LoW/octocov/file/some%20dir/a%23b.go?artifact_name=octocov-report&commit=0123456789abcdef"},
		{"a report of no commit links nowhere", "octocov-report", "", "", "report/report.go", ""},
		{"a report stored in no artifact links nowhere", "", "", "0123456789abcdef", "report/report.go", ""},
		{"a GitHub Enterprise Server run links nowhere", "octocov-report", "https://github.example.com", "0123456789abcdef", "report/report.go", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_SERVER_URL", tt.serverURL)
			v := NewViewer(tt.artifactName)
			got := v.fileURL(&Report{Repository: "k1LoW/octocov", Commit: tt.commit}, tt.path)
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
