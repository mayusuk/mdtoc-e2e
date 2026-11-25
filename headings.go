package mdtoc

import (
	"bytes"
	"strings"
)

// Heading is one heading found in a Markdown document.
type Heading struct {
	// Level is the heading level, 1 to 6.
	Level int
	// Text is the heading text with surrounding whitespace and any closing
	// `#` sequence removed. Inline Markdown is kept as written.
	Text string
	// Line is the 1-based line number of the heading text.
	Line int
}

// Headings returns the ATX and setext headings in src, in document order
// (FR-1). Lines inside fenced code blocks are ignored.
func Headings(src []byte) []Heading {
	var hs []Heading

	var (
		inFence   bool
		fenceChar byte
		fenceLen  int
		// candidate is a preceding text line that a setext underline
		// could turn into a heading.
		candidate     string
		candidateLine int
		hasCandidate  bool
	)

	lines := bytes.Split(src, []byte("\n"))
	for i, raw := range lines {
		lineNo := i + 1
		line := strings.TrimSuffix(string(raw), "\r")

		if inFence {
			if isClosingFence(line, fenceChar, fenceLen) {
				inFence = false
			}
			continue
		}

		if c, n, ok := openingFence(line); ok {
			inFence, fenceChar, fenceLen = true, c, n
			hasCandidate = false
			continue
		}

		if strings.TrimSpace(line) == "" {
			hasCandidate = false
			continue
		}

		if indent(line) >= 4 {
			// An indented line is never a heading, nor the text of a
			// setext heading (FR-1 AC4).
			hasCandidate = false
			continue
		}

		if level, ok := setextLevel(line); ok && hasCandidate {
			hs = append(hs, Heading{Level: level, Text: candidate, Line: candidateLine})
			hasCandidate = false
			continue
		}

		if level, text, ok := atx(line); ok {
			hs = append(hs, Heading{Level: level, Text: text, Line: lineNo})
			hasCandidate = false
			continue
		}

		candidate, candidateLine, hasCandidate = strings.TrimSpace(line), lineNo, true
	}
	return hs
}

// indent returns the width of the leading whitespace of line, counting a
// tab as four columns.
func indent(line string) int {
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

// openingFence reports whether line opens a fenced code block, and returns
// the fence character and length.
func openingFence(line string) (byte, int, bool) {
	if indent(line) >= 4 {
		return 0, 0, false
	}
	s := strings.TrimLeft(line, " ")
	if s == "" || (s[0] != '`' && s[0] != '~') {
		return 0, 0, false
	}
	c := s[0]
	n := runLen(s, c)
	if n < 3 {
		return 0, 0, false
	}
	if c == '`' && strings.IndexByte(s[n:], '`') >= 0 {
		// A backtick fence's info string cannot contain backticks.
		return 0, 0, false
	}
	return c, n, true
}

// isClosingFence reports whether line closes a fence of character c opened
// with n characters.
func isClosingFence(line string, c byte, n int) bool {
	if indent(line) >= 4 {
		return false
	}
	s := strings.TrimLeft(line, " ")
	m := runLen(s, c)
	return m >= n && strings.TrimSpace(s[m:]) == ""
}

// runLen returns the number of leading c characters in s.
func runLen(s string, c byte) int {
	n := 0
	for n < len(s) && s[n] == c {
		n++
	}
	return n
}

// setextLevel reports whether line is a setext underline: only `=` (level 1)
// or only `-` (level 2), with optional surrounding whitespace.
func setextLevel(line string) (int, bool) {
	s := strings.TrimSpace(line)
	if s == "" {
		return 0, false
	}
	switch c := s[0]; c {
	case '=', '-':
		if runLen(s, c) != len(s) {
			return 0, false
		}
		if c == '=' {
			return 1, true
		}
		return 2, true
	}
	return 0, false
}

// atx parses line as an ATX heading.
func atx(line string) (int, string, bool) {
	s := strings.TrimLeft(line, " ")
	n := runLen(s, '#')
	if n < 1 || n > 6 {
		return 0, "", false
	}
	rest := s[n:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return 0, "", false
	}
	text := strings.TrimSpace(rest)
	// Remove a closing sequence of `#`: either the whole text, or a run
	// preceded by whitespace.
	trimmed := strings.TrimRight(text, "#")
	if trimmed == "" {
		text = ""
	} else if len(trimmed) < len(text) {
		if last := trimmed[len(trimmed)-1]; last == ' ' || last == '\t' {
			text = strings.TrimSpace(trimmed)
		}
	}
	return n, text, true
}
