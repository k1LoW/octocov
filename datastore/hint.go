package datastore

import "github.com/k1LoW/octocov/report"

type hint struct {
	root   string
	report *report.Report
}

type HintFunc func(*hint) error

// Root hint for local datastore
func Root(p string) HintFunc {
	return func(h *hint) error {
		h.root = p
		return nil
	}
}

// Report hint for artifact datastore
func Report(r *report.Report) HintFunc {
	return func(h *hint) error {
		h.report = r
		return nil
	}
}
