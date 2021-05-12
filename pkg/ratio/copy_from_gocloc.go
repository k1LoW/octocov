package ratio

// original: https://github.com/hhatto/gocloc/blob/b2dad3847df87ab84c56bb8d27c91ca041e69c16/language.go
import (
	"bufio"
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	enry "github.com/go-enry/go-enry/v2"
)

var reShebangEnv = regexp.MustCompile("^#! *(\\S+/env) ([a-zA-Z]+)")
var reShebangLang = regexp.MustCompile("^#! *[.a-zA-Z/]+/([a-zA-Z]+)")

var shebang2ext = map[string]string{
	"gosh":    "scm",
	"make":    "make",
	"perl":    "pl",
	"rc":      "plan9sh",
	"python":  "py",
	"ruby":    "rb",
	"escript": "erl",
}

func getShebang(line string) (shebangLang string, ok bool) {
	ret := reShebangEnv.FindAllStringSubmatch(line, -1)
	if ret != nil && len(ret[0]) == 3 {
		shebangLang = ret[0][2]
		if sl, ok := shebang2ext[shebangLang]; ok {
			return sl, ok
		}
		return shebangLang, true
	}

	ret = reShebangLang.FindAllStringSubmatch(line, -1)
	if ret != nil && len(ret[0]) >= 2 {
		shebangLang = ret[0][1]
		if sl, ok := shebang2ext[shebangLang]; ok {
			return sl, ok
		}
		return shebangLang, true
	}

	return "", false
}

func getFileType(path string) (ext string, ok bool) {
	ext = filepath.Ext(path)
	base := filepath.Base(path)

	switch ext {
	case ".m", ".v", ".fs", ".r", ".ts":
		content, err := ioutil.ReadFile(filepath.Clean(path))
		if err != nil {
			return "", false
		}
		lang := enry.GetLanguage(path, content)
		log.Printf("path=%v, lang=%v\n", path, lang)
		return lang, true
	}

	switch base {
	case "meson.build", "meson_options.txt":
		return "meson", true
	case "CMakeLists.txt":
		return "cmake", true
	case "configure.ac":
		return "m4", true
	case "Makefile.am":
		return "makefile", true
	case "build.xml":
		return "Ant", true
	case "pom.xml":
		return "maven", true
	}

	switch strings.ToLower(base) {
	case "makefile":
		return "makefile", true
	case "nukefile":
		return "nu", true
	case "rebar": // skip
		return "", false
	}

	shebangLang, ok := getFileTypeByShebang(path)
	if ok {
		return shebangLang, true
	}

	if len(ext) >= 2 {
		return ext[1:], true
	}
	return ext, ok
}

func getFileTypeByShebang(path string) (shebangLang string, ok bool) {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		return // ignore error
	}
	defer func() {
		_ = f.Close()
	}()

	reader := bufio.NewReader(f)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return
	}
	line = bytes.TrimLeftFunc(line, unicode.IsSpace)

	if len(line) > 2 && line[0] == '#' && line[1] == '!' {
		return getShebang(string(line))
	}
	return
}
