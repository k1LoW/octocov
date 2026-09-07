package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/k1LoW/octocov/report"
)

func TestRoot(t *testing.T) {
	td := t.TempDir()
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{td, td, false},
		{"invalid_dir", "", true},
	}
	for _, tt := range tests {
		l, err := New(tt.in)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("got err %v\n", err)
			}
			continue
		} else {
			if tt.wantErr {
				t.Error("want err")
				continue
			}
		}
		got := l.Root()
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestStoreReport(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name        string
		ref         string
		baseRef     string
		pullRequest int
		want        []string
	}{
		{"the default branch is stored where comparisons read it", "refs/heads/main", "refs/heads/main", 0, []string{"owner", "repo", "report.json"}},
		{"a report without a base ref is stored there too", "", "", 0, []string{"owner", "repo", "report.json"}},
		{"a pull request is stored beside it", "refs/pull/123/merge", "refs/heads/main", 123, []string{"owner", "repo", "refs", "pull", "123", "report.json"}},
		{"a branch is stored beside it", "refs/heads/feat/x", "refs/heads/main", 0, []string{"owner", "repo", "refs", "heads", "feat", "x", "report.json"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := New(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			r := &report.Report{
				Repository:  "owner/repo",
				Ref:         tt.ref,
				BaseRef:     tt.baseRef,
				PullRequest: tt.pullRequest,
			}
			if err := l.StoreReport(ctx, r); err != nil {
				t.Fatal(err)
			}
			want := filepath.Join(append([]string{l.Root()}, tt.want...)...)
			if _, err := os.Lstat(want); err != nil {
				t.Errorf("%s does not exist", want)
			}
		})
	}
}
