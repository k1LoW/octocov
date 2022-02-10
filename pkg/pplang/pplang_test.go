package pplang

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetect(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir, err := filepath.Abs(filepath.Dir(filepath.Dir(wd)))
	if err != nil {
		t.Fatal(err)
	}
	got, err := Detect(dir)
	if err != nil {
		t.Error(err)
	}
	if want := "Go"; got != want {
		t.Errorf("got %v\nwant %v", got, want)
	}
}
