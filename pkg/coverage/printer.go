package coverage

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/fatih/color"
)

type Printer struct {
	fc *FileCoverage
}

func NewPrinter(fc *FileCoverage) *Printer {
	return &Printer{
		fc: fc,
	}
}

func (p *Printer) Print(src io.Reader, dest io.Writer) error {
	r2 := new(bytes.Buffer)
	r1 := io.TeeReader(src, r2)

	c, err := countLines(r1)
	if err != nil {
		return err
	}
	w := len(strconv.Itoa(c))
	scanner := bufio.NewScanner(r2)
	n := 1
	cl := color.New(color.FgYellow)
	cl.EnableColor()
	for scanner.Scan() {
		_, _ = fmt.Fprintf(dest, "%s %s\n", cl.Sprint(fmt.Sprintf(fmt.Sprintf("%%%dd", w), n)), paintLine(n, scanner.Text(), p.fc.FindBlocksByLine(n)))
		n += 1
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func countLines(r io.Reader) (int, error) {
	buf := make([]byte, 1024)
	count := 0
	sep := []byte{'\n'}

	for {
		n, err := r.Read(buf)
		if err != nil {
			if n == 0 && err == io.EOF {
				err = nil
			}
			return count, err
		}

		count += bytes.Count(buf[:n], sep)
	}
}

func paintLine(n int, in string, blocks BlockCoverages) string {
	g := color.New(color.FgGreen)
	g.EnableColor()
	r := color.New(color.FgRed)
	r.EnableColor()

	lc := len(in)
	l := make([]string, lc)

	for _, b := range blocks {
		var c string
		if *b.Count > 0 {
			c = "g"
		} else {
			c = "r"
		}
		switch b.Type {
		case TypeLOC:
			for i := 0; i < lc; i++ {
				l[i] = c
			}
		case TypeStmt:
			s := 0
			if *b.StartLine == n {
				s = *b.StartCol - 1
			}
			e := lc
			if *b.EndLine == n {
				e = *b.EndCol - 1
			}
			for i := s; i < e; i++ {
				l[i] = c
			}
		}
	}

	out := ""
	pos := 0
	current := ""
	for i, c := range l {
		if current == c {
			continue
		}
		switch current {
		case "":
			out += in[pos:i]
		case "g":
			out += g.Sprint(in[pos:i])
		case "r":
			out += r.Sprint(in[pos:i])
		}
		current = c
		pos = i
	}
	switch current {
	case "":
		out += in[pos:lc]
	case "g":
		out += g.Sprint(in[pos:lc])
	case "r":
		out += r.Sprint(in[pos:lc])
	}

	return out
}
