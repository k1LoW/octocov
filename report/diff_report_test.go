package report

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiff(t *testing.T) {
	a := &Report{}
	if err := a.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	if err := b.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "awspec", "report.json")); err != nil {
		t.Fatal(err)
	}
	buf := new(bytes.Buffer)
	a.Compare(b).Out(buf)
	got := buf.String()
	if want := "master (896d3c5)  master (5d1e926)    +/-"; !strings.Contains(got, want) {
		t.Errorf("got %v\nwant %v", got, want)
	}
	if want := "  \x1b[1mCoverage\x1b[0m                        68.5%             38.8%   \x1b[1;31m-29.7%\x1b[0m"; !strings.Contains(got, want) {
		t.Errorf("got %#v\nwant %v", got, want)
	}
}

func TestDiffTable(t *testing.T) {
	t.Setenv("GITHUB_SERVER_URL", "https://github.com")
	t.Setenv("GITHUB_REPOSITORY", "k1LoW/octocov")
	a := &Report{}
	if err := a.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "tbls", "report2.json")); err != nil {
		t.Fatal(err)
	}
	b := &Report{}
	if err := b.Load(filepath.Join(testdataDir(t), "reports", "k1LoW", "awspec", "report.json")); err != nil {
		t.Fatal(err)
	}

	got := a.Compare(b).Table()
	want := `|                         | [master](https://github.com/k1LoW/octocov/tree/master) ([896d3c5](https://github.com/k1LoW/octocov/commit/896d3c59e6595e582eedbc7d7a05b4922ea88064)) | [master](https://github.com/k1LoW/octocov/tree/master) ([5d1e926](https://github.com/k1LoW/octocov/commit/5d1e926b41be6660d138ba73846c29e42784b473)) |  +/-   |
|-------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------:|-----------------------------------------------------------------------------------------------------------------------------------------------------:|-------:|
| **Coverage**            |                                                                                                                                                68.5% |                                                                                                                                                38.8% | -29.7% |
| **Code to Test Ratio**  |                                                                                                                                                1:0.5 |                                                                                                                                                1:0.0 |   -0.5 |
| **Test Execution Time** |                                                                                                                                                4m40s |                                                                                                                                                    - | -4m40s |
` + "\n<details>\n\n<summary>Details</summary>\n\n``` diff\n" + `  |                     | master (896d3c5) | master (5d1e926) |   +/-   |
  |---------------------|------------------|------------------|---------|
- | Coverage            |            68.5% |            38.8% |  -29.7% |
  |   Files             |               31 |              335 |    +304 |
  |   Lines             |             2857 |             6043 |   +3186 |
+ |   Covered           |             1957 |             2347 |    +390 |
- | Code to Test Ratio  |            1:0.5 |            1:0.0 |    -0.5 |
  |   Code              |             7202 |           947827 | +940625 |
- |   Test              |             3704 |             2757 |    -947 |
+ | Test Execution Time |            4m40s |                - |  -4m40s |
` + "```\n\n</details>\n"
	if got != want {
		t.Errorf("got\n%v\nwant\n%v", got, want)
	}
}
