package gh

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestParse(t *testing.T) {
	tests := []struct {
		in      string
		want    *Repogitory
		wantErr bool
	}{
		{"owner/repo", &Repogitory{Owner: "owner", Repo: "repo"}, false},
		{"owner/repo/path/to", &Repogitory{Owner: "owner", Repo: "repo", Path: "path/to"}, false},
		{"owner", nil, true},
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
