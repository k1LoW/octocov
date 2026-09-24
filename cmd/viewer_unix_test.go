//go:build !windows

package cmd

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestReadSourceRefusesAFIFO(t *testing.T) {
	// A path out of a report can name a FIFO, whose open would wait for a writer forever.
	dir := t.TempDir()
	if err := syscall.Mkfifo(filepath.Join(dir, "fifo.go"), 0o600); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()

	done := make(chan error, 1)
	go func() {
		_, err := readSource(root, "fifo.go")
		done <- err
	}()
	select {
	case err := <-done:
		if err == nil {
			t.Error("a FIFO must not be read as a source")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("reading a FIFO blocked")
	}
}
