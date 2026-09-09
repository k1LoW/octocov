package coverage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// BenchmarkParseReportAndExclude measures the path report.MeasureCoverage takes for one LOC
// report, ParseReport followed by Exclude with no patterns, on a generated LCOV tracefile.
func BenchmarkParseReportAndExclude(b *testing.B) {
	const files, lines = 2000, 200
	var sb strings.Builder
	for f := range files {
		fmt.Fprintf(&sb, "SF:src/file%04d.ts\n", f)
		for l := 1; l <= lines; l++ {
			fmt.Fprintf(&sb, "DA:%d,%d\n", l, l%3)
		}
		sb.WriteString("end_of_record\n")
	}
	path := filepath.Join(b.TempDir(), "lcov.info")
	if err := os.WriteFile(path, []byte(sb.String()), 0600); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		cov, _, err := NewLcov().ParseReport(path)
		if err != nil {
			b.Fatal(err)
		}
		if err := cov.Exclude(nil); err != nil {
			b.Fatal(err)
		}
	}
}
