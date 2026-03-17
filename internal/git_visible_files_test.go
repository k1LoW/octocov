package internal

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func TestCollectGitVisibleFiles_IgnoresGitignoredCache(t *testing.T) {
	root := t.TempDir()
	repo := initGitRepo(t, root)

	repoFile := filepath.Join(root, "cmd", "main.go")
	cacheFile := filepath.Join(root, ".cache", "go-build", "deadbeef", "main.go")
	gitignoreFile := filepath.Join(root, ".gitignore")

	writeFile(t, repoFile)
	commitPaths(t, repo, "seed", "cmd/main.go")
	writeFile(t, gitignoreFile, ".cache/\n")
	writeFile(t, cacheFile)

	files, err := CollectGitVisibleFiles(root)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(files, repoFile) {
		t.Fatalf("CollectGitVisibleFiles() did not include tracked file %q: %v", repoFile, files)
	}
	if slices.Contains(files, cacheFile) {
		t.Fatalf("CollectGitVisibleFiles() unexpectedly included ignored cache file %q: %v", cacheFile, files)
	}
}

func TestCollectGitVisibleFiles_TrackedFileRemainsVisibleAfterIgnore(t *testing.T) {
	root := t.TempDir()
	repo := initGitRepo(t, root)

	trackedFile := filepath.Join(root, "tracked.go")
	writeFile(t, trackedFile)
	commitPaths(t, repo, "seed", "tracked.go")
	writeFile(t, filepath.Join(root, ".gitignore"), "tracked.go\n")

	files, err := CollectGitVisibleFiles(root)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(files, trackedFile) {
		t.Fatalf("CollectGitVisibleFiles() did not include tracked file %q after ignore: %v", trackedFile, files)
	}
}

func TestCollectGitVisibleFiles_RestrictsToWorkingDirectorySubtree(t *testing.T) {
	root := t.TempDir()
	repo := initGitRepo(t, root)

	cmdFile := filepath.Join(root, "cmd", "main.go")
	pkgFile := filepath.Join(root, "pkg", "helper.go")
	writeFile(t, cmdFile)
	writeFile(t, pkgFile)
	commitPaths(t, repo, "seed", "cmd/main.go", "pkg/helper.go")

	files, err := CollectGitVisibleFiles(filepath.Join(root, "cmd"))
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(files)

	want := []string{cmdFile}
	if !slices.Equal(files, want) {
		t.Fatalf("CollectGitVisibleFiles() = %v, want %v", files, want)
	}
}

func TestCollectGitVisibleFiles_AppliesAncestorGitignoreFromNestedWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	repo := initGitRepo(t, root)

	cmdFile := filepath.Join(root, "cmd", "main.go")
	ignoredFile := filepath.Join(root, "cmd", "main.gen.go")
	writeFile(t, cmdFile)
	commitPaths(t, repo, "seed", "cmd/main.go")
	writeFile(t, filepath.Join(root, ".gitignore"), "*.gen.go\n")
	writeFile(t, ignoredFile)

	files, err := CollectGitVisibleFiles(filepath.Join(root, "cmd"))
	if err != nil {
		t.Fatal(err)
	}

	if slices.Contains(files, ignoredFile) {
		t.Fatalf("CollectGitVisibleFiles() unexpectedly included ancestor-ignored file %q: %v", ignoredFile, files)
	}
	if !slices.Contains(files, cmdFile) {
		t.Fatalf("CollectGitVisibleFiles() did not include tracked file %q: %v", cmdFile, files)
	}
}

func TestCollectGitVisibleFiles_AppliesNestedGitignoreWithinWorkingDirectory(t *testing.T) {
	root := t.TempDir()
	repo := initGitRepo(t, root)

	cmdFile := filepath.Join(root, "cmd", "main.go")
	visibleFile := filepath.Join(root, "cmd", "internal", "helper.go")
	ignoredFile := filepath.Join(root, "cmd", "internal", "helper.tmp.go")
	writeFile(t, cmdFile)
	commitPaths(t, repo, "seed", "cmd/main.go")
	writeFile(t, filepath.Join(root, "cmd", "internal", ".gitignore"), "*.tmp.go\n")
	writeFile(t, visibleFile)
	writeFile(t, ignoredFile)

	files, err := CollectGitVisibleFiles(filepath.Join(root, "cmd"))
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Contains(files, visibleFile) {
		t.Fatalf("CollectGitVisibleFiles() did not include visible file %q: %v", visibleFile, files)
	}
	if slices.Contains(files, ignoredFile) {
		t.Fatalf("CollectGitVisibleFiles() unexpectedly included nested-ignored file %q: %v", ignoredFile, files)
	}
}

func initGitRepo(t *testing.T, root string) *git.Repository {
	t.Helper()

	repo, err := git.PlainInit(root, false)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func commitPaths(t *testing.T, repo *git.Repository, message string, paths ...string) {
	t.Helper()

	w, err := repo.Worktree()
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range paths {
		if _, err := w.Add(path); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "octocov test",
			Email: "octocov@example.com",
			When:  time.Unix(0, 0).UTC(),
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, contents ...string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	body := ""
	if len(contents) > 0 {
		body = contents[0]
	}
	if err := os.WriteFile(path, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}
