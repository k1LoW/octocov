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
			}
			continue
		}
		got := l.Root()
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestStoreReport(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	l, err := New(root)
	if err != nil {
		t.Fatal(err)
	}
	r := &report.Report{
		Repository: "owner/repo",
	}
	if err := l.StoreReport(ctx, r); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(l.Root(), "owner", "repo", "report.json")
	if _, err := os.Lstat(want); err != nil {
		t.Errorf("%s does not exist", want)
	}
}
