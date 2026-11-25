package mdtoc

import (
	"reflect"
	"testing"
)

func assertHeadings(t *testing.T, src string, want []Heading) {
	t.Helper()
	got := Headings([]byte(src))
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Headings(%q)\n got %+v\nwant %+v", src, got, want)
	}
}

func TestFR1AC1_ATXHeadingsInOrder(t *testing.T) {
	src := "# One\n\ntext\n## Two\n### Three\n#### Four\n##### Five\n###### Six\n"
	assertHeadings(t, src, []Heading{
		{1, "One", 1}, {2, "Two", 4}, {3, "Three", 5},
		{4, "Four", 6}, {5, "Five", 7}, {6, "Six", 8},
	})
}

func TestFR1AC1_ATXTextTrimmedAndClosingSequenceRemoved(t *testing.T) {
	cases := []struct {
		src  string
		want Heading
	}{
		{"## Title ##", Heading{2, "Title", 1}},
		{"#   Spaced   ", Heading{1, "Spaced", 1}},
		{"# Title #####   ", Heading{1, "Title", 1}},
		{"# Title#", Heading{1, "Title#", 1}},
		{"# C#", Heading{1, "C#", 1}},
		{"#\tTab", Heading{1, "Tab", 1}},
		{"   # Three spaces", Heading{1, "Three spaces", 1}},
		{"#", Heading{1, "", 1}},
		{"## ##", Heading{2, "", 1}},
		{"# Windows\r", Heading{1, "Windows", 1}},
	}
	for _, c := range cases {
		assertHeadings(t, c.src, []Heading{c.want})
	}
}

func TestFR1AC2_SetextHeadings(t *testing.T) {
	src := "Title\n=====\n\nSub *title*\n---\n\nOne\n=\n\nTwo\n-\n\nSpaced  \n  ===  \n"
	assertHeadings(t, src, []Heading{
		{1, "Title", 1}, {2, "Sub *title*", 4}, {1, "One", 7}, {2, "Two", 10}, {1, "Spaced", 13},
	})
}

func TestFR1AC2_UnderlineWithoutTextIsNotHeading(t *testing.T) {
	assertHeadings(t, "---\n\n===\n", nil)
	assertHeadings(t, "para\n\n---\n", nil)
	assertHeadings(t, "# ATX\n---\n", []Heading{{1, "ATX", 1}})
	assertHeadings(t, "Title\n=-=\n", nil)
}

func TestFR1AC2_MixedWithATX(t *testing.T) {
	assertHeadings(t, "# A\nB\n-\n## C\n", []Heading{{1, "A", 1}, {2, "B", 2}, {2, "C", 4}})
}

func TestFR1AC3_FencedCodeBlocksIgnored(t *testing.T) {
	src := "# Before\n```\n# not\nText\n===\n```\n~~~go\n## not\n~~~\n# After\n"
	assertHeadings(t, src, []Heading{{1, "Before", 1}, {1, "After", 10}})
}

func TestFR1AC3_FenceClosedOnlyBySameCharAtLeastAsLong(t *testing.T) {
	src := "````\n```\n# inside\n~~~~\n# inside\n````\n# out\n" +
		"~~~~~\n~~~\n# inside\n~~~~~~\n# out2\n"
	assertHeadings(t, src, []Heading{{1, "out", 7}, {1, "out2", 12}})
}

func TestFR1AC3_UnclosedFenceRunsToEnd(t *testing.T) {
	assertHeadings(t, "# A\n```\n# B\n", []Heading{{1, "A", 1}})
}

func TestFR1AC3_TwoCharactersIsNotFence(t *testing.T) {
	assertHeadings(t, "``\n# A\n``\n", []Heading{{1, "A", 2}})
}

func TestFR1AC3_FenceLineDoesNotBecomeSetextText(t *testing.T) {
	assertHeadings(t, "```\ncode\n```\n---\n", nil)
}

func TestFR1AC4_IndentedLineIsNotHeading(t *testing.T) {
	assertHeadings(t, "    # Code\n\t# Tab\n", nil)
	assertHeadings(t, "    Text\n====\n", nil)
	assertHeadings(t, "Text\n    ===\n", nil)
}

func TestFR1AC4_HashtagIsNotHeading(t *testing.T) {
	assertHeadings(t, "#hashtag\n##nope\n", nil)
}

func TestFR1AC4_SevenHashesIsNotHeading(t *testing.T) {
	assertHeadings(t, "####### Seven\n", nil)
}

func TestFR1AC5_InlineMarkdownKept(t *testing.T) {
	src := "# The `code` and *em* and [link](http://x)\nSetext __bold__\n===\n"
	assertHeadings(t, src, []Heading{
		{1, "The `code` and *em* and [link](http://x)", 1},
		{1, "Setext __bold__", 2},
	})
}

func TestFR1_EmptyInput(t *testing.T) {
	assertHeadings(t, "", nil)
}
