package coverage

import "testing"

func TestFileCoveragePatchCoverage(t *testing.T) {
	fc := &FileCoverage{
		File: "main.go",
		Blocks: BlockCoverages{
			&BlockCoverage{StartLine: intPtr(1), EndLine: intPtr(1), Count: execCountPtr(1)},
			&BlockCoverage{StartLine: intPtr(2), EndLine: intPtr(2), Count: execCountPtr(0)},
			&BlockCoverage{StartLine: intPtr(3), EndLine: intPtr(3), Count: execCountPtr(2)},
			&BlockCoverage{StartLine: intPtr(4), EndLine: intPtr(4), Count: execCountPtr(0)},
		},
	}
	got := fc.PatchCoverage([]int{1, 2, 3, 4})
	if got.Covered != 2 {
		t.Errorf("Covered got %v want 2", got.Covered)
	}
	if got.Total != 4 {
		t.Errorf("Total got %v want 4", got.Total)
	}
	if got.Rate() != 50.0 {
		t.Errorf("Rate got %v want 50.0", got.Rate())
	}
}

func TestFileCoveragePatchCoverageExcludesUninstrumentedLines(t *testing.T) {
	// The whole file is added by the pull request, and every statement is executed:
	//  1 package / 2 blank / 3 comment / 4 func A() { / 5-6 stmts / 7 }
	//  8 func B() { / 9-10 stmts / 11 } / 12 blank
	fc := &FileCoverage{
		File: "new.go",
		Blocks: BlockCoverages{
			&BlockCoverage{Type: TypeStmt, StartLine: intPtr(5), EndLine: intPtr(6), Count: execCountPtr(3)},
			&BlockCoverage{Type: TypeStmt, StartLine: intPtr(9), EndLine: intPtr(10), Count: execCountPtr(3)},
		},
	}
	got := fc.PatchCoverage([]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12})
	if got.Total != 4 {
		t.Errorf("Total got %v want 4", got.Total)
	}
	if got.Covered != 4 {
		t.Errorf("Covered got %v want 4", got.Covered)
	}
	if got.Rate() != 100.0 {
		t.Errorf("Rate got %v want 100.0", got.Rate())
	}
}

func TestFileCoveragePatchCoverageNoInstrumentedChangedLines(t *testing.T) {
	fc := &FileCoverage{
		File: "main.go",
		Blocks: BlockCoverages{
			&BlockCoverage{Type: TypeStmt, StartLine: intPtr(10), EndLine: intPtr(11), Count: execCountPtr(1)},
		},
	}
	got := fc.PatchCoverage([]int{1, 2, 3})
	if got.Total != 0 || got.Covered != 0 || got.Rate() != 0 {
		t.Errorf("got %+v, want all zero", got)
	}
}

func TestFileCoveragePatchCoverageNoChangedLines(t *testing.T) {
	fc := &FileCoverage{File: "main.go"}
	got := fc.PatchCoverage(nil)
	if got.Total != 0 || got.Covered != 0 || got.Rate() != 0 {
		t.Errorf("got %+v, want all zero", got)
	}
}

func TestCoveragePatchCoverage(t *testing.T) {
	c := &Coverage{
		Files: FileCoverages{
			&FileCoverage{
				File: "a.go",
				Blocks: BlockCoverages{
					&BlockCoverage{StartLine: intPtr(1), EndLine: intPtr(1), Count: execCountPtr(1)},
					&BlockCoverage{StartLine: intPtr(2), EndLine: intPtr(2), Count: execCountPtr(0)},
				},
			},
			&FileCoverage{
				File: "b.go",
				Blocks: BlockCoverages{
					&BlockCoverage{StartLine: intPtr(1), EndLine: intPtr(1), Count: execCountPtr(0)},
					&BlockCoverage{StartLine: intPtr(2), EndLine: intPtr(2), Count: execCountPtr(0)},
				},
			},
		},
	}
	got := c.PatchCoverage(map[string][]int{
		"a.go": {1, 2},
		"b.go": {1, 2},
		"c.go": {1}, // not present in Coverage.Files, should be skipped
	})
	if got.Total != 4 {
		t.Errorf("Total got %v want 4", got.Total)
	}
	if got.Covered != 1 {
		t.Errorf("Covered got %v want 1", got.Covered)
	}
	if len(got.Files) != 2 {
		t.Errorf("len(Files) got %v want 2", len(got.Files))
	}
	if got.Files[0].File != "a.go" || got.Files[1].File != "b.go" {
		t.Errorf("Files got %+v, want sorted [a.go, b.go]", got.Files)
	}
}

func intPtr(i int) *int {
	return &i
}

func execCountPtr(c ExecCount) *ExecCount {
	return &c
}

func TestPatchCoverageAfterDeleteBlockCoverages(t *testing.T) {
	// Storing a report shrinks away the block coverages, and patch coverage is derived from
	// them. Once shrunk, patch coverage reads as unmeasurable (Total 0) rather than being
	// answered from the line cache built before the shrink.
	c := &Coverage{
		Files: FileCoverages{
			&FileCoverage{
				File: "a.go",
				Blocks: BlockCoverages{
					&BlockCoverage{StartLine: intPtr(1), EndLine: intPtr(2), Count: execCountPtr(1)},
				},
			},
		},
	}
	changedFiles := map[string][]int{"a.go": {1, 2}}

	if got := c.PatchCoverage(changedFiles); got.Total != 2 {
		t.Fatalf("Total before shrink got %v want 2", got.Total)
	}

	c.DeleteBlockCoverages()

	if got := c.PatchCoverage(changedFiles); got.Total != 0 {
		t.Errorf("Total after shrink got %v want 0", got.Total)
	}
}

func TestPatchCoverageWithBlockMissingLineRange(t *testing.T) {
	// A report unmarshaled from a datastore can carry a block whose line range was omitted.
	// Such a block instruments no line, and must not take the whole measurement down with it.
	fc := &FileCoverage{
		File: "a.go",
		Blocks: BlockCoverages{
			&BlockCoverage{Count: execCountPtr(1)},
			&BlockCoverage{StartLine: intPtr(3), EndLine: intPtr(3), Count: execCountPtr(1)},
		},
	}
	got := fc.PatchCoverage([]int{1, 2, 3})
	if got.Total != 1 {
		t.Errorf("Total got %v want 1", got.Total)
	}
	if got.Covered != 1 {
		t.Errorf("Covered got %v want 1", got.Covered)
	}
}
