package mdtoc

import "testing"

func checkSlugs(t *testing.T, cases []struct{ in, want string }) {
	t.Helper()
	for _, c := range cases {
		if got := Slug(c.in); got != c.want {
			t.Errorf("Slug(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestFR2AC1_LowercaseDropPunctuationSpacesToHyphens(t *testing.T) {
	checkSlugs(t, []struct{ in, want string }{
		{"Hello, World!", "hello-world"},
		{"Go 1.25 notes", "go-125-notes"},
		{"Already-hyphen_and_underscore", "already-hyphen_and_underscore"},
		{"a  b", "a--b"},
		{"Tabs\tdropped", "tabsdropped"},
		{"Ünïcödé Straße", "ünïcödé-straße"},
		{"ΑΒΓ Ελληνικά", "αβγ-ελληνικά"},
		{"日本語 タイトル", "日本語-タイトル"},
		{"Café ½ ٣", "café--٣"},
	})
}

func TestFR2AC2_InlineCodeAndEmphasisMarkersRemoved(t *testing.T) {
	checkSlugs(t, []struct{ in, want string }{
		{"Use `go test`", "use-go-test"},
		{"The ``a`b`` span", "the-ab-span"},
		{"*emphasis* and **strong**", "emphasis-and-strong"},
		{"_emphasis_ and __strong__", "emphasis-and-strong"},
		{"snake_case name", "snake_case-name"},
		{"_snake_case_", "snake_case"},
		{"`_code_` keeps underscores", "_code_-keeps-underscores"},
		{"`snake_case`", "snake_case"},
		{"foo_ bar", "foo_-bar"},
		{"_unpaired", "_unpaired"},
		{"__init__ method", "init-method"},
		{"unclosed `tick", "unclosed-tick"},
	})
}

func TestFR2AC3_DuplicateSlugsGetSuffix(t *testing.T) {
	var s Slugger
	for i, want := range []string{"intro", "intro-1", "intro-2"} {
		if got := s.Slug("Intro"); got != want {
			t.Errorf("occurrence %d: got %q, want %q", i+1, got, want)
		}
	}
}

func TestFR2AC3_SuffixedSlugCollidingWithLiteralStaysUnique(t *testing.T) {
	var s Slugger
	steps := []struct{ in, want string }{
		{"Intro", "intro"},
		{"Intro", "intro-1"},
		{"Intro 1", "intro-1-1"}, // literal collides with generated intro-1
		{"Intro", "intro-2"},
	}
	for _, st := range steps {
		if got := s.Slug(st.in); got != st.want {
			t.Errorf("Slug(%q) = %q, want %q", st.in, got, st.want)
		}
	}

	var s2 Slugger
	steps = []struct{ in, want string }{
		{"Intro 1", "intro-1"}, // literal first
		{"Intro", "intro"},
		{"Intro", "intro-2"}, // intro-1 is taken, skip it
	}
	for _, st := range steps {
		if got := s2.Slug(st.in); got != st.want {
			t.Errorf("Slug(%q) = %q, want %q", st.in, got, st.want)
		}
	}
}

func TestFR2AC3_SluggersAreIndependent(t *testing.T) {
	var a, b Slugger
	a.Slug("x")
	if got := b.Slug("x"); got != "x" {
		t.Errorf("new Slugger: got %q, want %q", got, "x")
	}
}

func TestFR2AC4_EmptySlugIsSection(t *testing.T) {
	checkSlugs(t, []struct{ in, want string }{
		{"!!!", "section"},
		{"", "section"},
		{"**", "section"},
	})
	var s Slugger
	for i, want := range []string{"section", "section-1", "section-2"} {
		if got := s.Slug("?!"); got != want {
			t.Errorf("occurrence %d: got %q, want %q", i+1, got, want)
		}
	}
	if got := s.Slug("Section"); got != "section-3" {
		t.Errorf("literal Section after generated: got %q, want %q", got, "section-3")
	}
}
