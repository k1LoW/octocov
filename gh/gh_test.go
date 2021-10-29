package gh

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    *Repository
		wantErr bool
	}{
		{"owner/repo", &Repository{Owner: "owner", Repo: "repo"}, false},
		{"owner/repo/path/to", &Repository{Owner: "owner", Repo: "repo", Path: "path/to"}, false},
		{"owner/repo@sub", &Repository{Owner: "owner", Repo: "repo@sub"}, false},
		{"owner/repo.sub", &Repository{Owner: "owner", Repo: "repo.sub"}, false},
		{"owner/../sub", nil, true},
		{"owner", nil, true},
		{"owner/../sub", nil, true},
		{"owner/./sub", nil, true},
		{"owner//sub", nil, true},
		{"owner/repo/sub/", nil, true},
	}
	for _, tt := range tests {
		got, err := Parse(tt.in)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("got error %v\n", err)
			}
			continue
		}
		if diff := cmp.Diff(got, tt.want, nil); diff != "" {
			t.Errorf("%s", diff)
		}
	}
}
