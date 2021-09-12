package coverage

import (
	"bytes"
	"strings"
	"testing"
)

func TestPrint(t *testing.T) {
	code := `package coverage

import "fmt"

func IsOK(in string) error {
	if in != "ok" {
		return fmt.Errorf("error: %s", in)
	}
	return nil
}
`

	tests := []struct {
		blocks BlockCoverages
		want3  string
		want7  string
		want9  string
	}{
		{
			BlockCoverages{
				newBlockCoverage(TypeLOC, 6, -1, 6, -1, -1, 0),
				newBlockCoverage(TypeLOC, 7, -1, 7, -1, -1, 0),
				newBlockCoverage(TypeLOC, 8, -1, 8, -1, -1, 0),
				newBlockCoverage(TypeLOC, 9, -1, 9, -1, -1, 1),
			},
			"\x1b[33m 3\x1b[0m|  | import \"fmt\"",
			"\x1b[33m 7\x1b[0m|  | \x1b[31m\t\treturn fmt.Errorf(\"error: %s\", in)\x1b[0m",
			"\x1b[33m 9\x1b[0m| \x1b[92m1\x1b[0m| \x1b[32m\treturn nil\x1b[0m",
		},
		{
			BlockCoverages{
				newBlockCoverage(TypeStmt, 6, 16, 8, 3, 1, 0),
				newBlockCoverage(TypeStmt, 9, 2, 9, 12, 1, 1),
			},
			"\x1b[33m 3\x1b[0m|  | import \"fmt\"",
			"\x1b[33m 7\x1b[0m|  | \x1b[31m\t\treturn fmt.Errorf(\"error: %s\", in)\x1b[0m",
			"\x1b[33m 9\x1b[0m| \x1b[92m1\x1b[0m| \t\x1b[32mreturn nil\x1b[0m",
		},
	}

	for _, tt := range tests {
		fc := &FileCoverage{
			Blocks: tt.blocks,
			cache:  map[int]BlockCoverages{},
		}
		src := strings.NewReader(code)
		dest := new(bytes.Buffer)
		if err := NewPrinter(fc).Print(src, dest); err != nil {
			t.Error(err)
		}
		lines := strings.Split(dest.String(), "\n")
		if len(lines) != 11 {
			t.Fatalf("invalid dest\n%#v", lines)
		}
		if got := lines[2]; got != tt.want3 {
			t.Errorf("got\n%v\n%#v\n\nwant\n%v\n%#v", got, got, tt.want3, tt.want3)
		}
		if got := lines[6]; got != tt.want7 {
			t.Errorf("got\n%v\n%#v\n\nwant\n%v\n%#v", got, got, tt.want7, tt.want7)
		}
		if got := lines[8]; got != tt.want9 {
			t.Errorf("got\n%v\n%#v\n\nwant\n%v\n%#v", got, got, tt.want9, tt.want9)
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
