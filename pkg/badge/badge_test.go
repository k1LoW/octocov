package badge

import (
	"bytes"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/tenntenn/golden"
)

func TestRender(t *testing.T) {
	flag.Parse()

	tests := []struct {
		label    string
		message  string
		filename string
	}{
		{"coverage", "10%", "a"},
		{"code to test ratio", "1:1.3", "b"},
		{"テスト実行時間", "13sec", "c"},
	}
	for _, tt := range tests {
		b := New(tt.label, tt.message)
		got := new(bytes.Buffer)
		if err := b.Render(got); err != nil {
			t.Fatal(err)
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

func TestAddIconFile(t *testing.T) {
	b := New("with", "icon")
	b.AddIconFile(filepath.Join(testdataDir(t), "icon.svg"))
	got := new(bytes.Buffer)
	if err := b.Render(got); err != nil {
		t.Fatal(err)
	}
	filename := "add_icon"

	if os.Getenv("UPDATE_GOLDEN") != "" {
		golden.Update(t, testdataDir(t), filename, got)
		return
	}

	if diff := golden.Diff(t, testdataDir(t), filename, got); diff != "" {
		t.Error(diff)
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
