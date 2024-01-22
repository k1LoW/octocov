package report

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/olekukonko/tablewriter"
	"github.com/samber/lo"
	"github.com/xeipuuv/gojsonschema"
)

const swapXYMin = 5

//go:embed custom_metrics_schema.json
var schema []byte

type MetadataKV struct {
	Key   string `json:"key"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value"`
}

type CustomMetricSet struct {
	Key      string          `json:"key"`
	Name     string          `json:"name,omitempty"`
	Metadata []*MetadataKV   `json:"metadata,omitempty"`
	Metrics  []*CustomMetric `json:"metrics"`
	report   *Report
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
	if len(s.Metrics) >= swapXYMin {
		return s.tableSwaped()
	}
	report := s.report
	if report == nil {
		report = &Report{}
	}
	var (
		h []string
		d []string
	)
	for _, m := range s.Metrics {
		h = append(h, m.Name)
		d = append(d, fmt.Sprintf("%s%s", report.convertFormat(m.Value), m.Unit))
	}
	buf := new(bytes.Buffer)
	_, _ = buf.WriteString(fmt.Sprintf("## %s\n\n", s.Name)) //nostyle:handlerrors
	table := tablewriter.NewWriter(buf)
	table.SetHeader(h)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.Append(d)
	table.Render()
	return strings.Replace(buf.String(), "---|", "--:|", len(h))
}

func (s *CustomMetricSet) tableSwaped() string {
	buf := new(bytes.Buffer)
	_, _ = buf.WriteString(fmt.Sprintf("## %s\n\n", s.Name)) //nostyle:handlerrors
	table := tablewriter.NewWriter(buf)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetHeader([]string{"", makeHeadTitleWithLink(s.report.Ref, s.report.Commit, nil)})

	report := s.report
	for _, m := range s.Metrics {
		table.Append([]string{m.Name, fmt.Sprintf("%s%s", report.convertFormat(m.Value), m.Unit)})
	}
	table.Render()
	return strings.Replace(buf.String(), "---|", "--:|", len(s.Metrics))
}

func (s *CustomMetricSet) MetadataTable() string {
	if len(s.Metadata) == 0 {
		return ""
	}
	var h []string
	var d []string
	for _, m := range s.Metadata {
		if m.Name == "" {
			m.Name = m.Key
		}
		h = append(h, m.Name)
		d = append(d, m.Value)
	}
	buf := new(bytes.Buffer)
	buf.WriteString("<details><summary>Metadata</summary>\n\n")
	table := tablewriter.NewWriter(buf)
	table.SetHeader(h)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.Append(d)
	table.Render()
	buf.WriteString("\n</details>\n")
	return strings.Replace(buf.String(), "---|", "--:|", len(h))
}

func (s *CustomMetricSet) Out(w io.Writer) error {
	if len(s.Metrics) == 0 {
		return nil
	}
	table := tablewriter.NewWriter(w)
	if s.Name == "" {
		s.Name = s.Key
	}
	table.SetHeader([]string{s.Name, makeHeadTitle(s.report.Ref, s.report.Commit, s.report.covPaths)})
	table.SetHeaderColor(tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{})
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("-")
	table.SetHeaderLine(true)
	table.SetBorder(false)
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT})

	report := s.report
	if report == nil {
		report = &Report{}
	}

	for _, m := range s.Metrics {
		if m.Name == "" {
			m.Name = m.Key
		}
		table.Rich([]string{m.Name, fmt.Sprintf("%s%s", report.convertFormat(m.Value), m.Unit)}, []tablewriter.Colors{tablewriter.Colors{tablewriter.Bold}, tablewriter.Colors{}})
	}

	table.Render()
	return nil
}

func (s *CustomMetricSet) Compare(s2 *CustomMetricSet) *DiffCustomMetricSet {
	d := &DiffCustomMetricSet{
		Key:  s.Key,
		Name: s.Name,
		A:    s,
		B:    s2,
	}
	if s2 == nil {
		for _, m := range s.Metrics {
			d.Metrics = append(d.Metrics, m.Compare(nil))
		}
		return d
	}
	for _, m := range s.Metrics {
		m2 := s2.findMetricByKey(m.Key)
		d.Metrics = append(d.Metrics, m.Compare(m2))
	}

	return d
}

func (s *CustomMetricSet) findMetricByKey(key string) *CustomMetric {
	for _, m := range s.Metrics {
		if m.Key == key {
			return m
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

func (s *CustomMetricSet) Validate() error {
	cs := gojsonschema.NewBytesLoader(schema)
	cd := gojsonschema.NewGoLoader(s)
	result, err := gojsonschema.Validate(cs, cd)
	if err != nil {
		return err
	}
	if !result.Valid() {
		var errs error
		for _, err := range result.Errors() {
			errs = errors.Join(errs, errors.New(err.String()))
		}
		return errs
	}
	if len(s.Metrics) != len(lo.UniqBy(s.Metrics, func(m *CustomMetric) string {
		return m.Key
	})) {
		return fmt.Errorf("key of metrics must be unique: %s", lo.Map(s.Metrics, func(m *CustomMetric, _ int) string {
			return m.Key
		}))
	}
	if len(s.Metadata) != len(lo.UniqBy(s.Metadata, func(m *MetadataKV) string {
		return m.Key
	})) {
		return fmt.Errorf("key of metadata must be unique: %s", lo.Map(s.Metadata, func(m *MetadataKV, _ int) string {
			return m.Key
		}))
	}

	return nil
}

func (d *DiffCustomMetricSet) Table() string {
	if len(d.Metrics) == 0 {
		return ""
	}
	if d.B == nil {
		return d.A.Table()
	}
	buf := new(bytes.Buffer)
	if d.Name == "" {
		d.Name = d.Key
	}
	_, _ = buf.WriteString(fmt.Sprintf("## %s\n\n", d.Name)) //nostyle:handlerrors
	table := tablewriter.NewWriter(buf)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	table.SetHeader([]string{"", makeHeadTitleWithLink(d.B.report.Ref, d.B.report.Commit, nil), makeHeadTitleWithLink(d.A.report.Ref, d.A.report.Commit, nil), "+/-"})
	report := d.report()

	for _, m := range d.Metrics {
		var va, vb, diff string
		switch {
		case m.A == nil && m.B == nil:
			continue
		case m.A != nil && m.B == nil:
			vb = ""
			va = fmt.Sprintf("%s%s", report.convertFormat(*m.A), m.customMetricA.Unit)
			diff = fmt.Sprintf("%s%s", report.convertFormat(m.Diff), m.customMetricA.Unit)
		case m.A == nil && m.B != nil:
			va = ""
			vb = fmt.Sprintf("%s%s", report.convertFormat(*m.B), m.customMetricB.Unit)
			diff = fmt.Sprintf("%s%s", report.convertFormat(m.Diff), m.customMetricB.Unit)
		case isInt(*m.A) && isInt(*m.B):
			va = fmt.Sprintf("%s%s", report.convertFormat(*m.A), m.customMetricA.Unit)
			vb = fmt.Sprintf("%s%s", report.convertFormat(*m.B), m.customMetricB.Unit)
			diff = fmt.Sprintf("%s%s", report.convertFormat(m.Diff), m.customMetricA.Unit)
		default:
			va = fmt.Sprintf("%s%s", report.convertFormat(*m.A), m.customMetricA.Unit)
			vb = fmt.Sprintf("%s%s", report.convertFormat(*m.B), m.customMetricB.Unit)
			diff = fmt.Sprintf("%s%s", report.convertFormat(m.Diff), m.customMetricA.Unit)
		}
		if m.Name == "" {
			m.Name = m.Key
		}
		table.Append([]string{fmt.Sprintf("**%s**", m.Name), vb, va, diff})
	}
	table.Render()
	return strings.Replace(strings.Replace(buf.String(), "---|", "--:|", 4), "--:|", "---|", 1)
}

func (d *DiffCustomMetricSet) MetadataTable() string {
	if len(d.A.Metadata) == 0 {
		return ""
	}
	if d.B == nil || len(d.B.Metadata) == 0 {
		return d.A.MetadataTable()
	}
	buf := new(bytes.Buffer)
	buf.WriteString("<details><summary>Metadata</summary>\n\n")
	table := tablewriter.NewWriter(buf)
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT})
	table.SetHeader([]string{"", makeHeadTitleWithLink(d.B.report.Ref, d.B.report.Commit, nil), makeHeadTitleWithLink(d.A.report.Ref, d.A.report.Commit, nil)})
	for _, ma := range d.A.Metadata {
		mb, ok := lo.Find(d.B.Metadata, func(m *MetadataKV) bool {
			return m.Key == ma.Key
		})
		if !ok {
			mb = &MetadataKV{}
		}
		if ma.Name == "" {
			ma.Name = ma.Key
		}
		table.Append([]string{fmt.Sprintf("**%s**", ma.Name), mb.Value, ma.Value})
	}
	table.Render()
	buf.WriteString("\n</details>\n")
	return strings.Replace(strings.Replace(buf.String(), "---|", "--:|", 4), "--:|", "---|", 1)
}

func (d *DiffCustomMetricSet) report() *Report {
	if d.A != nil && d.A.report != nil {
		return d.A.report
	}
	if d.B != nil && d.B.report != nil {
		return d.B.report
	}

	return &Report{}
}

func isInt(v float64) bool {
	return v == float64(int64(v))
}
