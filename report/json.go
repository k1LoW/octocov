package report

import "github.com/goccy/go-json"

func (r *Report) UnmarshalJSON(b []byte) error {
	type Alias Report
	s := &struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	for _, set := range r.CustomMetrics {
		set.report = r
	}
	return nil
}
