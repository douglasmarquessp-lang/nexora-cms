package ai

import (
	"testing"
)

func TestExtractDomain(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"https://www.openai.com/research/gpt-5", "openai.com"},
		{"https://reuters.com/technology", "reuters.com"},
		{"https://en.wikipedia.org/wiki/Artificial_intelligence", "wikipedia.org"},
		{"http://blog.google/foo", "google.com"},
		{"https://docs.microsoft.com/en-us", "microsoft.com"},
		{"https://nature.com/articles/x", "nature.com"},
		{"news.bbc.com/story", "bbc.com"},
		{"https://www.gov.br/planejamento", "gov.br"},
		{"", ""},
		{"https://sub.example.org:8443/path?q=1", "sub.example.org"},
	}
	for _, c := range cases {
		got := ExtractDomain(c.in)
		if got != c.want {
			t.Errorf("ExtractDomain(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestReliabilityOfDomain(t *testing.T) {
	cases := []struct {
		domain   string
		wantMin  int
		wantMax  int
		wantZero bool
	}{
		{"openai.com", 100, 100, false},
		{"GOOGLE.COM", 100, 100, false},
		{"reuters.com", 95, 95, false},
		{"wikipedia.org", 70, 70, false},
		{"unknown-blog.example", 30, 30, false},
		{"something.gov", 90, 100, false},
		{"university.edu", 75, 90, false},
		{"", 0, 0, true},
	}
	for _, c := range cases {
		score, label := ReliabilityOfDomain(c.domain)
		if c.wantZero {
			if score != 0 || label != "unknown" {
				t.Errorf("ReliabilityOfDomain(%q) = %d/%q, want 0/unknown", c.domain, score, label)
			}
			continue
		}
		if score < c.wantMin || score > c.wantMax {
			t.Errorf("ReliabilityOfDomain(%q) = %d, want in [%d,%d]", c.domain, score, c.wantMin, c.wantMax)
		}
		if label == "" {
			t.Errorf("ReliabilityOfDomain(%q) missing label", c.domain)
		}
	}
}

func TestReliabilityLabel(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{100, "verified"},
		{95, "verified"},
		{90, "verified"},
		{80, "official"},
		{75, "official"},
		{60, "established"},
		{30, "low"},
		{10, "unknown"},
		{0, "unknown"},
	}
	for _, c := range cases {
		if got := ReliabilityLabel(c.score); got != c.want {
			t.Errorf("ReliabilityLabel(%d) = %q, want %q", c.score, got, c.want)
		}
	}
}

func TestDefaultReliabilityScores(t *testing.T) {
	scores := DefaultReliabilityScores()
	if scores["openai.com"] != 100 {
		t.Error("openai.com should be 100")
	}
	if scores["reuters.com"] != 95 {
		t.Error("reuters.com should be 95")
	}
	if scores["wikipedia.org"] != 70 {
		t.Error("wikipedia.org should be 70")
	}
	if _, ok := scores["nope.example"]; ok {
		t.Error("unknown domains must not appear in the default ranking")
	}
}

func TestReliabilityDeterministic(t *testing.T) {
	first, _ := ReliabilityOfDomain("reuters.com")
	second, _ := ReliabilityOfDomain("reuters.com")
	if first != second {
		t.Error("reliability scoring must be deterministic")
	}
}

func TestReliabilityRankingPriority(t *testing.T) {
	// Trusted vendor beats unknown blog and community wiki.
	openai, _ := ReliabilityOfDomain("openai.com")
	wiki, _ := ReliabilityOfDomain("wikipedia.org")
	blog, _ := ReliabilityOfDomain("random-blog.xyz")
	if !(openai > wiki && wiki > blog) {
		t.Errorf("ranking order broken: %d %d %d", openai, wiki, blog)
	}
}
