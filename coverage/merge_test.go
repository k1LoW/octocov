package coverage

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		c1   *Coverage
		c2   *Coverage
		want *Coverage
	}{
		{
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File: "file_b.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_c.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 0),
						},
					},
				},
			},
			&Coverage{
				Type:    TypeLOC,
				Total:   9,
				Covered: 6,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
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
						Type:    TypeLOC,
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File:    "file_c.go",
						Type:    TypeLOC,
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 0),
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
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File: "file_b.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_b.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 0),
						},
					},
				},
			},
			&Coverage{
				Type:    TypeLOC,
				Total:   6,
				Covered: 5,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
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
						Type:    TypeLOC,
						Total:   3,
						Covered: 3,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 0),
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
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type:    TypeLOC,
				Total:   3,
				Covered: 2,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
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
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File: "file_b.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			nil,
			&Coverage{
				Type:    TypeLOC,
				Total:   6,
				Covered: 4,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
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
						Type:    TypeLOC,
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
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File: "file_b.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_c.go",
						Type:    TypeLOC,
						Total:   0,
						Covered: 0,
					},
				},
			},
			&Coverage{
				Type:    TypeLOC,
				Total:   6,
				Covered: 4,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
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
						Type:    TypeLOC,
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 0),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 1),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
					&FileCoverage{
						File:    "file_c.go",
						Type:    TypeLOC,
						Total:   0,
						Covered: 0,
					},
				},
			},
		},
		{
			&Coverage{
				Type: TypeStmt,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Type: TypeStmt,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 1, 1, 1, 1, 1),
							newBlockCoverage(TypeStmt, 2, 1, 2, 1, 1, 0),
							newBlockCoverage(TypeStmt, 3, 1, 10, 10, 1, 1),
						},
					},
					&FileCoverage{
						File: "file_b.go",
						Type: TypeStmt,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 1, 1, 1, 1, 0),
							newBlockCoverage(TypeStmt, 2, 1, 5, 1, 4, 1),
							newBlockCoverage(TypeStmt, 3, 1, 10, 1, 5, 1),
						},
					},
				},
			},
			&Coverage{
				Type: TypeStmt,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_c.go",
						Type: TypeStmt,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 1, 1, 1, 1, 1),
							newBlockCoverage(TypeStmt, 2, 1, 2, 1, 1, 1),
							newBlockCoverage(TypeStmt, 3, 1, 3, 1, 1, 0),
						},
					},
				},
			},
			&Coverage{
				Type:    TypeMerged,
				Total:   16,
				Covered: 13,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeStmt,
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 1, 1, 1, 1, 1),
							newBlockCoverage(TypeStmt, 2, 1, 2, 1, 1, 0),
							newBlockCoverage(TypeStmt, 3, 1, 10, 10, 1, 1),
						},
					},
					&FileCoverage{
						File:    "file_b.go",
						Type:    TypeStmt,
						Total:   10,
						Covered: 9,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 1, 1, 1, 1, 0),
							newBlockCoverage(TypeStmt, 2, 1, 5, 1, 4, 1),
							newBlockCoverage(TypeStmt, 3, 1, 10, 1, 5, 1),
						},
					},
					&FileCoverage{
						File:    "file_c.go",
						Type:    TypeStmt,
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 1, 1, 1, 1, 1),
							newBlockCoverage(TypeStmt, 2, 1, 2, 1, 1, 1),
							newBlockCoverage(TypeStmt, 3, 1, 3, 1, 1, 0),
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
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type: TypeStmt,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_b.go",
						Type: TypeStmt,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 0, 1, 10, 1, 1),
							newBlockCoverage(TypeStmt, 2, 0, 2, 10, 1, 0),
							newBlockCoverage(TypeStmt, 3, 0, 3, 10, 2, 2),
						},
					},
				},
			},
			&Coverage{
				Type:    TypeMerged,
				Total:   7,
				Covered: 5,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
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
						Type:    TypeStmt,
						Total:   4,
						Covered: 3,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 0, 1, 10, 1, 1),
							newBlockCoverage(TypeStmt, 2, 0, 2, 10, 1, 0),
							newBlockCoverage(TypeStmt, 3, 0, 3, 10, 2, 2),
						},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		if err := tt.c1.Merge(tt.c2); err != nil {
			t.Fatal(err)
		}
		got := tt.c1

		opts := []cmp.Option{
			cmpopts.IgnoreUnexported(FileCoverage{}),
		}

		if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
			t.Error(diff)
		}
	}
}
