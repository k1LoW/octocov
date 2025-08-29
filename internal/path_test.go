package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRootPath(t *testing.T) {
	t.Run("git config", func(t *testing.T) {
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
			got, err := RootPath(tt.base)
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
	})

	t.Run("octocov.yml", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "x", "y", "z"), 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(filepath.Join(dir, "x", ".octocov.yml"))
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		tests := []struct {
			base    string
			wantErr bool
		}{
			{filepath.Join(dir, "x", "y"), false},
			{filepath.Join(dir, "x", "y", "z"), false},
			{filepath.Join(dir, "x"), false},
			{dir, true},
		}
		for _, tt := range tests {
			got, err := RootPath(tt.base)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
				}
			} else {
				if tt.wantErr {
					t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
				}
				if want := filepath.Join(dir, "x"); got != want {
					t.Errorf("got %v\nwant %v", got, want)
				}
			}
		}
	})

	t.Run("octocov.yml (no dot) config file", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "a", "b", "c"), 0700); err != nil {
			t.Fatal(err)
		}
		// Create octocov.yml (without dot)
		f, err := os.Create(filepath.Join(dir, "a", "octocov.yml"))
		if err != nil {
			t.Fatal(err)
		}
		f.Close()

		tests := []struct {
			base    string
			wantErr bool
		}{
			{filepath.Join(dir, "a", "b"), false},
			{filepath.Join(dir, "a", "b", "c"), false},
			{filepath.Join(dir, "a"), false},
			{dir, true},
		}
		for _, tt := range tests {
			got, err := RootPath(tt.base)
			if err != nil {
				if !tt.wantErr {
					t.Errorf("got %v\nwantErr %v", err, tt.wantErr)
				}
			} else {
				if tt.wantErr {
					t.Errorf("got %v\nwantErr %v", nil, tt.wantErr)
				}
				if want := filepath.Join(dir, "a"); got != want {
					t.Errorf("got %v\nwant %v", got, want)
				}
			}
		}
	})

	t.Run("octocov.yml found first", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "p", "q", "r"), 0700); err != nil {
			t.Fatal(err)
		}

		// Create .octocov.yml at deeper level
		f1, err := os.Create(filepath.Join(dir, "p", "q", ".octocov.yml"))
		if err != nil {
			t.Fatal(err)
		}
		f1.Close()

		// Create .git/config at shallower level
		if err := os.Mkdir(filepath.Join(dir, "p", ".git"), 0700); err != nil {
			t.Fatal(err)
		}
		f2, err := os.Create(filepath.Join(dir, "p", ".git", "config"))
		if err != nil {
			t.Fatal(err)
		}
		f2.Close()

		got, err := RootPath(filepath.Join(dir, "p", "q", "r"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// .octocov.yml is found first when traversing from r -> q -> p
		if want := filepath.Join(dir, "p", "q"); got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	})

	t.Run("multiple config files - priority test", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "test", "sub"), 0700); err != nil {
			t.Fatal(err)
		}

		// Create both .octocov.yml and octocov.yml in the same directory
		// .octocov.yml should be checked first in the slice
		f1, err := os.Create(filepath.Join(dir, "test", ".octocov.yml"))
		if err != nil {
			t.Fatal(err)
		}
		f1.Close()

		f2, err := os.Create(filepath.Join(dir, "test", "octocov.yml"))
		if err != nil {
			t.Fatal(err)
		}
		f2.Close()

		got, err := RootPath(filepath.Join(dir, "test", "sub"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// Should find the directory with config files
		if want := filepath.Join(dir, "test"); got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	})

	t.Run("both files in same directory - git config takes precedence", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.MkdirAll(filepath.Join(dir, "m", "n"), 0700); err != nil {
			t.Fatal(err)
		}

		// Create both files in the same directory
		if err := os.Mkdir(filepath.Join(dir, "m", ".git"), 0700); err != nil {
			t.Fatal(err)
		}
		f1, err := os.Create(filepath.Join(dir, "m", ".git", "config"))
		if err != nil {
			t.Fatal(err)
		}
		f1.Close()

		f2, err := os.Create(filepath.Join(dir, "m", ".octocov.yml"))
		if err != nil {
			t.Fatal(err)
		}
		f2.Close()

		got, err := RootPath(filepath.Join(dir, "m", "n"))
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		// When both files exist in the same directory, .git/config takes precedence
		if want := filepath.Join(dir, "m"); got != want {
			t.Errorf("got %v\nwant %v", got, want)
		}
	})
}
