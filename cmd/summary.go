package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/k1LoW/repin"
)

func addReportContentToSummary(content, key string) error {
	sig := generateSig(key)
	p := os.Getenv("GITHUB_STEP_SUMMARY")
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	b, err := os.ReadFile(filepath.Clean(p))
	if err != nil {
		return err
	}
	current := string(b)
	var rep string
	if strings.Count(current, sig) < 2 {
		rep = fmt.Sprintf("%s\n%s\n%s\n%s\n", current, sig, content, sig)
	} else {
		buf := new(bytes.Buffer)
		if !strings.HasSuffix(current, "\n") {
			current += "\n"
		}
		if _, err := repin.Replace(strings.NewReader(current), strings.NewReader(content), sig, sig, false, buf); err != nil {
			return err
		}
		rep = buf.String()
	}
	if err := os.WriteFile(filepath.Clean(p), []byte(rep), fi.Mode()); err != nil {
		return err
	}
	return nil
}

func generateSig(key string) string {
	if key == "" {
		return "<!-- octocov -->"
	}
	return fmt.Sprintf("<!-- octocov:%s -->", key)
}
