package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetRootPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "a", "b", "c", "d"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "a", "b", ".git"), 0700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(filepath.Join(dir, "a", "b", ".git", "config"))
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	tests := []struct {
		base    string
		wantErr bool
	}{
		{filepath.Join(dir, "a", "b", "c"), false},
		{filepath.Join(dir, "a", "b", "c", "d"), false},
		{filepath.Join(dir, "a", "b"), false},
		{filepath.Join(dir, "a"), true},
	}
	for _, tt := range tests {
		got, err := GetRootPath(tt.base)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
			}
		} else {
			if tt.wantErr {
				t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
			}
			if want := filepath.Join(dir, "a", "b"); got != want {
				t.Errorf("got %v\nwant %v", got, want)
			}
		}
	}
}
