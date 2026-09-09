package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k1LoW/octocov/coverage"
)

func TestLoadNamesTheReportItRejects(t *testing.T) {
	// BlockCoverage.UnmarshalJSON rejects a block whose range no source file could have, and it
	// cannot know which report it is decoding, so the error must carry the path.
	r := &Report{Coverage: &coverage.Coverage{
		Type: coverage.TypeStmt,
		Files: coverage.FileCoverages{
			&coverage.FileCoverage{File: "x.go", Type: coverage.TypeStmt, Blocks: coverage.BlockCoverages{
				&coverage.BlockCoverage{Type: coverage.TypeStmt, StartLine: new(5), StartCol: new(1), EndLine: new(7), EndCol: new(1), NumStmt: new(1), Count: new(coverage.ExecCount(1))},
			}},
		},
	}}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	content := strings.Replace(string(b), `"end_line":7`, `"end_line":9223372036854775806`, 1)
	if content == string(b) {
		t.Fatal("fixture did not take")
	}
	path := filepath.Join(t.TempDir(), "report.json")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	err = (&Report{}).Load(path)
	if err == nil {
		t.Fatal("want error")
	}
	for _, want := range []string{path, "block reaches line"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	}
}
