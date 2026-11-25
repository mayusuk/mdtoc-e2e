package mdtoc

import (
	"strconv"
	"strings"
)

// Options controls which headings Render includes and how it lists them.
type Options struct {
	// MinLevel and MaxLevel bound the heading levels included. Zero means
	// 1 and 6 respectively.
	MinLevel, MaxLevel int
	// Ordered numbers list items per nesting level instead of using `-`.
	Ordered bool
}

// Render returns a Markdown list linking to each heading in hs whose level is
// within opt's range (FR-3). Every heading, in range or not, is slugged with
// one Slugger, so the links match the anchors GitHub generates. Render
// returns "" for empty output or an invalid range.
func Render(hs []Heading, opt Options) string {
	lo, hi := opt.MinLevel, opt.MaxLevel
	if lo == 0 {
		lo = 1
	}
	if hi == 0 {
		hi = 6
	}
	if lo < 1 || hi > 6 || lo > hi {
		return ""
	}

	type item struct {
		level      int
		text, slug string
	}
	var items []item
	var slugger Slugger
	shallowest := hi
	for _, h := range hs {
		// The slug is made from the text with links reduced (plan decision
		// D2); Slug removes the remaining inline markers itself.
		slug := slugger.Slug(reduceLinks(h.Text))
		if h.Level < lo || h.Level > hi {
			continue
		}
		items = append(items, item{h.Level, stripInline(h.Text), slug})
		shallowest = min(shallowest, h.Level)
	}

	var b strings.Builder
	var counters [6]int // next number per nesting depth, for Ordered
	for _, it := range items {
		depth := it.level - shallowest
		counters[depth]++
		for d := depth + 1; d < len(counters); d++ {
			counters[d] = 0
		}
		b.WriteString(strings.Repeat("  ", depth))
		if opt.Ordered {
			b.WriteString(strconv.Itoa(counters[depth]))
			b.WriteByte('.')
		} else {
			b.WriteByte('-')
		}
		b.WriteString(" [")
		b.WriteString(it.text)
		b.WriteString("](#")
		b.WriteString(it.slug)
		b.WriteString(")\n")
	}
	return b.String()
}

// stripInline returns text with inline Markdown removed for use as link text
// (FR-3 AC3): links become their text, and code backticks and emphasis
// markers are removed with their content kept.
func stripInline(text string) string {
	return stripMarkers(reduceLinks(text))
}

// reduceLinks replaces each inline link `[text](destination)` in text with
// its text. Brackets inside code spans are not links.
func reduceLinks(text string) string {
	rs := []rune(text)
	code, _ := codeSpans(rs)
	out := make([]rune, 0, len(rs))
	for i := 0; i < len(rs); i++ {
		if rs[i] == '[' && !code[i] {
			if end := matching(rs, code, i, '[', ']'); end >= 0 && end+1 < len(rs) && rs[end+1] == '(' {
				if close := matching(rs, code, end+1, '(', ')'); close >= 0 {
					out = append(out, []rune(reduceLinks(string(rs[i+1:end])))...)
					i = close
					continue
				}
			}
		}
		out = append(out, rs[i])
	}
	return string(out)
}

// matching returns the index of the close rune balancing the open rune at
// rs[start], skipping code spans, or -1.
func matching(rs []rune, code []bool, start int, open, close rune) int {
	depth := 0
	for j := start; j < len(rs); j++ {
		if code[j] {
			continue
		}
		switch rs[j] {
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return j
			}
		}
	}
	return -1
}
