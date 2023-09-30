package cmd

import (
	"fmt"
	"os"
	"path/filepath"
)

func addReportContentToSummary(content string) error {
	p := os.Getenv("GITHUB_STEP_SUMMARY")
	if _, err := os.Stat(p); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Clean(p), os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer func() {
		_ = f.Close() //nostyle:handlerrors
	}()
	if _, err := fmt.Fprintln(f, content); err != nil {
		return err
	}
	return nil
}
