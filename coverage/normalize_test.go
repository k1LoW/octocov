package coverage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestEffectivePath(t *testing.T) {
	tests := []struct {
		name           string
		file           string
		normalizedPath string
		want           string
	}{
		{"NormalizedPath set", "github.com/user/repo/cmd/main.go", "cmd/main.go", "cmd/main.go"},
		{"NormalizedPath empty", "github.com/user/repo/cmd/main.go", "", "github.com/user/repo/cmd/main.go"},
		{"Both empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := &FileCoverage{
				File:           tt.file,
				NormalizedPath: tt.normalizedPath,
			}
			if got := fc.EffectivePath(); got != tt.want {
				t.Errorf("EffectivePath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNormalizePaths(t *testing.T) {
	root := t.TempDir()

	// Create directory structure:
	// root/
	//   cmd/root.go
	//   cmd/sub/main.go
	//   internal/frontend/src/utils/groups.ts
	//   pkg/handler.go
	dirs := []string{
		filepath.Join(root, "cmd", "sub"),
		filepath.Join(root, "internal", "frontend", "src", "utils"),
		filepath.Join(root, "pkg"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	files := []string{
		filepath.Join(root, "cmd", "root.go"),
		filepath.Join(root, "cmd", "sub", "main.go"),
		filepath.Join(root, "internal", "frontend", "src", "utils", "groups.ts"),
		filepath.Join(root, "pkg", "handler.go"),
	}
	for _, f := range files {
		if err := os.WriteFile(f, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name           string
		file           string
		wantNormalized string
	}{
		{
			"Go module path (suffix match: cmd/root.go)",
			"github.com/k1LoW/myrepo/cmd/root.go",
			"cmd/root.go",
		},
		{
			"Relative path from test tool (suffix match: src/utils/groups.ts)",
			"src/utils/groups.ts",
			"internal/frontend/src/utils/groups.ts",
		},
		{
			"Absolute path within root",
			filepath.Join(root, "pkg", "handler.go"),
			"pkg/handler.go",
		},
		{
			"Already relative path matching exactly",
			"cmd/sub/main.go",
			"cmd/sub/main.go",
		},
		{
			"No match found",
			"nonexistent/file.go",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cov := &Coverage{
				Files: FileCoverages{
					{File: tt.file},
				},
			}
			cov.NormalizePaths(root, files)
			got := cov.Files[0].NormalizedPath
			if got != tt.wantNormalized {
				t.Errorf("NormalizedPath = %q, want %q", got, tt.wantNormalized)
			}
		})
	}
}

func TestNormalizePaths_NilAndEmpty(t *testing.T) {
	t.Run("nil coverage", func(t *testing.T) {
		var c *Coverage
		c.NormalizePaths("/root", []string{"/root/file.go"})
	})

	t.Run("empty fsFiles", func(t *testing.T) {
		c := &Coverage{
			Files: FileCoverages{
				{File: "file.go"},
			},
		}
		c.NormalizePaths("/root", nil)
		if c.Files[0].NormalizedPath != "" {
			t.Errorf("expected empty NormalizedPath, got %q", c.Files[0].NormalizedPath)
		}
	})

	t.Run("empty root", func(t *testing.T) {
		c := &Coverage{
			Files: FileCoverages{
				{File: "file.go"},
			},
		}
		c.NormalizePaths("", []string{"/root/file.go"})
		if c.Files[0].NormalizedPath != "" {
			t.Errorf("expected empty NormalizedPath, got %q", c.Files[0].NormalizedPath)
		}
	})
}

func TestNormalizePaths_MultipleFiles(t *testing.T) {
	root := t.TempDir()

	dirs := []string{
		filepath.Join(root, "cmd"),
		filepath.Join(root, "internal", "frontend", "src", "utils"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0700); err != nil {
			t.Fatal(err)
		}
	}
	fsFiles := []string{
		filepath.Join(root, "cmd", "root.go"),
		filepath.Join(root, "internal", "frontend", "src", "utils", "groups.ts"),
	}
	for _, f := range fsFiles {
		if err := os.WriteFile(f, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}
	}

	cov := &Coverage{
		Files: FileCoverages{
			{File: "github.com/k1LoW/mo/cmd/root.go"},
			{File: "src/utils/groups.ts"},
		},
	}
	cov.NormalizePaths(root, fsFiles)

	if got := cov.Files[0].NormalizedPath; got != "cmd/root.go" {
		t.Errorf("Files[0].NormalizedPath = %q, want %q", got, "cmd/root.go")
	}
	if got := cov.Files[1].NormalizedPath; got != "internal/frontend/src/utils/groups.ts" {
		t.Errorf("Files[1].NormalizedPath = %q, want %q", got, "internal/frontend/src/utils/groups.ts")
	}
}

func TestFindByFile_WithNormalizedPath(t *testing.T) {
	fcs := FileCoverages{
		{File: "github.com/k1LoW/repo/cmd/root.go", NormalizedPath: "cmd/root.go", Total: 10, Covered: 5},
		{File: "src/utils/groups.ts", NormalizedPath: "internal/frontend/src/utils/groups.ts", Total: 20, Covered: 15},
	}

	tests := []struct {
		name    string
		file    string
		want    string
		wantErr bool
	}{
		{"find by NormalizedPath", "cmd/root.go", "github.com/k1LoW/repo/cmd/root.go", false},
		{"find by original File", "github.com/k1LoW/repo/cmd/root.go", "github.com/k1LoW/repo/cmd/root.go", false},
		{"find by NormalizedPath (ts)", "internal/frontend/src/utils/groups.ts", "src/utils/groups.ts", false},
		{"not found", "nonexistent.go", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc, err := fcs.FindByFile(tt.file)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fc.File != tt.want {
				t.Errorf("found File = %q, want %q", fc.File, tt.want)
			}
		})
	}
}

func TestFuzzyFindByFile_WithNormalizedPath(t *testing.T) {
	fcs := FileCoverages{
		{File: "github.com/k1LoW/repo/cmd/root.go", NormalizedPath: "cmd/root.go"},
		{File: "/abs/path/to/pkg/handler.go", NormalizedPath: "pkg/handler.go"},
	}

	tests := []struct {
		name    string
		file    string
		want    string
		wantErr bool
	}{
		{"fuzzy match by NormalizedPath suffix", "./cmd/root.go", "github.com/k1LoW/repo/cmd/root.go", false},
		{"fuzzy match by NormalizedPath - handler.go", "some/path/pkg/handler.go", "/abs/path/to/pkg/handler.go", false},
		{"no match at all", "nonexistent/xyz.py", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc, err := fcs.FuzzyFindByFile(tt.file)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fc.File != tt.want {
				t.Errorf("found File = %q, want %q", fc.File, tt.want)
			}
		})
	}
}

func TestCompare_WithNormalizedPaths(t *testing.T) {
	a := &Coverage{
		Total:   100,
		Covered: 54,
		Files: FileCoverages{
			{File: "github.com/user/repo/file_a.go", NormalizedPath: "file_a.go", Total: 60, Covered: 39},
			{File: "src/file_b.ts", NormalizedPath: "internal/src/file_b.ts", Total: 40, Covered: 15},
		},
	}

	b := &Coverage{
		Total:   100,
		Covered: 54,
		Files: FileCoverages{
			{File: "github.com/user/repo/file_a.go", NormalizedPath: "file_a.go", Total: 60, Covered: 39},
			{File: "src/file_b.ts", NormalizedPath: "internal/src/file_b.ts", Total: 40, Covered: 15},
		},
	}

	got := a.Compare(b)

	want := &DiffCoverage{
		A:    54.0,
		B:    54.0,
		Diff: 0.0,
		Files: DiffFileCoverages{
			{File: "file_a.go", A: 65.0, B: 65.0, Diff: 0.0},
			{File: "internal/src/file_b.ts", A: 37.5, B: 37.5, Diff: 0.0},
		},
	}

	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(DiffCoverage{}),
		cmpopts.IgnoreFields(DiffCoverage{}, "CoverageA", "CoverageB"),
		cmpopts.SortSlices(func(i, j *DiffFileCoverage) bool {
			return i.File < j.File
		}),
		cmpopts.IgnoreFields(DiffFileCoverage{}, "FileCoverageA", "FileCoverageB"),
	}

	if diff := cmp.Diff(got, want, opts...); diff != "" {
		t.Error(diff)
	}
}

func TestCompare_MixedFormats_MatchByNormalizedPath(t *testing.T) {
	// Simulates: current report from Go cover + LCOV, previous report from same
	// Both have different File paths but same NormalizedPath
	a := &Coverage{
		Total:   100,
		Covered: 60,
		Files: FileCoverages{
			{File: "github.com/user/repo/cmd/main.go", NormalizedPath: "cmd/main.go", Total: 50, Covered: 30},
			{File: "src/utils.ts", NormalizedPath: "frontend/src/utils.ts", Total: 50, Covered: 30},
		},
	}

	b := &Coverage{
		Total:   100,
		Covered: 50,
		Files: FileCoverages{
			// Same normalized path, different original File
			{File: "github.com/user/repo/cmd/main.go", NormalizedPath: "cmd/main.go", Total: 50, Covered: 25},
			{File: "src/utils.ts", NormalizedPath: "frontend/src/utils.ts", Total: 50, Covered: 25},
		},
	}

	got := a.Compare(b)

	// Should have exactly 2 diff files (not 4), because NormalizedPath matches
	if len(got.Files) != 2 {
		t.Errorf("expected 2 DiffFileCoverages, got %d", len(got.Files))
		for _, f := range got.Files {
			t.Logf("  file: %s", f.File)
		}
	}

	// Both files should have both A and B coverage
	for _, f := range got.Files {
		if f.FileCoverageA == nil {
			t.Errorf("file %q: FileCoverageA is nil", f.File)
		}
		if f.FileCoverageB == nil {
			t.Errorf("file %q: FileCoverageB is nil", f.File)
		}
	}
}

func TestMerge_WithNormalizedPaths(t *testing.T) {
	// When two coverages have different File but same NormalizedPath,
	// they should be merged into one entry
	c1 := &Coverage{
		Type: TypeLOC,
		Files: FileCoverages{
			{
				File:           "github.com/user/repo/cmd/root.go",
				NormalizedPath: "cmd/root.go",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
					newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
				},
			},
		},
	}

	c2 := &Coverage{
		Type: TypeLOC,
		Files: FileCoverages{
			{
				File:           "cmd/root.go",
				NormalizedPath: "cmd/root.go",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
					newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
				},
			},
		},
	}

	if err := c1.Merge(c2); err != nil {
		t.Fatal(err)
	}

	// Should merge into single file entry because NormalizedPath matches
	if len(c1.Files) != 1 {
		t.Errorf("expected 1 file after merge, got %d", len(c1.Files))
		for _, f := range c1.Files {
			t.Logf("  file: %s (normalized: %s)", f.File, f.NormalizedPath)
		}
	}

	if c1.Files[0].Total != 2 {
		t.Errorf("expected Total=2 after merge, got %d", c1.Files[0].Total)
	}
	if c1.Files[0].Covered != 2 {
		t.Errorf("expected Covered=2 after merge, got %d", c1.Files[0].Covered)
	}
}

func TestExclude_WithNormalizedPaths(t *testing.T) {
	cov := &Coverage{
		Type: TypeLOC,
		Files: FileCoverages{
			{
				File:           "github.com/user/repo/cmd/root.go",
				NormalizedPath: "cmd/root.go",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
			{
				File:           "src/utils/groups.ts",
				NormalizedPath: "internal/frontend/src/utils/groups.ts",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
		},
	}

	// Exclude using NormalizedPath pattern
	if err := cov.Exclude([]string{"cmd/**"}); err != nil {
		t.Fatal(err)
	}

	if len(cov.Files) != 1 {
		t.Fatalf("expected 1 file after exclude, got %d", len(cov.Files))
	}

	if cov.Files[0].NormalizedPath != "internal/frontend/src/utils/groups.ts" {
		t.Errorf("wrong file remaining: %s", cov.Files[0].NormalizedPath)
	}
}

func TestExclude_WithNormalizedPaths_Glob(t *testing.T) {
	cov := &Coverage{
		Type: TypeLOC,
		Files: FileCoverages{
			{
				File:           "github.com/user/repo/cmd/root.go",
				NormalizedPath: "cmd/root.go",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
			{
				File:           "src/utils/groups.ts",
				NormalizedPath: "internal/frontend/src/utils/groups.ts",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
			{
				File:           "test.go",
				NormalizedPath: "test.go",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
		},
	}

	// Exclude TypeScript files using glob pattern on NormalizedPath
	if err := cov.Exclude([]string{"**/*.ts"}); err != nil {
		t.Fatal(err)
	}

	if len(cov.Files) != 2 {
		t.Fatalf("expected 2 files after exclude, got %d", len(cov.Files))
	}
}

func TestExclude_FallbackToOriginalFile(t *testing.T) {
	cov := &Coverage{
		Type: TypeLOC,
		Files: FileCoverages{
			{
				File:           "github.com/owner/repo/internal/database/db/query.go",
				NormalizedPath: "internal/database/db/query.go",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
			{
				File:           "github.com/owner/repo/cmd/root.go",
				NormalizedPath: "cmd/root.go",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
			{
				File:           "src/utils/groups.ts",
				NormalizedPath: "internal/frontend/src/utils/groups.ts",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
		},
	}

	tests := []struct {
		name      string
		exclude   []string
		wantFiles []string
	}{
		{
			// Exclude using module path pattern (backward compat)
			name:      "module path glob",
			exclude:   []string{"github.com/owner/repo/internal/database/db/*.go"},
			wantFiles: []string{"cmd/root.go", "internal/frontend/src/utils/groups.ts"},
		},
		{
			// Exclude using normalized path pattern
			name:      "normalized path glob",
			exclude:   []string{"internal/database/db/*.go"},
			wantFiles: []string{"cmd/root.go", "internal/frontend/src/utils/groups.ts"},
		},
		{
			// Exclude using both module path and normalized path
			name:      "both patterns",
			exclude:   []string{"github.com/owner/repo/cmd/*.go", "internal/frontend/**/*.ts"},
			wantFiles: []string{"internal/database/db/query.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Deep copy
			c := &Coverage{
				Type:  cov.Type,
				Files: make(FileCoverages, len(cov.Files)),
			}
			for i, f := range cov.Files {
				fc := *f
				c.Files[i] = &fc
			}
			if err := c.Exclude(tt.exclude); err != nil {
				t.Fatal(err)
			}
			var got []string
			for _, f := range c.Files {
				got = append(got, f.EffectivePath())
			}
			if diff := cmp.Diff(tt.wantFiles, got); diff != "" {
				t.Errorf("files mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestNormalizePaths_DotSlashPrefix(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "src", "utils")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(root, "src", "utils", "helper.go")
	if err := os.WriteFile(f, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	cov := &Coverage{
		Files: FileCoverages{
			{File: "./src/utils/helper.go"},
		},
	}
	cov.NormalizePaths(root, []string{f})

	if got := cov.Files[0].NormalizedPath; got != "src/utils/helper.go" {
		t.Errorf("NormalizedPath = %q, want %q", got, "src/utils/helper.go")
	}
}

func TestCompare_OnlyOneSideNormalized(t *testing.T) {
	// A has NormalizedPath, B does not (e.g., old stored report)
	a := &Coverage{
		Total:   100,
		Covered: 60,
		Files: FileCoverages{
			{File: "github.com/user/repo/cmd/main.go", NormalizedPath: "cmd/main.go", Total: 50, Covered: 30},
			{File: "src/utils.ts", NormalizedPath: "frontend/src/utils.ts", Total: 50, Covered: 30},
		},
	}
	b := &Coverage{
		Total:   100,
		Covered: 50,
		Files: FileCoverages{
			{File: "github.com/user/repo/cmd/main.go", Total: 50, Covered: 25},
			{File: "src/utils.ts", Total: 50, Covered: 25},
		},
	}

	got := a.Compare(b)

	if len(got.Files) != 2 {
		t.Fatalf("expected 2 DiffFileCoverages, got %d", len(got.Files))
	}
	for _, f := range got.Files {
		if f.FileCoverageA == nil {
			t.Errorf("file %q: FileCoverageA is nil", f.File)
		}
		if f.FileCoverageB == nil {
			t.Errorf("file %q: FileCoverageB is nil", f.File)
		}
	}

	// Reverse: B has NormalizedPath, A does not
	got2 := b.Compare(a)
	if len(got2.Files) != 2 {
		t.Fatalf("(reverse) expected 2 DiffFileCoverages, got %d", len(got2.Files))
	}
	for _, f := range got2.Files {
		if f.FileCoverageA == nil {
			t.Errorf("(reverse) file %q: FileCoverageA is nil", f.File)
		}
		if f.FileCoverageB == nil {
			t.Errorf("(reverse) file %q: FileCoverageB is nil", f.File)
		}
	}
}

func TestMerge_OnlyOneSideNormalized(t *testing.T) {
	c1 := &Coverage{
		Type: TypeLOC,
		Files: FileCoverages{
			{
				File:           "github.com/user/repo/cmd/root.go",
				NormalizedPath: "cmd/root.go",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
		},
	}

	// c2 has no NormalizedPath (simulates old report)
	c2 := &Coverage{
		Type: TypeLOC,
		Files: FileCoverages{
			{
				File: "github.com/user/repo/cmd/root.go",
				Type: TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
				},
			},
		},
	}

	if err := c1.Merge(c2); err != nil {
		t.Fatal(err)
	}

	if len(c1.Files) != 1 {
		t.Errorf("expected 1 file after merge, got %d", len(c1.Files))
	}

	// Reverse: c1 has no NormalizedPath, c2 does
	c3 := &Coverage{
		Type: TypeLOC,
		Files: FileCoverages{
			{
				File: "github.com/user/repo/cmd/root.go",
				Type: TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
				},
			},
		},
	}
	c4 := &Coverage{
		Type: TypeLOC,
		Files: FileCoverages{
			{
				File:           "github.com/user/repo/cmd/root.go",
				NormalizedPath: "cmd/root.go",
				Type:           TypeLOC,
				Blocks: BlockCoverages{
					newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
				},
			},
		},
	}

	if err := c3.Merge(c4); err != nil {
		t.Fatal(err)
	}

	if len(c3.Files) != 1 {
		t.Errorf("(reverse) expected 1 file after merge, got %d", len(c3.Files))
	}
}

func TestNormalizePaths_AbsolutePathOutsideRoot(t *testing.T) {
	root := t.TempDir()
	otherDir := t.TempDir()

	f := filepath.Join(root, "main.go")
	if err := os.WriteFile(f, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	// Absolute path outside root should not match
	cov := &Coverage{
		Files: FileCoverages{
			{File: filepath.Join(otherDir, "main.go")},
		},
	}
	cov.NormalizePaths(root, []string{f})

	// Should still try suffix match and find it
	if got := cov.Files[0].NormalizedPath; got != "main.go" {
		t.Errorf("NormalizedPath = %q, want %q", got, "main.go")
	}
}
