// Command mdtoc prints a table of contents for a Markdown file.
//
// Usage:
//
//	mdtoc [flags] [file]
//
// With no file, or with `-`, mdtoc reads standard input. See specs/SPEC.md
// (FR-4) for the full behaviour.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	mdtoc "github.com/mayusuk/mdtoc-e2e"
)

const (
	beginMarker = "<!-- toc -->"
	endMarker   = "<!-- /toc -->"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

// errOutOfDate is reported by --check when the file would change (FR-4 AC5).
var errOutOfDate = errors.New("TOC is out of date")

// run executes mdtoc with args (excluding the program name) and returns the
// exit code: 0 on success, 1 on a usage or input error (FR-4 AC4).
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("mdtoc", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	minLevel := fs.Int("min", 1, "shallowest heading `level` to include (1-6)")
	maxLevel := fs.Int("max", 6, "deepest heading `level` to include (1-6)")
	ordered := fs.Bool("ordered", false, "number list items instead of using -")
	inPlace := fs.Bool("in-place", false, "replace the TOC between <!-- toc --> and <!-- /toc --> in file")
	check := fs.Bool("check", false, "like --in-place, but only report whether the file is out of date")

	fail := func(err error) int {
		fmt.Fprintf(stderr, "mdtoc: %v\n", err)
		return 1
	}
	usage := func() {
		fmt.Fprintln(stderr, "usage: mdtoc [flags] [file]")
		fs.SetOutput(stderr)
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			usage()
			return 0
		}
		return fail(err)
	}
	if fs.NArg() > 1 {
		return fail(fmt.Errorf("too many arguments: %q", fs.Args()))
	}
	file := fs.Arg(0)
	if file == "-" {
		file = ""
	}

	if *minLevel < 1 || *minLevel > 6 || *maxLevel < 1 || *maxLevel > 6 {
		return fail(fmt.Errorf("--min and --max must be between 1 and 6"))
	}
	if *minLevel > *maxLevel {
		return fail(fmt.Errorf("--min %d is greater than --max %d", *minLevel, *maxLevel))
	}
	opt := mdtoc.Options{MinLevel: *minLevel, MaxLevel: *maxLevel, Ordered: *ordered}

	if *inPlace || *check {
		if file == "" {
			flagName := "--in-place"
			if !*inPlace {
				flagName = "--check"
			}
			return fail(fmt.Errorf("%s requires a file", flagName))
		}
		if err := updateFile(file, opt, *check); err != nil {
			return fail(err)
		}
		return 0
	}

	var src []byte
	var err error
	if file == "" {
		src, err = io.ReadAll(stdin)
	} else {
		src, err = os.ReadFile(file)
	}
	if err != nil {
		return fail(err)
	}
	if _, err := io.WriteString(stdout, mdtoc.Render(mdtoc.Headings(src), opt)); err != nil {
		return fail(err)
	}
	return 0
}

// updateFile regenerates the TOC between the markers in file (FR-4 AC3). It
// writes the file only when the content changes. With check, it never writes
// and returns errOutOfDate when the file would change (FR-4 AC5).
func updateFile(file string, opt mdtoc.Options, check bool) error {
	src, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	out, err := replaceTOC(src, opt)
	if err != nil {
		return fmt.Errorf("%s: %w", file, err)
	}
	if bytes.Equal(out, src) {
		return nil
	}
	if check {
		return errOutOfDate
	}
	info, err := os.Stat(file)
	if err != nil {
		return err
	}
	return os.WriteFile(file, out, info.Mode().Perm())
}

// replaceTOC returns src with the lines between the TOC markers replaced by
// the TOC of src's headings outside the markers. The marker lines are kept.
func replaceTOC(src []byte, opt mdtoc.Options) ([]byte, error) {
	m, err := findMarkers(src)
	if err != nil {
		return nil, err
	}

	var hs []mdtoc.Heading
	for _, h := range mdtoc.Headings(src) {
		if h.Line > m.beginLine && h.Line < m.endLine {
			continue // inside the markers: the old TOC, not the document
		}
		hs = append(hs, h)
	}

	out := make([]byte, 0, len(src))
	out = append(out, src[:m.afterBegin]...)
	out = append(out, mdtoc.Render(hs, opt)...)
	out = append(out, src[m.endStart:]...)
	return out, nil
}

// markers locates the TOC marker lines in a document.
type markers struct {
	// beginLine and endLine are the 1-based line numbers of the markers.
	beginLine, endLine int
	// afterBegin is the byte offset just past the begin marker's line
	// ending; endStart is the byte offset where the end marker's line
	// starts.
	afterBegin, endStart int
}

// findMarkers finds exactly one `<!-- toc -->` line followed by one
// `<!-- /toc -->` line. Marker lines inside fenced code blocks are not
// markers (plan decision D3).
func findMarkers(src []byte) (markers, error) {
	var m markers
	var fence fenceTracker
	offset := 0
	for i, raw := range bytes.SplitAfter(src, []byte("\n")) {
		lineNo := i + 1
		start := offset
		offset += len(raw)
		line := strings.TrimRight(string(raw), "\r\n")
		if fence.inFence(line) {
			continue
		}
		switch strings.TrimSpace(line) {
		case beginMarker:
			if m.beginLine != 0 {
				return markers{}, fmt.Errorf("more than one %s marker (line %d)", beginMarker, lineNo)
			}
			if !strings.HasSuffix(string(raw), "\n") {
				return markers{}, fmt.Errorf("missing %s marker after line %d", endMarker, lineNo)
			}
			m.beginLine, m.afterBegin = lineNo, offset
		case endMarker:
			if m.beginLine == 0 {
				return markers{}, fmt.Errorf("%s marker on line %d has no %s before it", endMarker, lineNo, beginMarker)
			}
			if m.endLine != 0 {
				return markers{}, fmt.Errorf("more than one %s marker (line %d)", endMarker, lineNo)
			}
			m.endLine, m.endStart = lineNo, start
		}
	}
	switch {
	case m.beginLine == 0:
		return markers{}, fmt.Errorf("missing %s marker", beginMarker)
	case m.endLine == 0:
		return markers{}, fmt.Errorf("missing %s marker after line %d", endMarker, m.beginLine)
	}
	return m, nil
}

// fenceTracker follows fenced code blocks line by line, using the same rules
// as heading extraction (FR-1 AC3).
type fenceTracker struct {
	open bool
	char byte
	n    int
}

// inFence reports whether line is part of a fenced code block, including
// its opening and closing fence lines, and updates the tracker.
func (f *fenceTracker) inFence(line string) bool {
	if leadingIndent(line) >= 4 {
		return f.open
	}
	s := strings.TrimLeft(line, " ")
	if f.open {
		if n := runLen(s, f.char); n >= f.n && strings.TrimSpace(s[n:]) == "" {
			f.open = false
		}
		return true
	}
	if s == "" || (s[0] != '`' && s[0] != '~') {
		return false
	}
	c := s[0]
	n := runLen(s, c)
	if n < 3 || (c == '`' && strings.IndexByte(s[n:], '`') >= 0) {
		return false
	}
	f.open, f.char, f.n = true, c, n
	return true
}

// leadingIndent returns the width of the leading whitespace of line,
// counting a tab as four columns.
func leadingIndent(line string) int {
	n := 0
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case ' ':
			n++
		case '\t':
			n += 4 - n%4
		default:
			return n
		}
	}
	return n
}

// runLen returns the number of leading c characters in s.
func runLen(s string, c byte) int {
	n := 0
	for n < len(s) && s[n] == c {
		n++
	}
	return n
}
