package report

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"github.com/k1LoW/octocov/coverage"
	"github.com/k1LoW/octocov/gh"
	"github.com/k1LoW/octocov/ratio"
	"github.com/olekukonko/tablewriter"
)

type DiffReport struct {
	RepositoryA       string                 `json:"repository_a"`
	RepositoryB       string                 `json:"repository_b"`
	RefA              string                 `json:"ref_a"`
	RefB              string                 `json:"ref_b"`
	CommitA           string                 `json:"commit_a"`
	CommitB           string                 `json:"commit_b"`
	Coverage          *coverage.DiffCoverage `json:"coverage,omitempty"`
	CodeToTestRatio   *ratio.DiffRatio       `json:"code_to_test_ratio,omitempty"`
	TestExecutionTime *DiffTestExecutionTime `json:"test_execution_time,omitempty"`
	CustomMetrics     []*DiffCustomMetricSet `json:"custom_metrics,omitempty"`
	TimestampA        time.Time              `json:"timestamp_a"`
	TimestampB        time.Time              `json:"timestamp_b"`
	ReportA           *Report                `json:"-"`
	ReportB           *Report                `json:"-"`
}

type DiffTestExecutionTime struct {
	A                  *float64 `json:"a"`
	B                  *float64 `json:"b"`
	Diff               float64  `json:"diff"`
	TestExecutionTimeA *float64 `json:"-"`
	TestExecutionTimeB *float64 `json:"-"`
}

func (d *DiffReport) Out(w io.Writer) {
	table := tablewriter.NewWriter(w)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("-")
	table.SetHeaderLine(true)
	table.SetBorder(false)
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	g := tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor}
	r := tablewriter.Colors{tablewriter.Bold, tablewriter.FgRedColor}
	b := tablewriter.Colors{tablewriter.Bold}

	d.renderTable(table, g, r, b, true, false)

	table.Render()
}

var leftSepRe = regexp.MustCompile(`(?m)^\|`)

func (d *DiffReport) Table() string {
	var out []string

	// Markdown table
	buf := new(bytes.Buffer)
	table := tablewriter.NewWriter(buf)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	d.renderTable(table, tablewriter.Colors{}, tablewriter.Colors{}, tablewriter.Colors{}, false, true)
	table.Render()
	out = append(out, strings.Replace(strings.Replace(buf.String(), "---|", "--:|", 4), "--:|", "---|", 1))

	// Diff code block
	buf2 := new(bytes.Buffer)
	table2 := tablewriter.NewWriter(buf2)
	table2.SetAutoFormatHeaders(false)
	table2.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table2.SetCenterSeparator("|")
	table2.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	d.renderTable(table2, tablewriter.Colors{}, tablewriter.Colors{}, tablewriter.Colors{}, true, false)
	table2.Render()
	t2 := leftSepRe.ReplaceAllString(buf2.String(), "  |")
	if d.Coverage != nil {
		if d.Coverage.Diff > 0 {
			t2 = strings.Replace(t2, "  | Coverage", "+ | Coverage", 1)
		} else if d.Coverage.Diff < 0 {
			t2 = strings.Replace(t2, "  | Coverage", "- | Coverage", 1)
		}
		if d.Coverage.CoverageA != nil && d.Coverage.CoverageB != nil {
			if d.Coverage.CoverageA.Covered < d.Coverage.CoverageB.Covered {
				t2 = strings.Replace(t2, "  |   Covered", "- |   Covered", 1)
			} else if d.Coverage.CoverageA.Covered > d.Coverage.CoverageB.Covered {
				t2 = strings.Replace(t2, "  |   Covered", "+ |   Covered", 1)
			}
		}
	}
	if d.CodeToTestRatio != nil {
		if d.CodeToTestRatio.Diff > 0 {
			t2 = strings.Replace(t2, "  | Code to", "+ | Code to", 1)
		} else if d.CodeToTestRatio.Diff < 0 {
			t2 = strings.Replace(t2, "  | Code to", "- | Code to", 1)
		}
		if d.CodeToTestRatio.RatioA != nil && d.CodeToTestRatio.RatioB != nil {
			if d.CodeToTestRatio.RatioA.Test < d.CodeToTestRatio.RatioB.Test {
				t2 = strings.Replace(t2, "  |   Test", "- |   Test", 1)
			} else if d.CodeToTestRatio.RatioA.Test > d.CodeToTestRatio.RatioB.Test {
				t2 = strings.Replace(t2, "  |   Test", "+ |   Test", 1)
			}
		}
	}
	if d.TestExecutionTime != nil {
		if d.TestExecutionTime.Diff > 0 {
			t2 = strings.Replace(t2, "  | Test Execution", "- | Test Execution", 1)
		} else if d.TestExecutionTime.Diff < 0 {
			t2 = strings.Replace(t2, "  | Test Execution", "+ | Test Execution", 1)
		}
	}
	out = append(out, fmt.Sprintf("<details>\n\n<summary>Details</summary>\n\n``` diff\n%s```\n\n</details>\n", t2))

	return strings.Join(out, "\n")
}

func (d *DiffReport) renderTable(table *tablewriter.Table, g, r, b tablewriter.Colors, detail bool, withLink bool) {
	if withLink {
		table.SetHeader([]string{"", makeHeadTitleWithLink(d.RefB, d.CommitB, d.ReportB.covPaths), makeHeadTitleWithLink(d.RefA, d.CommitA, d.ReportA.covPaths), "+/-"})
	} else {
		table.SetHeader([]string{"", makeHeadTitle(d.RefB, d.CommitB, d.ReportB.covPaths), makeHeadTitle(d.RefA, d.CommitA, d.ReportA.covPaths), "+/-"})
	}
	if d.Coverage != nil {
		{
			dd := d.Coverage.Diff
			ds := fmt.Sprintf("%.1f%%", floor1(dd))
			cc := tablewriter.Colors{}
			if dd > 0 {
				ds = fmt.Sprintf("+%.1f%%", floor1(dd))
				cc = g
			} else if dd < 0 {
				ds = fmt.Sprintf("%.1f%%", floor1(dd))
				cc = r
			}
			t := "Coverage"
			if !detail {
				t = "**Coverage**"
			}
			table.Rich([]string{t, fmt.Sprintf("%.1f%%", floor1(d.Coverage.B)), fmt.Sprintf("%.1f%%", floor1(d.Coverage.A)), ds}, []tablewriter.Colors{b, tablewriter.Colors{}, tablewriter.Colors{}, cc})
		}
		if detail && d.Coverage.CoverageA != nil && d.Coverage.CoverageB != nil {
			{
				dd := len(d.Coverage.CoverageA.Files) - len(d.Coverage.CoverageB.Files)
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Files", fmt.Sprintf("%d", len(d.Coverage.CoverageB.Files)), fmt.Sprintf("%d", len(d.Coverage.CoverageA.Files)), ds})
			}

			{
				dd := d.Coverage.CoverageA.Total - d.Coverage.CoverageB.Total
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Lines", fmt.Sprintf("%d", d.Coverage.CoverageB.Total), fmt.Sprintf("%d", d.Coverage.CoverageA.Total), ds})
			}

			{
				dd := d.Coverage.CoverageA.Covered - d.Coverage.CoverageB.Covered
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Covered", fmt.Sprintf("%d", d.Coverage.CoverageB.Covered), fmt.Sprintf("%d", d.Coverage.CoverageA.Covered), ds})
			}
		}

	}
	if d.CodeToTestRatio != nil {
		dd := d.CodeToTestRatio.Diff
		ds := fmt.Sprintf("%.1f", floor1(dd))
		cc := tablewriter.Colors{}
		if dd > 0 {
			ds = fmt.Sprintf("+%.1f", floor1(dd))
			cc = g
		} else if dd < 0 {
			ds = fmt.Sprintf("%.1f", floor1(dd))
			cc = r
		}
		ratioA := "-"
		ratioB := "-"
		if d.CodeToTestRatio.RatioA != nil && (d.CodeToTestRatio.RatioA.Code != 0 || d.CodeToTestRatio.RatioA.Test != 0) {
			ratioA = fmt.Sprintf("1:%.1f", floor1(d.CodeToTestRatio.A))
		}
		if d.CodeToTestRatio.RatioB != nil && (d.CodeToTestRatio.RatioB.Code != 0 || d.CodeToTestRatio.RatioB.Test != 0) {
			ratioB = fmt.Sprintf("1:%.1f", floor1(d.CodeToTestRatio.B))
		}
		t := "Code to Test Ratio"
		if !detail {
			t = "**Code to Test Ratio**"
		}
		table.Rich([]string{t, ratioB, ratioA, ds}, []tablewriter.Colors{b, tablewriter.Colors{}, tablewriter.Colors{}, cc})

		if detail && d.CodeToTestRatio.RatioA != nil && d.CodeToTestRatio.RatioB != nil {
			{
				dd := d.CodeToTestRatio.RatioA.Code - d.CodeToTestRatio.RatioB.Code
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Code", fmt.Sprintf("%d", d.CodeToTestRatio.RatioB.Code), fmt.Sprintf("%d", d.CodeToTestRatio.RatioA.Code), ds})
			}
			{
				dd := d.CodeToTestRatio.RatioA.Test - d.CodeToTestRatio.RatioB.Test
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Test", fmt.Sprintf("%d", d.CodeToTestRatio.RatioB.Test), fmt.Sprintf("%d", d.CodeToTestRatio.RatioA.Test), ds})
			}
		}
	}
	if d.TestExecutionTime != nil {
		ta := "-"
		tb := "-"
		if d.TestExecutionTime.A != nil {
			ta = time.Duration(*d.TestExecutionTime.A).String()
		}
		if d.TestExecutionTime.B != nil {
			tb = time.Duration(*d.TestExecutionTime.B).String()
		}
		dd := d.TestExecutionTime.Diff
		ds := time.Duration(dd).String()
		cc := tablewriter.Colors{}
		if dd > 0 {
			ds = fmt.Sprintf("+%s", time.Duration(dd).String())
			cc = r
		} else if dd < 0 {
			ds = time.Duration(dd).String()
			cc = g
		}
		t := "Test Execution Time"
		if !detail {
			t = "**Test Execution Time**"
		}
		table.Rich([]string{t, tb, ta, ds}, []tablewriter.Colors{b, tablewriter.Colors{}, tablewriter.Colors{}, cc})
	}
}

func (d *DiffReport) FileCoveragesTable(files []*gh.PullRequestFile) string {
	if d.Coverage == nil {
		return ""
	}
	if len(files) == 0 {
		return ""
	}
	var t, c, pt, pc int
	exist := false
	var rows [][]string
	for _, f := range files {
		fc, err := d.Coverage.Files.FuzzyFindByFile(f.Filename)
		if err != nil {
			continue
		}
		exist = true
		diff := fmt.Sprintf("%.1f%%", floor1(fc.Diff))
		if fc.Diff > 0 {
			diff = fmt.Sprintf("+%.1f%%", floor1(fc.Diff))
		}
		if fc.FileCoverageA != nil {
			c += fc.FileCoverageA.Covered
			t += fc.FileCoverageA.Total
		}
		if fc.FileCoverageB != nil {
			pc += fc.FileCoverageB.Covered
			pt += fc.FileCoverageB.Total
		}
		rows = append(rows, []string{fmt.Sprintf("[%s](%s)", f.Filename, f.BlobURL), fmt.Sprintf("%.1f%%", floor1(fc.A)), diff, f.Status})
	}
	if !exist {
		return ""
	}
	coverAll := float64(c) / float64(t) * 100
	if t == 0 {
		coverAll = 0.0
	}
	prevAll := float64(pc) / float64(pt) * 100
	if pt == 0 {
		prevAll = 0.0
	}
	arrow := "→"
	title := fmt.Sprintf("### Code coverage of files in pull request scope (%.1f%% %s %.1f%%)", floor1(prevAll), arrow, floor1(coverAll))
	buf := new(bytes.Buffer)
	buf.WriteString(fmt.Sprintf("%s\n\n", title))

	if len(rows) > filesSkipMax {
		buf.WriteString(fmt.Sprintf("Skip file coverages because there are too many files (%d)\n", len(rows)))
		return buf.String()
	}

	if len(rows) > filesHideMin {
		buf.WriteString("<details>\n\n")
	}

	table := tablewriter.NewWriter(buf)
	h := []string{"Files", "Coverage", "+/-", "Status"}
	table.SetHeader(h)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	for _, v := range rows {
		table.Append(v)
	}
	table.Render()

	if len(rows) > filesHideMin {
		buf.WriteString("\n</details>\n")
	}

	return strings.Replace(strings.Replace(buf.String(), "---|", "--:|", len(h)), "--:|", "---|", 1)
}
