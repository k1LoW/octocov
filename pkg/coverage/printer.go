package coverage

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/spiegel-im-spiegel/gnkf/enc"
	"github.com/spiegel-im-spiegel/gnkf/guess"
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
	dup := new(bytes.Buffer)
	io.Copy(dup, src)
	b := dup.Bytes()
	c := bytes.Count(b, []byte{'\n'})

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

	e, err := guess.EncodingBytes(b)
	if err != nil {
		return err
	}
	dup2 := new(bytes.Buffer)
	if err := enc.Convert("UTF-8", dup2, e[0], dup); err != nil {
		return err
	}

	scanner := bufio.NewScanner(dup2)
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
			if *b.StartLine == n && b.StartCol != nil {
				s = *b.StartCol - 1
			}
			e := lc
			if *b.EndLine == n && b.EndCol != nil {
				e = *b.EndCol - 1
			}
			if e > lc {
				// coverage report and source code are out of sync.
				e = lc
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
