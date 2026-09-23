package artifact

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/k1LoW/octocov/gh"
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

type fakeClient struct {
	putErr    error
	deleteErr error
	calls     []string
}

func (c *fakeClient) PutArtifact(ctx context.Context, owner, repo string, runID int64, name, fp string, content []byte) error {
	c.calls = append(c.calls, "put "+name)
	return c.putErr
}

func (c *fakeClient) DeleteArtifactsBeforeRun(ctx context.Context, owner, repo, name string, runID int64) error {
	c.calls = append(c.calls, "delete "+name)
	return c.deleteErr
}

func (c *fakeClient) FetchLatestArtifact(ctx context.Context, owner, repo, name, fp string) (*gh.ArtifactFile, error) {
	return nil, errors.New("not implemented")
}

func TestStoreReport(t *testing.T) {
	tests := []struct {
		name        string
		pullRequest int
		putErr      error
		deleteErr   error
		wantCalls   []string
		wantErr     bool
		wantStderr  string
	}{
		{"a pull request deletes the earlier reports after storing its own", 123, nil, nil, []string{"put octocov-report@refs_pull_123", "delete octocov-report@refs_pull_123"}, false, ""},
		{"a branch keeps the earlier reports", 0, nil, nil, []string{"put octocov-report@refs_heads_feat_x"}, false, ""},
		{"a failed upload deletes nothing", 123, errors.New("upload failed"), nil, []string{"put octocov-report@refs_pull_123"}, true, ""},
		{"a failed delete is a warning", 123, nil, errors.New("403 Resource not accessible by integration"), []string{"put octocov-report@refs_pull_123", "delete octocov-report@refs_pull_123"}, false, "Skip deleting the previous reports of octocov-report@refs_pull_123: 403 Resource not accessible by integration\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("GITHUB_REPOSITORY", "owner/repo")
			t.Setenv("GITHUB_RUN_ID", "20")
			c := &fakeClient{putErr: tt.putErr, deleteErr: tt.deleteErr}
			stderr := new(bytes.Buffer)
			a := &Artifact{gh: c, repository: "owner/repo", name: defaultArtifactName, stderr: stderr}
			ref := "refs/heads/feat/x"
			if tt.pullRequest > 0 {
				ref = "refs/pull/123/merge"
			}
			err := a.StoreReport(context.TODO(), &report.Report{
				Repository:  "owner/repo",
				Ref:         ref,
				BaseRef:     "refs/heads/main",
				PullRequest: tt.pullRequest,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("got err %v, want err %v", err, tt.wantErr)
			}
			if diff := cmp.Diff(c.calls, tt.wantCalls, nil); diff != "" {
				t.Error(diff)
			}
			if got := stderr.String(); got != tt.wantStderr {
				t.Errorf("got %q\nwant %q", got, tt.wantStderr)
			}
		})
	}
}
