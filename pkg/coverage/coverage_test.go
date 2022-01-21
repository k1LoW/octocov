package coverage

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestCompare(t *testing.T) {
	a := &Coverage{
		Total:   100,
		Covered: 54,
		Files: FileCoverages{
			&FileCoverage{File: "file_a.go", Total: 60, Covered: 39},
			&FileCoverage{File: "file_b.go", Total: 40, Covered: 15},
		},
	}

	tests := []struct {
		b    *Coverage
		want *DiffCoverage
	}{
		{
			&Coverage{
				Total:   100,
				Covered: 54,
				Files: FileCoverages{
					&FileCoverage{File: "file_a.go", Total: 60, Covered: 39},
					&FileCoverage{File: "file_b.go", Total: 40, Covered: 15},
				},
			},
			&DiffCoverage{
				A:    54.0,
				B:    54.0,
				Diff: 0.0,
				Files: DiffFileCoverages{
					&DiffFileCoverage{File: "file_a.go", A: 65.0, B: 65.0, Diff: 0.0},
					&DiffFileCoverage{File: "file_b.go", A: 37.5, B: 37.5, Diff: 0.0},
				},
			},
		},
		{
			nil,
			&DiffCoverage{
				A:    54.0,
				B:    0.0,
				Diff: -54.0,
				Files: DiffFileCoverages{
					&DiffFileCoverage{File: "file_a.go", A: 65.0, B: 0.0, Diff: -65.0},
					&DiffFileCoverage{File: "file_b.go", A: 37.5, B: 0.0, Diff: -37.5},
				},
			},
		},
		{
			&Coverage{
				Total:   100,
				Covered: 95,
				Files: FileCoverages{
					&FileCoverage{File: "file_a.go", Total: 60, Covered: 59},
					&FileCoverage{File: "file_b.go", Total: 40, Covered: 35},
				},
			},
			&DiffCoverage{
				A:    54.0,
				B:    95.0,
				Diff: 41.0,
				Files: DiffFileCoverages{
					&DiffFileCoverage{File: "file_a.go", A: 65.0, B: 98.33333333333333, Diff: 33.33333333333333},
					&DiffFileCoverage{File: "file_b.go", A: 37.5, B: 87.5, Diff: 50.0},
				},
			},
		},
	}
	for _, tt := range tests {
		got := a.Compare(tt.b)

		opts := []cmp.Option{
			cmpopts.IgnoreUnexported(DiffCoverage{}),
			cmpopts.IgnoreFields(DiffCoverage{}, "CoverageA", "CoverageB"),
			cmpopts.SortSlices(func(i, j *DiffFileCoverage) bool {
				return i.File < j.File
			}),
			cmpopts.IgnoreFields(DiffFileCoverage{}, "FileCoverageA", "FileCoverageB"),
		}

		if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
			t.Errorf("%s", diff)
		}
	}
}

func TestPathPrefix(t *testing.T) {
	tests := []struct {
		files FileCoverages
		want  string
	}{
		{
			FileCoverages{
				&FileCoverage{File: "file_a.go"},
			},
			"",
		},
		{
			FileCoverages{
				&FileCoverage{File: "path/to/file_a.go"},
				&FileCoverage{File: "path/file_b.go"},
			},
			"path",
		},
		{
			FileCoverages{
				&FileCoverage{File: "/path/to/file_a.go"},
				&FileCoverage{File: "/path/file_b.go"},
			},
			"/path",
		},
		{
			FileCoverages{
				&FileCoverage{File: "/path/to/foo/file_a.go"},
				&FileCoverage{File: "/path/to/foo/bar/file_b.go"},
			},
			"/path/to/foo",
		},
		{
			FileCoverages{
				&FileCoverage{File: "/to/foo/file_a.go"},
				&FileCoverage{File: "/path/to/foo/bar/file_b.go"},
			},
			"/",
		},
	}
	for _, tt := range tests {
		got, err := tt.files.PathPrefix()
		if err != nil {
			t.Fatal(err)
		}
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestMaxCount(t *testing.T) {
	tests := []struct {
		blocks BlockCoverages
		want   int
	}{
		{
			BlockCoverages{
				newBlockCoverage(TypeLOC, 6, -1, 6, -1, -1, 10),
				newBlockCoverage(TypeLOC, 7, -1, 7, -1, -1, 100),
				newBlockCoverage(TypeLOC, 8, -1, 8, -1, -1, 11),
				newBlockCoverage(TypeLOC, 9, -1, 9, -1, -1, 1),
			},
			100,
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeStmt, 6, 1, 7, 10, 1, 10),
				newBlockCoverage(TypeStmt, 7, 1, 7, 10, 1, 100),
				newBlockCoverage(TypeStmt, 7, 1, 8, 10, 1, 11),
				newBlockCoverage(TypeStmt, 9, 1, 9, 10, 1, 1),
			},
			121,
		},
	}
	for _, tt := range tests {
		got := tt.blocks.MaxCount()
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func TestToLineCoverages(t *testing.T) {
	tests := []struct {
		blocks      BlockCoverages
		want        LineCoverages
		wantTotal   int
		wantCovered int
	}{
		{
			BlockCoverages{
				newBlockCoverage(TypeLOC, 6, -1, 6, -1, -1, 10),
			},
			LineCoverages{
				&LineCoverage{Line: 6, Count: 10, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 10}, &PosCoverage{Pos: endPos, Count: 10}}},
			},
			1,
			1,
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeLOC, 6, -1, 6, -1, -1, 10),
				newBlockCoverage(TypeLOC, 8, -1, 8, -1, -1, 11),
			},
			LineCoverages{
				&LineCoverage{Line: 6, Count: 10, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 10}, &PosCoverage{Pos: endPos, Count: 10}}},
				&LineCoverage{Line: 8, Count: 11, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 11}, &PosCoverage{Pos: endPos, Count: 11}}},
			},
			2,
			2,
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeLOC, 6, -1, 6, -1, -1, 3),
				newBlockCoverage(TypeLOC, 6, -1, 6, -1, -1, 7),
			},
			LineCoverages{
				&LineCoverage{Line: 6, Count: 10, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 10}, &PosCoverage{Pos: endPos, Count: 10}}},
			},
			1,
			1,
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeStmt, 6, 0, 8, 10, 1, 7),
			},
			LineCoverages{
				&LineCoverage{Line: 6, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: 0, Count: 7}, &PosCoverage{Pos: endPos, Count: 7}}},
				&LineCoverage{Line: 7, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 7}, &PosCoverage{Pos: endPos, Count: 7}}},
				&LineCoverage{Line: 8, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 7}, &PosCoverage{Pos: 10, Count: 7}}},
			},
			3,
			3,
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeStmt, 6, 0, 6, 3, 1, 7),
			},
			LineCoverages{
				&LineCoverage{Line: 6, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: 0, Count: 7}, &PosCoverage{Pos: 1, Count: 7}, &PosCoverage{Pos: 2, Count: 7}, &PosCoverage{Pos: 3, Count: 7}}},
			},
			1,
			1,
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeStmt, 6, 0, 8, 10, 1, 7),
				newBlockCoverage(TypeStmt, 6, 0, 6, 3, 1, 7),
			},
			LineCoverages{
				&LineCoverage{Line: 6, Count: 14, PosCoverages: PosCoverages{&PosCoverage{Pos: 0, Count: 14}, &PosCoverage{Pos: 1, Count: 14}, &PosCoverage{Pos: 2, Count: 14}, &PosCoverage{Pos: 3, Count: 14}, &PosCoverage{Pos: endPos, Count: 7}}},
				&LineCoverage{Line: 7, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 7}, &PosCoverage{Pos: endPos, Count: 7}}},
				&LineCoverage{Line: 8, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 7}, &PosCoverage{Pos: 10, Count: 7}}},
			},
			3,
			3,
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeStmt, 6, 1, 7, 1, 1, 7),
				newBlockCoverage(TypeStmt, 7, 3, 7, 3, 1, 7),
				newBlockCoverage(TypeStmt, 7, 5, 8, 3, 1, 7),
			},
			LineCoverages{
				&LineCoverage{Line: 6, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: 1, Count: 7}, &PosCoverage{Pos: endPos, Count: 7}}},
				&LineCoverage{Line: 7, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 7}, &PosCoverage{Pos: 1, Count: 7}, &PosCoverage{Pos: 3, Count: 7}, &PosCoverage{Pos: 5, Count: 7}, &PosCoverage{Pos: endPos, Count: 7}}},
				&LineCoverage{Line: 8, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 7}, &PosCoverage{Pos: 3, Count: 7}}},
			},
			3,
			3,
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeStmt, 6, 1, 6, 3, 1, 7),
				newBlockCoverage(TypeStmt, 6, 3, 7, 1, 1, 7),
				newBlockCoverage(TypeStmt, 6, 3, 6, 5, 1, 7),
			},
			LineCoverages{
				&LineCoverage{Line: 6, Count: 21, PosCoverages: PosCoverages{&PosCoverage{Pos: 1, Count: 7}, &PosCoverage{Pos: 2, Count: 7}, &PosCoverage{Pos: 3, Count: 21}, &PosCoverage{Pos: 4, Count: 14}, &PosCoverage{Pos: 5, Count: 14}, &PosCoverage{Pos: endPos, Count: 7}}},
				&LineCoverage{Line: 7, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 7}, &PosCoverage{Pos: 1, Count: 7}}},
			},
			2,
			2,
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeStmt, 6, 1, 7, 1, 1, 7),
				newBlockCoverage(TypeStmt, 7, 3, 7, 3, 1, 7),
				newBlockCoverage(TypeStmt, 7, 5, 8, 3, 1, 0),
			},
			LineCoverages{
				&LineCoverage{Line: 6, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: 1, Count: 7}, &PosCoverage{Pos: endPos, Count: 7}}},
				&LineCoverage{Line: 7, Count: 7, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 7}, &PosCoverage{Pos: 1, Count: 7}, &PosCoverage{Pos: 3, Count: 7}, &PosCoverage{Pos: 5, Count: 0}, &PosCoverage{Pos: endPos, Count: 0}}},
				&LineCoverage{Line: 8, Count: 0, PosCoverages: PosCoverages{&PosCoverage{Pos: startPos, Count: 0}, &PosCoverage{Pos: 3, Count: 0}}},
			},
			3,
			2,
		},
	}

	for _, tt := range tests {
		got := tt.blocks.ToLineCoverages()
		if diff := cmp.Diff(got, tt.want, nil); diff != "" {
			t.Errorf("%s", diff)
		}

		if got.Total() != tt.wantTotal {
			t.Errorf("got %v\nwant %v", got.Total(), tt.wantTotal)
		}

		if got.Covered() != tt.wantCovered {
			t.Errorf("got %v\nwant %v", got.Covered(), tt.wantCovered)
		}
	}
}

func newBlockCoverage(t Type, sl, sc, el, ec, ns, c int) *BlockCoverage {
	bc := &BlockCoverage{
		Type:      t,
		StartLine: &sl,
		EndLine:   &el,
		Count:     &c,
	}
	if sc >= 0 {
		bc.StartCol = &sc
	}
	if ec >= 0 {
		bc.EndCol = &ec
	}
	if ns >= 0 {
		bc.NumStmt = &ns
	}

	return bc
}
