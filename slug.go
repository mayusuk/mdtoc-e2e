package mdtoc

import (
	"strconv"
	"strings"
	"unicode"
)

// emptySlug is the slug used for text that slugs to nothing (FR-2 AC4).
const emptySlug = "section"

// Slug returns the GitHub-style anchor slug for a heading's text (FR-2 AC1,
// AC2, AC4). It does not make the slug unique; use a Slugger for that.
func Slug(text string) string {
	text = stripMarkers(text)
	var b strings.Builder
	for _, r := range strings.ToLower(text) {
		switch {
		case r == ' ':
			b.WriteByte('-')
		case r == '-' || r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return emptySlug
	}
	return b.String()
}

// Slugger makes slugs unique within one document (FR-2 AC3). The zero value
// is ready to use. Use one Slugger per document.
type Slugger struct {
	used  map[string]bool
	count map[string]int
}

// Slug returns Slug(text), suffixed with -1, -2, ... when that slug has
// already been returned by this Slugger. A suffixed slug that is already
// taken, for example by an earlier literal heading, is skipped.
func (s *Slugger) Slug(text string) string {
	if s.used == nil {
		s.used = make(map[string]bool)
		s.count = make(map[string]int)
	}
	base := Slug(text)
	slug := base
	for s.used[slug] {
		s.count[base]++
		slug = base + "-" + strconv.Itoa(s.count[base])
	}
	s.used[slug] = true
	return slug
}

// stripMarkers removes inline code backticks and emphasis markers from text,
// keeping their content (FR-2 AC2). Code span content is kept as written.
// An underscore run is removed only when it opens or closes emphasis at a word
// boundary, so snake_case keeps its underscore (plan decision D1). Asterisks
// and unpaired backticks need no handling here: Slug drops them anyway.
func stripMarkers(text string) string {
	rs := []rune(text)
	var out []rune
	seg := 0 // start of the current segment outside code spans
	for i := 0; i < len(rs); {
		if rs[i] != '`' {
			i++
			continue
		}
		n := runLen(rs, i, '`')
		end := findCloser(rs, i+n, n)
		if end < 0 {
			i += n
			continue
		}
		out = append(out, stripUnderscores(rs[seg:i])...)
		out = append(out, rs[i+n:end]...)
		i = end + n
		seg = i
	}
	out = append(out, stripUnderscores(rs[seg:])...)
	return string(out)
}

// findCloser returns the index of the next run of exactly n backticks at or
// after from, or -1.
func findCloser(rs []rune, from, n int) int {
	for j := from; j < len(rs); {
		if rs[j] != '`' {
			j++
			continue
		}
		m := runLen(rs, j, '`')
		if m == n {
			return j
		}
		j += m
	}
	return -1
}

// stripUnderscores removes paired underscore emphasis runs from rs.
func stripUnderscores(rs []rune) []rune {
	type run struct{ start, n int }
	var openers []run
	drop := make([]bool, len(rs))
	for i := 0; i < len(rs); {
		if rs[i] != '_' {
			i++
			continue
		}
		n := runLen(rs, i, '_')
		var prev, next rune
		hasPrev, hasNext := i > 0, i+n < len(rs)
		if hasPrev {
			prev = rs[i-1]
		}
		if hasNext {
			next = rs[i+n]
		}
		opener := (!hasPrev || !isWord(prev)) && hasNext && !unicode.IsSpace(next)
		closer := hasPrev && !unicode.IsSpace(prev) && (!hasNext || !isWord(next))
		matched := false
		if closer {
			for k := len(openers) - 1; k >= 0; k-- {
				if openers[k].n == n {
					for j := 0; j < n; j++ {
						drop[openers[k].start+j] = true
						drop[i+j] = true
					}
					openers = openers[:k]
					matched = true
					break
				}
			}
		}
		if !matched && opener {
			openers = append(openers, run{i, n})
		}
		i += n
	}
	out := make([]rune, 0, len(rs))
	for i, r := range rs {
		if !drop[i] {
			out = append(out, r)
		}
	}
	return out
}

func runLen(rs []rune, i int, c rune) int {
	n := 0
	for i+n < len(rs) && rs[i+n] == c {
		n++
	}
	return n
}

func isWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
