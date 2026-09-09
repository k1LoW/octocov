package coverage

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMerge(t *testing.T) {
	tests := []struct {
		name string
		c1   *Coverage
		c2   *Coverage
		want *Coverage
	}{
		{
			"Simple",
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
			"Merge file_b.go (LOC)",
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
			"Merge file_a.go (LOC)",
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
			"Merge file_a.go (LOC and Stmt)",
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
						File: "file_a.go",
						Type: TypeStmt,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 0, 1, 10, 1, 1),
							newBlockCoverage(TypeStmt, 2, 0, 2, 10, 1, 0),
							newBlockCoverage(TypeStmt, 3, 0, 3, 10, 1, 1),
						},
					},
				},
			},
			&Coverage{
				Type:    TypeMerged,
				Total:   3,
				Covered: 2,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeMerged,
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
							newBlockCoverage(TypeLOC, 2, -1, 2, -1, -1, 0),
							newBlockCoverage(TypeLOC, 3, -1, 3, -1, -1, 1),
							newBlockCoverage(TypeStmt, 1, 0, 1, 10, 1, 1),
							newBlockCoverage(TypeStmt, 2, 0, 2, 10, 1, 0),
							newBlockCoverage(TypeStmt, 3, 0, 3, 10, 1, 1),
						},
					},
				},
			},
		},
		{
			"Merge file_a.go and file_b.go (LOC)",
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
			"Merge no covered file (file_c.go)",
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
			"Simple (stmt)",
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
				Total:   23,
				Covered: 20,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeStmt,
						Total:   10,
						Covered: 9,
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
			"Merge LOC and Stmt",
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
						Type:    TypeStmt,
						Total:   3,
						Covered: 2,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeStmt, 1, 0, 1, 10, 1, 1),
							newBlockCoverage(TypeStmt, 2, 0, 2, 10, 1, 0),
							newBlockCoverage(TypeStmt, 3, 0, 3, 10, 2, 2),
						},
					},
				},
			},
		},
		{
			"Merge same format",
			&Coverage{
				Type:   TypeLOC,
				Format: "LCOV",
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type:   TypeLOC,
				Format: "LCOV",
				Files: FileCoverages{
					&FileCoverage{
						File: "file_b.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type:    TypeLOC,
				Format:  "LCOV",
				Total:   2,
				Covered: 2,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
						Total:   1,
						Covered: 1,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
					&FileCoverage{
						File:    "file_b.go",
						Type:    TypeLOC,
						Total:   1,
						Covered: 1,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
		},
		{
			"Merge different formats",
			&Coverage{
				Type:   TypeLOC,
				Format: "Go coverage",
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type:   TypeLOC,
				Format: "LCOV",
				Files: FileCoverages{
					&FileCoverage{
						File: "file_b.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type:    TypeLOC,
				Format:  FormatMerged,
				Total:   2,
				Covered: 2,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
						Total:   1,
						Covered: 1,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
					&FileCoverage{
						File:    "file_b.go",
						Type:    TypeLOC,
						Total:   1,
						Covered: 1,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
		},
		{
			"Merge with empty format",
			&Coverage{
				Type: TypeLOC,
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type:   TypeLOC,
				Format: "LCOV",
				Files: FileCoverages{
					&FileCoverage{
						File: "file_b.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
			&Coverage{
				Type:    TypeLOC,
				Format:  "LCOV",
				Total:   2,
				Covered: 2,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
						Total:   1,
						Covered: 1,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
					&FileCoverage{
						File:    "file_b.go",
						Type:    TypeLOC,
						Total:   1,
						Covered: 1,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
		},
		{
			"Merge nil preserves format",
			&Coverage{
				Type:   TypeLOC,
				Format: "Go coverage",
				Files: FileCoverages{
					&FileCoverage{
						File: "file_a.go",
						Type: TypeLOC,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
			nil,
			&Coverage{
				Type:    TypeLOC,
				Format:  "Go coverage",
				Total:   1,
				Covered: 1,
				Files: FileCoverages{
					&FileCoverage{
						File:    "file_a.go",
						Type:    TypeLOC,
						Total:   1,
						Covered: 1,
						Blocks: BlockCoverages{
							newBlockCoverage(TypeLOC, 1, -1, 1, -1, -1, 1),
						},
					},
				},
			},
		},
	}
	opts := []cmp.Option{
		cmpopts.IgnoreUnexported(FileCoverage{}),
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.c1.Merge(tt.c2); err != nil {
				t.Fatal(err)
			}
			got := tt.c1
			if diff := cmp.Diff(got, tt.want, opts...); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func TestReCalcSkipsStatementBlockWithoutCountOrNumStmt(t *testing.T) {
	// A statement block's count and statement number are both omitempty, so a stored report
	// can come back without either. Recalculating from such a block must count nothing for it
	// instead of dereferencing nil, and the complete block beside it must still be counted.
	complete := newBlockCoverage(TypeStmt, 3, 1, 3, 5, 2, 1)
	tests := []struct {
		name  string
		block *BlockCoverage
	}{
		{"no count", &BlockCoverage{Type: TypeStmt, StartLine: new(1), StartCol: new(1), EndLine: new(1), EndCol: new(5), NumStmt: new(1)}},
		{"no num_stmt", &BlockCoverage{Type: TypeStmt, StartLine: new(1), StartCol: new(1), EndLine: new(1), EndCol: new(5), Count: new(ExecCount(1))}},
		{"neither", &BlockCoverage{Type: TypeStmt}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Coverage{
				Type: TypeStmt,
				Files: FileCoverages{
					&FileCoverage{File: "a.go", Type: TypeStmt, Blocks: BlockCoverages{tt.block, complete}},
				},
			}
			if err := c.Exclude(nil); err != nil {
				t.Fatal(err)
			}
			if want := 2; c.Total != want {
				t.Errorf("got %v\nwant %v", c.Total, want)
			}
			if want := 2; c.Covered != want {
				t.Errorf("got %v\nwant %v", c.Covered, want)
			}
		})
	}
}
