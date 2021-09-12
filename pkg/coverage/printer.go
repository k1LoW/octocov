package coverage

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

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
	c2 := 0
	if p.fc != nil {
		for _, b := range p.fc.Blocks {
			if *b.Count > c2 {
				c2 = *b.Count
			}
		}
	}
	w2 := len(strconv.Itoa(c2))

	scanner := bufio.NewScanner(r2)
	n := 1
	cl := color.New(color.FgYellow)
	cl.EnableColor()
	for scanner.Scan() {
		c, out := paintLine(n, w2, scanner.Text(), p.fc.FindBlocksByLine(n))
		_, _ = fmt.Fprintf(dest, "%s %s %s\n", cl.Sprint(fmt.Sprintf(fmt.Sprintf("%%%dd", w), n)), c, out)
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

func paintLine(n, w int, in string, blocks BlockCoverages) (string, string) {
	g := color.New(color.FgGreen)
	g.EnableColor()
	r := color.New(color.FgRed)
	r.EnableColor()

	lc := len(in)
	l := make([]string, lc)

	c := 0
	for _, b := range blocks {
		var cl string
		if *b.Count > 0 {
			cl = "g"
		} else {
			cl = "r"
		}
		switch b.Type {
		case TypeLOC:
			for i := 0; i < lc; i++ {
				l[i] = cl
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
				l[i] = cl
			}
		}

		if *b.Count > c {
			c = *b.Count
		}
	}

	out := ""
	pos := 0
	current := ""
	for i, cl := range l {
		if current == cl {
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
		current = cl
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

	s := strings.Repeat(" ", w)
	if c > 0 {
		g := color.New(color.FgHiGreen)
		g.EnableColor()
		s = g.Sprintf(fmt.Sprintf("%%%dd", w), c)
	}

	return s, out
}
