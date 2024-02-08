package coverage

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestExclude(t *testing.T) {
	tests := []struct {
		c       *Coverage
		exclude []string
		want    *Coverage
	}{
		{
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File: "file_b.go",
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			[]string{},
			&Coverage{
				Type:    TypeLOC,
				Total:   6,
				Covered: 4,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File:    "file_b.go",
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
		},
		{
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File: "file_b.go",
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			[]string{
				"file_a.go",
			},
			&Coverage{
				Type:    TypeLOC,
				Total:   3,
				Covered: 2,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_b.go",
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
		},
		{
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File: "file_b.go",
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			[]string{
				"file_*.go",
			},
			&Coverage{
				Type:    TypeLOC,
				Total:   0,
				Covered: 0,
				Files:   nil,
			},
		},
		{
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File: "file_b.go",
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			[]string{
				"file_*.go",
				"!**/*.go",
			},
			&Coverage{
				Type:    TypeLOC,
				Total:   6,
				Covered: 4,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File:    "file_b.go",
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		if err := tt.c.Exclude(tt.exclude); err != nil {
			t.Fatal(err)
		}
		got := tt.c

		opts := []cmp.Option{
			cmpopts.IgnoreUnexported(FileCoverage{}),
		}
		if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
			t.Error(diff)
		}
	}
}
