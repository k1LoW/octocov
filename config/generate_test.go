package config

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/tenntenn/golden"
)

func TestGenerate(t *testing.T) {
	ctx := context.TODO()
	tests := []struct {
		filename string
		lang     string
	}{
		{"base_octocov.yml", ""},
		{"go_octocov.yml", "Go"},
		{"base_octocov.yml", "Unknown"},
	}
	for _, tt := range tests {
		got := new(bytes.Buffer)
		if err := Generate(ctx, tt.lang, got); err != nil {
			t.Error(err)
		}

		if os.Getenv("UPDATE_GOLDEN") != "" {
			golden.Update(t, testdataDir(t), tt.filename, got)
			continue
		}

		if diff := golden.Diff(t, testdataDir(t), tt.filename, got); diff != "" {
			t.Error(diff)
		}
	}
}

func testdataDir(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Join(wd, "testdata"))
	if err != nil {
		t.Fatal(err)
	}
	return dir
}
