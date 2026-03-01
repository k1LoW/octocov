package internal

import (
	"os"
	"path/filepath"
	"slices"
	"sort"
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

	for _, path := range ConfigPaths {
		t.Run(path, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.MkdirAll(filepath.Join(dir, "x", "y", "z"), 0700); err != nil {
				t.Fatal(err)
			}
			f, err := os.Create(filepath.Join(dir, "x", path))
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
	}

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

func TestCollectFiles(t *testing.T) {
	root := t.TempDir()

	// Create directory structure:
	// root/
	//   main.go
	//   cmd/root.go
	//   .git/config        (should be skipped)
	//   node_modules/x.js  (should be skipped)
	//   vendor/v.go         (should be skipped)
	//   src/app.ts
	dirs := []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, ".git"),
		filepath.Join(root, "node_modules"),
		filepath.Join(root, "vendor"),
		filepath.Join(root, "src"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}

	filesToCreate := map[string]bool{
		filepath.Join(root, "main.go"):           true,
		filepath.Join(root, "cmd", "root.go"):    true,
		filepath.Join(root, "src", "app.ts"):     true,
		filepath.Join(root, ".git", "config"):    false, // should be skipped
		filepath.Join(root, "node_modules", "x.js"): false, // should be skipped
		filepath.Join(root, "vendor", "v.go"):       false, // should be skipped
	}
	for f := range filesToCreate {
		if err := os.WriteFile(f, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := CollectFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(got)

	var want []string
	for f, included := range filesToCreate {
		if included {
			want = append(want, f)
		}
	}
	sort.Strings(want)

	if !slices.Equal(got, want) {
		t.Errorf("CollectFiles() =\n  %v\nwant\n  %v", got, want)
	}
}

func TestCollectFiles_Empty(t *testing.T) {
	root := t.TempDir()
	got, err := CollectFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty result for empty directory, got %v", got)
	}
}

func TestCollectFiles_NestedSkipDirs(t *testing.T) {
	root := t.TempDir()

	// __pycache__ nested inside src/ should also be skipped
	dirs := []string{
		filepath.Join(root, "src", "__pycache__"),
		filepath.Join(root, "src"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}

	files := []string{
		filepath.Join(root, "src", "app.py"),
		filepath.Join(root, "src", "__pycache__", "app.pyc"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}
	}

	got, err := CollectFiles(root)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{filepath.Join(root, "src", "app.py")}
	sort.Strings(got)

	if !slices.Equal(got, want) {
		t.Errorf("CollectFiles() = %v, want %v", got, want)
	}
}
