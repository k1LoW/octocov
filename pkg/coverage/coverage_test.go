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
			cmpopts.IgnoreFields(DiffFileCoverage{}, "FileCoverageA", "FileCoverageB"),
			cmpopts.SortSlices(func(i, j DiffFileCoverage) bool {
				return i.File < j.File
			}),
		}

		if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
			t.Errorf("%s", diff)
		}
	}
}
