package report

type CustomMetricSet struct {
	Key     string          `json:"key"`
	Name    string          `json:"name,omitempty"`
	Metrics []*CustomMetric `json:"metrics"`
}

type CustomMetric struct {
	Key   string  `json:"key"`
	Name  string  `json:"name,omitempty"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit,omitempty"`
}
