package report

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
)

type CustomMetricSet struct {
	Key     string          `json:"key"`
	Name    string          `json:"name,omitempty"`
	Metrics []*CustomMetric `json:"metrics"`
	report  *Report
}

type CustomMetric struct {
	Key   string  `json:"key"`
	Name  string  `json:"name,omitempty"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}

type DiffCustomMetricSet struct {
	Key     string              `json:"key"`
	Name    string              `json:"name,omitempty"`
	A       *CustomMetricSet    `json:"a"`
	B       *CustomMetricSet    `json:"b"`
	Metrics []*DiffCustomMetric `json:"metrics"`
	reportA *Report
	reportB *Report
}

type DiffCustomMetric struct {
	Key           string   `json:"key"`
	Name          string   `json:"name,omitempty"`
	A             *float64 `json:"a"`
	B             *float64 `json:"b"`
	Diff          float64  `json:"diff"`
	customMetricA *CustomMetric
	customMetricB *CustomMetric
}

func (s *CustomMetricSet) Table() string {
	if len(s.Metrics) == 0 {
		return ""
	}
	h := []string{}
	m := []string{}
	for _, metric := range s.Metrics {
		h = append(h, metric.Name)
		m = append(m, fmt.Sprintf("%.1f%s", metric.Value, metric.Unit))
	}
	buf := new(bytes.Buffer)
	_, _ = buf.WriteString(fmt.Sprintf("## %s\n\n", s.Name))
	table := tablewriter.NewWriter(buf)
	table.SetHeader(h)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.Append(m)
	table.Render()
	return strings.Replace(buf.String(), "---|", "--:|", len(h))
}

func (s *CustomMetricSet) Out(w io.Writer) error {
	if len(s.Metrics) == 0 {
		return nil
	}
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{s.Name, makeHeadTitle(s.report.Ref, s.report.Commit, s.report.covPaths)})
	table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{})
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("-")
	table.SetHeaderLine(true)
	table.SetBorder(false)

	for _, metric := range s.Metrics {
		table.Rich([]string{metric.Name, fmt.Sprintf("%.1f%s", metric.Value, metric.Unit)}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}})
	}

	table.Render()
	return nil
}

func (s *CustomMetricSet) Compare(s2 *CustomMetricSet) *DiffCustomMetricSet {
	d := &DiffCustomMetricSet{
		Key:     s.Key,
		Name:    s.Name,
		A:       s,
		B:       s2,
		Metrics: []*DiffCustomMetric{},
		reportA: s.report,
	}
	if s2 == nil {
		for _, metric := range s.Metrics {
			d.Metrics = append(d.Metrics, metric.Compare(nil))
		}
		return d
	}
	d.reportB = s2.report
	for _, metric := range s.Metrics {
		metric2 := s2.findMetricByKey(metric.Key)
		d.Metrics = append(d.Metrics, metric.Compare(metric2))
	}

	return d
}

func (s *CustomMetricSet) findMetricByKey(key string) *CustomMetric {
	for _, metric := range s.Metrics {
		if metric.Key == key {
			return metric
		}
	}
	return nil
}

func (m *CustomMetric) Compare(m2 *CustomMetric) *DiffCustomMetric {
	d := &DiffCustomMetric{
		Key:           m.Key,
		Name:          m.Name,
		customMetricA: m,
		customMetricB: m2,
	}
	var v1, v2 float64
	v1 = m.Value
	d.A = &v1
	if m2 != nil {
		v2 = m2.Value
		d.B = &v2
	}
	d.Diff = v1 - v2

	return d
}

func (d *DiffCustomMetricSet) Table() string {
	if len(d.Metrics) == 0 {
		return ""
	}
	if d.B == nil {
		return d.A.Table()
	}
	buf := new(bytes.Buffer)
	_, _ = buf.WriteString(fmt.Sprintf("## %s\n\n", d.Name))
	table := tablewriter.NewWriter(buf)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})

	table.SetHeader([]string{"", makeHeadTitleWithLink(d.reportB.Ref, d.reportB.Commit, nil), makeHeadTitleWithLink(d.reportA.Ref, d.reportA.Commit, nil), "+/-"})
	for _, metric := range d.Metrics {
		table.Append([]string{metric.Name, fmt.Sprintf("%.1f%s", *metric.B, metric.customMetricB.Unit), fmt.Sprintf("%.1f%s", *metric.A, metric.customMetricA.Unit), fmt.Sprintf("%.1f%s", metric.Diff, metric.customMetricA.Unit)})
	}
	table.Render()
	return strings.Replace(strings.Replace(buf.String(), "---|", "--:|", 4), "--:|", "---|", 1)
}
