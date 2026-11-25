package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const sampleDoc = "# Title\n\n## Intro\n\n### Detail\n\n## Usage\n"

// runMdtoc runs the command with args and stdin and returns its exit code,
// standard output and standard error.
func runMdtoc(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// writeFile writes content to a new file in a temporary directory and
// returns its path.
func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "doc.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

const sampleTOC = "- [Title](#title)\n  - [Intro](#intro)\n    - [Detail](#detail)\n  - [Usage](#usage)\n"

func TestFR4AC1_ReadsFile(t *testing.T) {
	path := writeFile(t, sampleDoc)
	code, out, errOut := runMdtoc(t, "", path)
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if out != sampleTOC {
		t.Errorf("stdout = %q, want %q", out, sampleTOC)
	}
}

func TestFR4AC1_ReadsStdinWithoutFile(t *testing.T) {
	code, out, errOut := runMdtoc(t, sampleDoc)
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if out != sampleTOC {
		t.Errorf("stdout = %q, want %q", out, sampleTOC)
	}
}

func TestFR4AC1_ReadsStdinWithDash(t *testing.T) {
	code, out, errOut := runMdtoc(t, sampleDoc, "-")
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if out != sampleTOC {
		t.Errorf("stdout = %q, want %q", out, sampleTOC)
	}
}

func TestFR4AC1_NoHeadingsPrintsNothing(t *testing.T) {
	code, out, errOut := runMdtoc(t, "just text\n")
	if code != 0 || out != "" || errOut != "" {
		t.Errorf("exit %d, stdout %q, stderr %q; want 0, empty, empty", code, out, errOut)
	}
}

func TestFR4AC2_FlagsMapToOptions(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{"min", []string{"--min", "2"}, "- [Intro](#intro)\n  - [Detail](#detail)\n- [Usage](#usage)\n"},
		{"max", []string{"--max", "2"}, "- [Title](#title)\n  - [Intro](#intro)\n  - [Usage](#usage)\n"},
		{"min and max", []string{"--min", "2", "--max", "2"}, "- [Intro](#intro)\n- [Usage](#usage)\n"},
		{"ordered", []string{"--ordered"}, "1. [Title](#title)\n  1. [Intro](#intro)\n    1. [Detail](#detail)\n  2. [Usage](#usage)\n"},
		{"single dash", []string{"-min=3"}, "- [Detail](#detail)\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errOut := runMdtoc(t, sampleDoc, tt.args...)
			if code != 0 || errOut != "" {
				t.Fatalf("exit %d, stderr %q", code, errOut)
			}
			if out != tt.want {
				t.Errorf("stdout = %q, want %q", out, tt.want)
			}
		})
	}
}

func TestFR4AC3_InPlaceReplacesBetweenMarkers(t *testing.T) {
	doc := "# Title\n\n<!-- toc -->\n- stale entry\n<!-- /toc -->\n\n## Intro\n\n## Usage\n"
	want := "# Title\n\n<!-- toc -->\n- [Title](#title)\n  - [Intro](#intro)\n  - [Usage](#usage)\n<!-- /toc -->\n\n## Intro\n\n## Usage\n"
	path := writeFile(t, doc)
	code, out, errOut := runMdtoc(t, "", "--in-place", path)
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if out != "" {
		t.Errorf("stdout = %q, want empty", out)
	}
	if got := readFile(t, path); got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestFR4AC3_InPlaceFillsEmptyMarkersAndUsesFlags(t *testing.T) {
	doc := "<!-- toc -->\n<!-- /toc -->\n# Title\n## Intro\n### Detail\n"
	want := "<!-- toc -->\n1. [Intro](#intro)\n<!-- /toc -->\n# Title\n## Intro\n### Detail\n"
	path := writeFile(t, doc)
	code, _, errOut := runMdtoc(t, "", "--in-place", "--min", "2", "--max", "2", "--ordered", path)
	if code != 0 || errOut != "" {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if got := readFile(t, path); got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestFR4AC3_InPlaceIgnoresHeadingsInsideMarkers(t *testing.T) {
	doc := "<!-- toc -->\n# Old heading\n<!-- /toc -->\n\n# Real\n"
	want := "<!-- toc -->\n- [Real](#real)\n<!-- /toc -->\n\n# Real\n"
	path := writeFile(t, doc)
	if code, _, errOut := runMdtoc(t, "", "--in-place", path); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if got := readFile(t, path); got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestFR4AC3_InPlaceKeepsCRLFOutsideTOC(t *testing.T) {
	doc := "# A\r\n<!-- toc -->\r\n<!-- /toc -->\r\n"
	want := "# A\r\n<!-- toc -->\r\n- [A](#a)\n<!-- /toc -->\r\n"
	path := writeFile(t, doc)
	if code, _, errOut := runMdtoc(t, "", "--in-place", path); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if got := readFile(t, path); got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestFR4AC3_InPlaceDoesNotRewriteUnchangedFile(t *testing.T) {
	doc := "<!-- toc -->\n- [A](#a)\n<!-- /toc -->\n# A\n"
	path := writeFile(t, doc)
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runMdtoc(t, "", "--in-place", path); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.ModTime().Equal(old) {
		t.Errorf("file was rewritten: mtime %v, want %v", info.ModTime(), old)
	}
	if got := readFile(t, path); got != doc {
		t.Errorf("file = %q, want %q", got, doc)
	}
}

func TestFR4AC3_InPlaceMarkerErrors(t *testing.T) {
	tests := []struct {
		name, doc, wantErr string
	}{
		{"no markers", "# A\n", "missing <!-- toc --> marker"},
		{"no end marker", "<!-- toc -->\n# A\n", "missing <!-- /toc --> marker"},
		{"no begin marker", "<!-- /toc -->\n# A\n", "has no <!-- toc --> before it"},
		{"end before begin", "<!-- /toc -->\n<!-- toc -->\n# A\n", "has no <!-- toc --> before it"},
		{"second begin", "<!-- toc -->\n<!-- /toc -->\n<!-- toc -->\n<!-- /toc -->\n", "more than one <!-- toc --> marker"},
		{"second end", "<!-- toc -->\n<!-- /toc -->\n<!-- /toc -->\n", "more than one <!-- /toc --> marker"},
		{"markers only in code fence (D3)", "# A\n```\n<!-- toc -->\n<!-- /toc -->\n```\n", "missing <!-- toc --> marker"},
		{"end marker only in tilde fence (D3)", "<!-- toc -->\n~~~~\n<!-- /toc -->\n~~~~\n", "missing <!-- /toc --> marker"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFile(t, tt.doc)
			code, out, errOut := runMdtoc(t, "", "--in-place", path)
			if code != 1 {
				t.Errorf("exit %d, want 1", code)
			}
			if out != "" {
				t.Errorf("stdout = %q, want empty", out)
			}
			if !strings.HasPrefix(errOut, "mdtoc: ") || !strings.Contains(errOut, tt.wantErr) {
				t.Errorf("stderr = %q, want prefix %q and %q", errOut, "mdtoc: ", tt.wantErr)
			}
			if got := readFile(t, path); got != tt.doc {
				t.Errorf("file changed to %q", got)
			}
		})
	}
}

func TestFR4AC3_InPlaceSkipsMarkersInFenceAndUsesRealOnes(t *testing.T) {
	doc := "```md\n<!-- toc -->\n<!-- /toc -->\n```\n<!-- toc -->\n<!-- /toc -->\n# A\n"
	want := "```md\n<!-- toc -->\n<!-- /toc -->\n```\n<!-- toc -->\n- [A](#a)\n<!-- /toc -->\n# A\n"
	path := writeFile(t, doc)
	if code, _, errOut := runMdtoc(t, "", "--in-place", path); code != 0 {
		t.Fatalf("exit %d, stderr %q", code, errOut)
	}
	if got := readFile(t, path); got != want {
		t.Errorf("file = %q, want %q", got, want)
	}
}

func TestFR4AC3_InPlaceRequiresFile(t *testing.T) {
	for _, args := range [][]string{{"--in-place"}, {"--in-place", "-"}} {
		code, out, errOut := runMdtoc(t, "<!-- toc -->\n<!-- /toc -->\n# A\n", args...)
		if code != 1 || out != "" || !strings.HasPrefix(errOut, "mdtoc: ") {
			t.Errorf("%q: exit %d, stdout %q, stderr %q; want 1, empty, mdtoc: error", args, code, out, errOut)
		}
	}
}

func TestFR4AC4_ErrorsArePrefixedAndExitOne(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing.md")
	tests := []struct {
		name string
		args []string
	}{
		{"missing file", []string{missing}},
		{"unknown flag", []string{"--bogus"}},
		{"bad flag value", []string{"--min", "x"}},
		{"too many arguments", []string{"a.md", "b.md"}},
		{"min above max", []string{"--min", "3", "--max", "2"}},
		{"min below 1", []string{"--min", "0"}},
		{"max above 6", []string{"--max", "7"}},
		{"check without file", []string{"--check"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, out, errOut := runMdtoc(t, sampleDoc, tt.args...)
			if code != 1 {
				t.Errorf("exit %d, want 1", code)
			}
			if out != "" {
				t.Errorf("stdout = %q, want empty", out)
			}
			if !strings.HasPrefix(errOut, "mdtoc: ") || strings.Count(errOut, "\n") != 1 {
				t.Errorf("stderr = %q, want one line prefixed %q", errOut, "mdtoc: ")
			}
		})
	}
}

func TestFR4AC4_SuccessExitsZero(t *testing.T) {
	if code, _, errOut := runMdtoc(t, sampleDoc, "--min", "1", "--max", "6"); code != 0 || errOut != "" {
		t.Errorf("exit %d, stderr %q; want 0, empty", code, errOut)
	}
}

func TestFR4AC5_CheckReportsStaleFileWithoutWriting(t *testing.T) {
	doc := "<!-- toc -->\n- stale\n<!-- /toc -->\n# A\n"
	path := writeFile(t, doc)
	for _, args := range [][]string{{"--check", path}, {"--in-place", "--check", path}} {
		code, out, errOut := runMdtoc(t, "", args...)
		if code != 1 {
			t.Errorf("%q: exit %d, want 1", args, code)
		}
		if out != "" {
			t.Errorf("%q: stdout = %q, want empty", args, out)
		}
		if errOut != "mdtoc: TOC is out of date\n" {
			t.Errorf("%q: stderr = %q, want %q", args, errOut, "mdtoc: TOC is out of date\n")
		}
		if got := readFile(t, path); got != doc {
			t.Errorf("%q: file changed to %q", args, got)
		}
	}
}

func TestFR4AC5_CheckPassesWhenCurrent(t *testing.T) {
	doc := "<!-- toc -->\n- [A](#a)\n  - [B](#b)\n<!-- /toc -->\n# A\n## B\n"
	path := writeFile(t, doc)
	code, out, errOut := runMdtoc(t, "", "--check", path)
	if code != 0 || out != "" || errOut != "" {
		t.Errorf("exit %d, stdout %q, stderr %q; want 0, empty, empty", code, out, errOut)
	}
}

func TestFR4AC5_CheckReportsMissingMarkers(t *testing.T) {
	path := writeFile(t, "# A\n")
	code, _, errOut := runMdtoc(t, "", "--check", path)
	if code != 1 || !strings.Contains(errOut, "mdtoc: ") || !strings.Contains(errOut, "missing <!-- toc --> marker") {
		t.Errorf("exit %d, stderr %q; want 1 and a missing-marker error", code, errOut)
	}
}
