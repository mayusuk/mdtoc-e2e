# mdtoc specification

`mdtoc` generates a table of contents (TOC) for a Markdown document. It is a Go library (`github.com/mayusuk/mdtoc-e2e`, package `mdtoc`) and a command (`cmd/mdtoc`). Standard library only.

This spec is the source of truth. Requirement IDs (FR-n ACm) are used in test names, for example `TestFR2AC3_DuplicateSlugsGetSuffix`.

## FR-1 Heading extraction

`func Headings(src []byte) []Heading`, where `type Heading struct { Level int; Text string; Line int }` (`Line` is 1-based).

- **AC1:** ATX headings (`#` to `######` followed by a space or end of line) are returned in document order with their level and text. Leading and trailing whitespace and a closing sequence of `#` characters are removed from the text (`## Title ##` → `Title`).
- **AC2:** Setext headings are recognised: a line of text followed by a line of only `=` (level 1) or only `-` (level 2), at least one character long.
- **AC3:** Nothing inside a fenced code block (opened by three or more backticks or tildes, closed by a fence of the same character at least as long) is a heading.
- **AC4:** A line indented by four or more spaces is not a heading. `#hashtag` (no space after the `#` run) is not a heading. Seven or more `#` is not a heading.
- **AC5:** Inline Markdown in the text is kept as written; it is stripped only when rendering (FR-3 AC3).

## FR-2 Slugs

`func Slug(text string) string` and `type Slugger struct` with `func (s *Slugger) Slug(text string) string`.

- **AC1:** A slug is the text lower-cased (Unicode-aware), with every character that is not a letter, a digit, a space, `-` or `_` removed, then each space replaced with `-`. (`Hello, World!` → `hello-world`, `Go 1.25 notes` → `go-125-notes`.)
- **AC2:** Inline code backticks and emphasis markers (`*`, `_` pairs) are removed before slugging, but their content is kept.
- **AC3:** A `Slugger` returns unique slugs within one document: the second occurrence of a slug gets `-1`, the third `-2`, and so on (`intro`, `intro-1`, `intro-2`). A generated suffixed slug that collides with a later literal heading is still made unique.
- **AC4:** An empty slug (for example a heading of only punctuation) is returned as `section`, and is made unique like any other.

## FR-3 Rendering

`func Render(hs []Heading, opt Options) string`, with `type Options struct { MinLevel, MaxLevel int; Ordered bool }`. Zero values mean `MinLevel = 1`, `MaxLevel = 6`.

- **AC1:** Each heading in range becomes one list line `- [Text](#slug)`, indented by two spaces per level below the shallowest level present in range. Output ends with a newline; empty input renders as the empty string.
- **AC2:** Headings outside `MinLevel..MaxLevel` are skipped, but still take part in slug uniqueness, so links match the anchors GitHub generates.
- **AC3:** Link text has inline Markdown stripped (backticks, emphasis, and links rendered as their text: `[a](b)` → `a`).
- **AC4:** With `Ordered`, list markers are `1.` numbered per nesting level instead of `-`.
- **AC5:** `MinLevel > MaxLevel`, or either outside 1..6, makes `Render` return an empty string; the CLI reports it as an error (FR-4 AC4).

## FR-4 Command

`mdtoc [flags] [file]`

- **AC1:** Reads the file, or standard input when no file or `-` is given, and prints the TOC to standard output.
- **AC2:** Flags `--min N`, `--max N`, `--ordered` map to `Options`.
- **AC3:** `--in-place` (requires a file) replaces the text between the lines `<!-- toc -->` and `<!-- /toc -->` with the TOC, keeping the marker lines, and writes the file only if the content changed. Headings inside the markers are ignored. Missing markers is an error.
- **AC4:** Errors go to standard error with the prefix `mdtoc: `, and the exit code is 1 for a usage or input error, 0 on success.
- **AC5:** `--check` (with `--in-place` semantics, but never writing) exits 1 and prints `mdtoc: TOC is out of date` when the file would change, 0 otherwise.
