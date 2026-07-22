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
