package internal

import (
	"path/filepath"
	"strings"
)

func GeneratePrefix(wd, p string) string {
	prefix := p
	for {
		if strings.HasSuffix(wd, prefix) {
			prefix += "/"
			break
		}
		if prefix == "." || prefix == "/" {
			prefix = ""
			break
		}
		prefix = filepath.Dir(prefix)
	}
	return prefix
}
