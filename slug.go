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
// boundary, so snake_case keeps its underscore (plan decision D1). Emphasis
// may enclose a code span, so underscores are paired in one pass over the
// whole text, with code spans opaque. Asterisks and unpaired backticks need
// no handling here: Slug drops them anyway.
func stripMarkers(text string) string {
	rs := []rune(text)
	code := make([]bool, len(rs)) // inside a code span, backticks included
	drop := make([]bool, len(rs))
	for i := 0; i < len(rs); {
		if rs[i] != '`' {
			i++
			continue
		}
		n := runeRunLen(rs, i, '`')
		end := findCloser(rs, i+n, n)
		if end < 0 {
			i += n
			continue
		}
		for j := i; j < end+n; j++ {
			code[j] = true
		}
		for j := 0; j < n; j++ {
			drop[i+j] = true
			drop[end+j] = true
		}
		i = end + n
	}
	markUnderscores(rs, code, drop)
	out := make([]rune, 0, len(rs))
	for i, r := range rs {
		if !drop[i] {
			out = append(out, r)
		}
	}
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
		m := runeRunLen(rs, j, '`')
		if m == n {
			return j
		}
		j += m
	}
	return -1
}

// markUnderscores sets drop for paired underscore emphasis runs in rs,
// ignoring underscores inside code spans. Neighbours are read from rs as
// written, so a code span's backtick counts as a non-word boundary.
func markUnderscores(rs []rune, code, drop []bool) {
	type run struct{ start, n int }
	var openers []run
	for i := 0; i < len(rs); {
		if rs[i] != '_' || code[i] {
			i++
			continue
		}
		n := runeRunLen(rs, i, '_')
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
}

// runeRunLen returns the length of the run of c starting at rs[i].
func runeRunLen(rs []rune, i int, c rune) int {
	n := 0
	for i+n < len(rs) && rs[i+n] == c {
		n++
	}
	return n
}

func isWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }
