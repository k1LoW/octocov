package coverage

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/goark/gnkf/enc"
	"github.com/goark/gnkf/guess"
)

const maxSrcSize = 1073741824 //1GB

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
	size, err := io.CopyN(dup, src, maxSrcSize)
	if !errors.Is(err, io.EOF) {
		return err
	}
	if size >= maxSrcSize {
		return fmt.Errorf("too large file size to copy: %d >= %d", size, maxSrcSize)
	}
	b := dup.Bytes()
	c := bytes.Count(b, []byte{'\n'})

	w := len(strconv.Itoa(c))
	w2 := len(strconv.Itoa(p.fc.Blocks.MaxCount()))

	e, err := guess.EncodingBytes(b)
	if err != nil {
		return err
	}
	dup2 := new(bytes.Buffer)
	if err := enc.Convert("UTF-8", dup2, e[0], dup); err != nil {
		return err
	}

	lcs := p.fc.Blocks.ToLineCoverages()

	scanner := bufio.NewScanner(dup2)
	n := 1
	cl := color.New(color.FgYellow)
	cl.EnableColor()
	for scanner.Scan() {
		lc, _ := lcs.FindByLine(n)
		c, out := paintLine(n, w2, scanner.Text(), lc)
		_, _ = fmt.Fprintf(dest, "%s %s %s\n", cl.Sprint(fmt.Sprintf(fmt.Sprintf("%%%dd", w), n)), c, out)
		n += 1
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

const (
	posGreen = "g"
	posRed   = "r"
)

func lineCovered(lcnt int, lc *LineCoverage) (int, []string) {
	l := make([]string, lcnt)
	if lc == nil {
		return 0, l
	}

	for i := 0; i < lcnt; i++ {
		c, err := lc.PosCoverages.FindCountByPos(i + 1)
		if err != nil {
			continue
		}
		if c > 0 {
			l[i] = posGreen
		} else {
			l[i] = posRed
		}
	}

	return lc.Count, l
}

func paintLine(n, w int, in string, lc *LineCoverage) (string, string) {
	g := color.New(color.FgGreen)
	g.EnableColor()
	r := color.New(color.FgRed)
	r.EnableColor()

	lcnt := len(in)
	c, l := lineCovered(lcnt, lc)

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
		case posGreen:
			out += g.Sprint(in[pos:i])
		case posRed:
			out += r.Sprint(in[pos:i])
		}
		current = cl
		pos = i
	}
	switch current {
	case "":
		out += in[pos:lcnt]
	case posGreen:
		out += g.Sprint(in[pos:lcnt])
	case posRed:
		out += r.Sprint(in[pos:lcnt])
	}

	s := strings.Repeat(" ", w)
	if c > 0 {
		g := color.New(color.FgHiGreen)
		g.EnableColor()
		s = g.Sprintf(fmt.Sprintf("%%%dd", w), c)
	}

	return s, out
}
