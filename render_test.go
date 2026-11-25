package mdtoc

import "testing"

func h(level int, text string) Heading { return Heading{Level: level, Text: text} }

func checkRender(t *testing.T, hs []Heading, opt Options, want string) {
	t.Helper()
	if got := Render(hs, opt); got != want {
		t.Errorf("Render(%v, %+v) =\n%q\nwant\n%q", hs, opt, got, want)
	}
}

func TestFR3AC1_NestedListIndentedFromShallowestLevel(t *testing.T) {
	hs := []Heading{h(2, "Install"), h(3, "From source"), h(4, "Deep"), h(2, "Usage")}
	checkRender(t, hs, Options{}, ""+
		"- [Install](#install)\n"+
		"  - [From source](#from-source)\n"+
		"    - [Deep](#deep)\n"+
		"- [Usage](#usage)\n")
}

func TestFR3AC1_ShallowestLevelNeedNotComeFirst(t *testing.T) {
	hs := []Heading{h(3, "Intro"), h(2, "Top"), h(3, "Child")}
	checkRender(t, hs, Options{}, ""+
		"  - [Intro](#intro)\n"+
		"- [Top](#top)\n"+
		"  - [Child](#child)\n")
}

func TestFR3AC1_EmptyInputRendersEmptyString(t *testing.T) {
	checkRender(t, nil, Options{}, "")
	checkRender(t, []Heading{}, Options{Ordered: true}, "")
}

func TestFR3AC1_UsesFullHeadingsOutput(t *testing.T) {
	src := []byte("# Title\n\nSub\n---\n\n```\n# not\n```\n## Title\n")
	checkRender(t, Headings(src), Options{}, ""+
		"- [Title](#title)\n"+
		"  - [Sub](#sub)\n"+
		"  - [Title](#title-1)\n")
}

func TestFR3AC2_OutOfRangeSkippedButReserveSlugs(t *testing.T) {
	hs := []Heading{h(1, "Notes"), h(2, "Notes"), h(3, "Notes"), h(2, "Other"), h(4, "Notes")}
	checkRender(t, hs, Options{MinLevel: 2, MaxLevel: 3}, ""+
		"- [Notes](#notes-1)\n"+
		"  - [Notes](#notes-2)\n"+
		"- [Other](#other)\n")
}

func TestFR3AC2_DefaultsAppliedPerField(t *testing.T) {
	hs := []Heading{h(1, "A"), h(2, "B"), h(3, "C")}
	checkRender(t, hs, Options{MaxLevel: 2}, "- [A](#a)\n  - [B](#b)\n")
	checkRender(t, hs, Options{MinLevel: 2}, "- [B](#b)\n  - [C](#c)\n")
}

func TestFR3AC2_NoHeadingsInRange(t *testing.T) {
	checkRender(t, []Heading{h(1, "A")}, Options{MinLevel: 2}, "")
}

func TestFR3AC3_LinkTextStripsInlineMarkdown(t *testing.T) {
	cases := []struct{ in, wantText, wantSlug string }{
		{"Use `go test`", "Use go test", "use-go-test"},
		{"*emphasis* and **strong**", "emphasis and strong", "emphasis-and-strong"},
		{"_emphasis_ and __strong__", "emphasis and strong", "emphasis-and-strong"},
		{"See [the docs](https://example.com/a_b)", "See the docs", "see-the-docs"},
		{"[**Bold** link](x) end", "Bold link end", "bold-link-end"},
		{"[`code` link](x)", "code link", "code-link"},
		{"[a [b] c](x)", "a [b] c", "a-b-c"},
		{"[a](b(c)) d", "a d", "a-d"},
		{"`[a](b)` in code", "[a](b) in code", "ab-in-code"},
		{"`_x_` and `*y*`", "_x_ and *y*", "_x_-and-y"},
		{"snake_case and 2 * 3", "snake_case and 2 * 3", "snake_case-and-2--3"},
		{"[not a link] (x)", "[not a link] (x)", "not-a-link-x"},
		{"[unclosed](x", "[unclosed](x", "unclosedx"},
	}
	for _, c := range cases {
		want := "- [" + c.wantText + "](#" + c.wantSlug + ")\n"
		checkRender(t, []Heading{h(1, c.in)}, Options{}, want)
	}
}

func TestFR3AC3_SlugUsesStrippedLinkText(t *testing.T) {
	// D2: the link destination never reaches the slug.
	hs := []Heading{h(1, "[Intro](https://example.com)"), h(1, "Intro")}
	checkRender(t, hs, Options{}, "- [Intro](#intro)\n- [Intro](#intro-1)\n")
}

func TestFR3AC4_OrderedNumbersPerNestingLevel(t *testing.T) {
	hs := []Heading{
		h(1, "A"), h(2, "A1"), h(2, "A2"), h(3, "A2x"),
		h(1, "B"), h(2, "B1"), h(3, "B1x"), h(3, "B1y"), h(2, "B2"),
	}
	checkRender(t, hs, Options{Ordered: true}, ""+
		"1. [A](#a)\n"+
		"  1. [A1](#a1)\n"+
		"  2. [A2](#a2)\n"+
		"    1. [A2x](#a2x)\n"+
		"2. [B](#b)\n"+
		"  1. [B1](#b1)\n"+
		"    1. [B1x](#b1x)\n"+
		"    2. [B1y](#b1y)\n"+
		"  2. [B2](#b2)\n")
}

func TestFR3AC4_OrderedRestartsAfterSkippedDepth(t *testing.T) {
	hs := []Heading{h(1, "A"), h(3, "Deep"), h(2, "Mid"), h(3, "Deep2")}
	checkRender(t, hs, Options{Ordered: true}, ""+
		"1. [A](#a)\n"+
		"    1. [Deep](#deep)\n"+
		"  1. [Mid](#mid)\n"+
		"    1. [Deep2](#deep2)\n")
}

func TestFR3AC5_InvalidRangeRendersEmpty(t *testing.T) {
	hs := []Heading{h(1, "A"), h(2, "B")}
	for _, opt := range []Options{
		{MinLevel: 3, MaxLevel: 2},
		{MinLevel: 7},
		{MaxLevel: 7},
		{MinLevel: -1},
		{MaxLevel: -1},
		{MinLevel: 0, MaxLevel: 7},
		{MinLevel: 4, Ordered: true, MaxLevel: 1},
	} {
		checkRender(t, hs, opt, "")
	}
}
