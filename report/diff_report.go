package report

import (
	"fmt"
	"io"
	"time"

	"github.com/k1LoW/octocov/pkg/coverage"
	"github.com/k1LoW/octocov/pkg/ratio"
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

	table.SetHeader([]string{"", makeHeadTitle(d.RefA, d.CommitA, d.ReportA.rp), makeHeadTitle(d.RefB, d.CommitB, d.ReportB.rp), "+/-"})
	table.SetAutoFormatHeaders(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("-")
	table.SetHeaderLine(true)
	table.SetBorder(false)
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})

	g := tablewriter.Colors{tablewriter.Bold, tablewriter.FgGreenColor}
	r := tablewriter.Colors{tablewriter.Bold, tablewriter.FgRedColor}

	if d.Coverage != nil {
		{
			dd := d.Coverage.Diff
			ds := fmt.Sprintf("%.1f%%", dd)
			cc := tablewriter.Colors{}
			if dd > 0 {
				ds = fmt.Sprintf("+%.1f%%", dd)
				cc = g
			} else if dd < 0 {
				ds = fmt.Sprintf("%.1f%%", dd)
				cc = r
			}
			table.Rich([]string{"Coverage", fmt.Sprintf("%.1f%%", d.Coverage.A), fmt.Sprintf("%.1f%%", d.Coverage.B), ds}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}, tablewriter.Colors{}, cc})
		}
		if d.Coverage.CoverageA != nil && d.Coverage.CoverageB != nil {
			{
				dd := len(d.Coverage.CoverageA.Files) - len(d.Coverage.CoverageB.Files)
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Files", fmt.Sprintf("%d", len(d.Coverage.CoverageA.Files)), fmt.Sprintf("%d", len(d.Coverage.CoverageB.Files)), ds})
			}

			{
				dd := d.Coverage.CoverageA.Total - d.Coverage.CoverageB.Total
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Lines", fmt.Sprintf("%d", d.Coverage.CoverageA.Total), fmt.Sprintf("%d", d.Coverage.CoverageB.Total), ds})
			}

			{
				dd := d.Coverage.CoverageA.Covered - d.Coverage.CoverageB.Covered
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Covered", fmt.Sprintf("%d", d.Coverage.CoverageA.Covered), fmt.Sprintf("%d", d.Coverage.CoverageB.Covered), ds})
			}
		}

	}
	if d.CodeToTestRatio != nil {
		dd := d.CodeToTestRatio.Diff
		ds := fmt.Sprintf("%.1f", dd)
		cc := tablewriter.Colors{}
		if dd > 0 {
			ds = fmt.Sprintf("+%.1f", dd)
			cc = g
		} else if dd < 0 {
			ds = fmt.Sprintf("%.1f", dd)
			cc = r
		}
		table.Rich([]string{"Code to Test Ratio", fmt.Sprintf("1:%.1f", d.CodeToTestRatio.A), fmt.Sprintf("1:%.1f", d.CodeToTestRatio.B), ds}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}, tablewriter.Colors{}, cc})

		if d.CodeToTestRatio.RatioA != nil && d.CodeToTestRatio.RatioB != nil {
			{
				dd := d.CodeToTestRatio.RatioA.Code - d.CodeToTestRatio.RatioB.Code
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Code", fmt.Sprintf("%d", d.CodeToTestRatio.RatioA.Code), fmt.Sprintf("%d", d.CodeToTestRatio.RatioB.Code), ds})
			}
			{
				dd := d.CodeToTestRatio.RatioA.Test - d.CodeToTestRatio.RatioB.Test
				ds := fmt.Sprintf("%d", dd)
				if dd > 0 {
					ds = fmt.Sprintf("+%d", dd)
				}
				table.Append([]string{"  Test", fmt.Sprintf("%d", d.CodeToTestRatio.RatioA.Test), fmt.Sprintf("%d", d.CodeToTestRatio.RatioB.Test), ds})
			}
		}
	}
	if d.TestExecutionTime != nil {
		a := "-"
		b := "-"
		if d.TestExecutionTime.A != nil {
			a = time.Duration(*d.TestExecutionTime.A).String()
		}
		if d.TestExecutionTime.B != nil {
			b = time.Duration(*d.TestExecutionTime.B).String()
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
		table.Rich([]string{"Test Execution Time", a, b, ds}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}, tablewriter.Colors{}, cc})
	}

	table.Render()
}
