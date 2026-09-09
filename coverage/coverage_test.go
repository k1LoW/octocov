package coverage

import (
	"math"
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
				Diff: 54.0,
				Files: DiffFileCoverages{
					&DiffFileCoverage{File: "file_a.go", A: 65.0, B: 0.0, Diff: 65.0},
					&DiffFileCoverage{File: "file_b.go", A: 37.5, B: 0.0, Diff: 37.5},
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
				Diff: -41.0,
				Files: DiffFileCoverages{
					&DiffFileCoverage{File: "file_a.go", A: 65.0, B: 98.33333333333333, Diff: -33.33333333333333},
					&DiffFileCoverage{File: "file_b.go", A: 37.5, B: 87.5, Diff: -50.0},
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
			t.Error(diff)
		}
	}
}

func TestMaxCount(t *testing.T) {
	tests := []struct {
		blocks BlockCoverages
		want   ExecCount
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
			t.Error(diff)
		}

		if got.Total() != tt.wantTotal {
			t.Errorf("got %v\nwant %v", got.Total(), tt.wantTotal)
		}

		if got.Covered() != tt.wantCovered {
			t.Errorf("got %v\nwant %v", got.Covered(), tt.wantCovered)
		}
	}
}

func TestFuzzyFindByFile(t *testing.T) {
	tests := []struct {
		coverageFiles []string
		file          string
		want          string
		wantErr       bool
	}{
		{
			[]string{
				"/path/to/owner/repo/other.go",
				"/path/to/owner/repo/target.go",
			},
			"./owner/repo/target.go",
			"/path/to/owner/repo/target.go",
			false,
		},
		{[]string{"org/101000lab/owner/repo/target.go"}, "./owner/repo/target.go", "org/101000lab/owner/repo/target.go", false},
		{[]string{"org/101000lab/owner/repo/target.go"}, "path/to/src/org/101000lab/owner/repo/target.go", "org/101000lab/owner/repo/target.go", false},
		{
			[]string{
				"/path/to/owner/repo/target.go",
				"/path/to/owner/repo/a/target.go",
			},
			"target.go",
			"/path/to/owner/repo/target.go",
			false,
		},
		{
			[]string{
				"/path/to/owner/repo/target.go",
				"/path/to/owner/repo/a/target.go",
			},
			"a/target.go",
			"/path/to/owner/repo/a/target.go",
			false,
		},
		{
			[]string{
				"/path/to/owner/repo/a/target.go",
				"/path/to/owner/repo/target.go",
			},
			"target.go",
			"/path/to/owner/repo/target.go",
			false,
		},
		// The whole path outranks any suffix of it.
		{[]string{"a/target.go", "src/a/target.go"}, "src/a/target.go", "src/a/target.go", false},
		// A package path outranks a bare filename for a lookup that contains both, since it
		// shares more of the path, whichever order the report lists them in. A JaCoCo default
		// package produces the bare filename.
		{[]string{"Foo.java", "com/example/Foo.java"}, "src/main/java/com/example/Foo.java", "com/example/Foo.java", false},
		{[]string{"com/example/Foo.java", "Foo.java"}, "src/main/java/com/example/Foo.java", "com/example/Foo.java", false},
		{[]string{"Foo.java", "com/example/Foo.java"}, "src/main/java/Foo.java", "Foo.java", false},
		// A match has to end at a path segment, in either direction.
		{[]string{"Foo.java"}, "src/main/java/MyFoo.java", "", true},
		{[]string{"/path/to/repo/domain.go"}, "main.go", "", true},
		// An entry with no name is a suffix of nothing rather than of everything.
		{[]string{"", "src/a/target.go"}, "src/a/target.go", "src/a/target.go", false},
		{[]string{""}, "src/a/target.go", "", true},
	}
	for _, tt := range tests {
		fcs := FileCoverages{}
		for _, f := range tt.coverageFiles {
			fcs = append(fcs, &FileCoverage{File: f})
		}
		fc, err := fcs.FuzzyFindByFile(tt.file)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("got err: %v", err)
			}
			continue
		}
		if tt.wantErr {
			t.Error("want error")
		}
		got := fc.File
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}

		dfcs := DiffFileCoverages{}
		for _, f := range tt.coverageFiles {
			dfcs = append(dfcs, &DiffFileCoverage{File: f})
		}
		if _, err := dfcs.FuzzyFindByFile(tt.file); err != nil {
			if !tt.wantErr {
				t.Errorf("got err: %v", err)
			}
			continue
		}
		if tt.wantErr {
			t.Error("want error")
		}
	}
}

func TestDiffFileCoveragesFuzzyFindByFile(t *testing.T) {
	tests := []struct {
		coverageFiles []string
		file          string
		want          string
		wantErr       bool
	}{
		{
			[]string{
				"/path/to/owner/repo/other.go",
				"/path/to/owner/repo/target.go",
			},
			"./owner/repo/target.go",
			"/path/to/owner/repo/target.go",
			false,
		},
		{[]string{"org/101000lab/owner/repo/target.go"}, "./owner/repo/target.go", "org/101000lab/owner/repo/target.go", false},
		{[]string{"org/101000lab/owner/repo/target.go"}, "path/to/src/org/101000lab/owner/repo/target.go", "org/101000lab/owner/repo/target.go", false},
		{
			[]string{
				"/path/to/owner/repo/target.go",
				"/path/to/owner/repo/a/target.go",
			},
			"target.go",
			"/path/to/owner/repo/target.go",
			false,
		},
		{
			[]string{
				"/path/to/owner/repo/target.go",
				"/path/to/owner/repo/a/target.go",
			},
			"a/target.go",
			"/path/to/owner/repo/a/target.go",
			false,
		},
		{
			[]string{
				"/path/to/owner/repo/a/target.go",
				"/path/to/owner/repo/target.go",
			},
			"target.go",
			"/path/to/owner/repo/target.go",
			false,
		},
	}
	for _, tt := range tests {
		dfcs := DiffFileCoverages{}
		for _, f := range tt.coverageFiles {
			dfcs = append(dfcs, &DiffFileCoverage{File: f})
		}
		dfc, err := dfcs.FuzzyFindByFile(tt.file)
		if err != nil {
			if !tt.wantErr {
				t.Errorf("got err: %v", err)
			}
			continue
		}
		if tt.wantErr {
			t.Error("want error")
		}
		got := dfc.File
		if got != tt.want {
			t.Errorf("got %v\nwant %v", got, tt.want)
		}
	}
}

func newBlockCoverage(t Type, sl, sc, el, ec, ns, c int) *BlockCoverage {
	cc := toExecCount(c)
	bc := &BlockCoverage{
		Type:      t,
		StartLine: &sl,
		EndLine:   &el,
		Count:     &cc,
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

func TestLineRangeWalksTerminateAtMaxInt(t *testing.T) {
	// A line or column number of math.MaxInt used to wrap the loop counter to math.MinInt, so
	// the walk never ended and ToLineCoverages grew a nested map per key while it spun. Every
	// walk must instead treat such a block as the single line or column it describes, and a
	// block whose start is above its end must still yield nothing.
	t.Run("ToLineCoverages", func(t *testing.T) {
		blocks := BlockCoverages{
			&BlockCoverage{Type: TypeLOC, StartLine: new(math.MaxInt), EndLine: new(math.MaxInt), Count: new(ExecCount(1))},
		}
		lcs := blocks.ToLineCoverages()
		if want := 1; lcs.Total() != want {
			t.Errorf("got %v\nwant %v", lcs.Total(), want)
		}
		if want := 1; lcs.Covered() != want {
			t.Errorf("got %v\nwant %v", lcs.Covered(), want)
		}
		if want := math.MaxInt; lcs[0].Line != want {
			t.Errorf("got %v\nwant %v", lcs[0].Line, want)
		}
	})

	t.Run("MaxCount", func(t *testing.T) {
		blocks := BlockCoverages{
			&BlockCoverage{Type: TypeLOC, StartLine: new(math.MaxInt), EndLine: new(math.MaxInt), Count: new(ExecCount(7))},
		}
		if want := ExecCount(7); blocks.MaxCount() != want {
			t.Errorf("got %v\nwant %v", blocks.MaxCount(), want)
		}
	})

	t.Run("FindBlocksByLine", func(t *testing.T) {
		fc := &FileCoverage{
			File: "a.kt",
			Blocks: BlockCoverages{
				&BlockCoverage{Type: TypeLOC, StartLine: new(math.MaxInt), EndLine: new(math.MaxInt), Count: new(ExecCount(1))},
			},
		}
		if want := 1; len(fc.FindBlocksByLine(math.MaxInt)) != want {
			t.Errorf("got %v\nwant %v", len(fc.FindBlocksByLine(math.MaxInt)), want)
		}
	})

	t.Run("column walk", func(t *testing.T) {
		// The TypeStmt path walks StartCol to EndCol for a block confined to one line, which
		// is the fourth copy of the same shape and is not reachable through a line number.
		blocks := BlockCoverages{
			&BlockCoverage{
				Type:      TypeStmt,
				StartLine: new(1), EndLine: new(1),
				StartCol: new(math.MaxInt), EndCol: new(math.MaxInt),
				Count: new(ExecCount(1)),
			},
		}
		lcs := blocks.ToLineCoverages()
		if want := 1; lcs.Total() != want {
			t.Errorf("got %v\nwant %v", lcs.Total(), want)
		}
	})

	t.Run("start above end yields nothing", func(t *testing.T) {
		blocks := BlockCoverages{
			&BlockCoverage{Type: TypeLOC, StartLine: new(5), EndLine: new(3), Count: new(ExecCount(1))},
		}
		if want := 0; blocks.ToLineCoverages().Total() != want {
			t.Errorf("got %v\nwant %v", blocks.ToLineCoverages().Total(), want)
		}
		if want := ExecCount(0); blocks.MaxCount() != want {
			t.Errorf("got %v\nwant %v", blocks.MaxCount(), want)
		}
	})
}
