package cmd

import (
	"fmt"
	"os"
)

func addReportContentToSummary(content string) error {
	p := os.Getenv("GITHUB_STEP_SUMMARY")
	if _, err := os.Stat(p); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := fmt.Fprintln(f, content); err != nil {
		return err
	}
	return nil
}
