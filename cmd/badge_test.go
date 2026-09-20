package cmd

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteOutKeepsTheFileWhenTheRenderFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "coverage.svg")
	if err := os.WriteFile(path, []byte("previous badge"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := writeOut(path, func(io.Writer) error { return errors.New("invalid color") }); err == nil {
		t.Fatal("want error")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := "previous badge"; string(got) != want {
		t.Errorf("got %v\nwant %v", string(got), want)
	}
}

func TestBadgeFileKeepsTheFileWhenTheRenderFails(t *testing.T) {
	path := filepath.Join(t.TempDir(), "badges", "coverage.svg")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("previous badge"), 0600); err != nil {
		t.Fatal(err)
	}

	if err := badgeFile(path, func(io.Writer) error { return errors.New("invalid color") }); err == nil {
		t.Fatal("want error")
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := "previous badge"; string(got) != want {
		t.Errorf("got %v\nwant %v", string(got), want)
	}
}
