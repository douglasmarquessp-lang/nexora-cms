package ai

import (
	"strings"
)

// Reliability tiers for source domains. The pipeline uses these scores to
// prioritize which sources to trust during fact extraction and briefing.
const (
	ReliabilityUnknown     = 0
	ReliabilityLow         = 30
	ReliabilityAcceptable  = 55
	ReliabilityEstablished = 75
	ReliabilityOfficial    = 90
	ReliabilityVerified    = 100
)

// defaultReliabilityScores is the built-in domain ranking used when no
// per-site override exists. Domains not listed fall back to the suffix-based
// rules (gov/edu) or ReliabilityLow (30) for unknown domains.
var defaultReliabilityScores = map[string]int{
	"openai.com":         100,
	"google.com":         100,
	"microsoft.com":      100,
	"nature.com":         100,
	"anthropic.com":      100,
	"deepmind.com":       100,
	"reuters.com":        95,
	"apnews.com":         95,
	"bbc.com":            90,
	"bbc.co.uk":          90,
	"nytimes.com":        90,
	"wsj.com":            90,
	"ft.com":             90,
	"theguardian.com":    90,
	"washingtonpost.com": 90,
	"statista.com":       80,
	"ourworldindata.org": 85,
	"gov.br":             90,
	"wikipedia.org":      70,
	"github.com":         75,
	"arxiv.org":          85,
	"ieee.org":           90,
	"acm.org":            90,
	"springer.com":       90,
	"elsevier.com":       90,
	"science.org":        100,
	"pnas.org":           100,
	"who.int":            95,
	"un.org":             95,
	"nasa.gov":           100,
	"nih.gov":            95,
	"cisa.gov":           90,
	"w3.org":             90,
}

// ExtractDomain returns the registrable domain of a URL (host without
// www./m./blog. prefixes and language subdomains). Unknown/malformed URLs
// yield "". Bare single-label hosts (e.g. "google" from blog.google) get the
// ".com" TLD appended so downstream scoring and display stay consistent.
func ExtractDomain(rawURL string) string {
	u := rawURL
	if idx := strings.Index(u, "://"); idx >= 0 {
		u = u[idx+3:]
	}
	if idx := strings.Index(u, "/"); idx >= 0 {
		u = u[:idx]
	}
	if idx := strings.Index(u, "?"); idx >= 0 {
		u = u[:idx]
	}
	if idx := strings.Index(u, "#"); idx >= 0 {
		u = u[:idx]
	}
	if idx := strings.Index(u, ":"); idx >= 0 {
		u = u[:idx]
	}
	u = strings.ToLower(strings.TrimSpace(u))
	for _, prefix := range []string{
		"www.", "m.", "blog.", "news.", "docs.",
		"en.", "pt.", "es.", "de.", "fr.", "it.", "nl.", "sv.", "ru.", "ar.",
		"zh.", "ja.", "ko.",
	} {
		for strings.HasPrefix(u, prefix) {
			u = u[len(prefix):]
		}
	}
	if u != "" && !strings.Contains(u, ".") {
		u += ".com"
	}
	return u
}

// ReliabilityOfDomain scores a domain deterministically: exact allowlist match
// wins; then gov/edu/inf/mil suffixes (also on progressively shortened hosts,
// e.g. en.wikipedia.org → wikipedia.org); unknown domains get ReliabilityLow (30).
func ReliabilityOfDomain(domain string) (score int, label string) {
	domain = strings.ToLower(strings.TrimSpace(domain))
	if domain == "" {
		return ReliabilityUnknown, "unknown"
	}
	for {
		if s, ok := defaultReliabilityScores[domain]; ok {
			return s, labelForScore(s)
		}
		switch {
		case strings.HasSuffix(domain, ".gov") || strings.HasSuffix(domain, ".gov.br") ||
			strings.HasSuffix(domain, ".mil") || strings.HasSuffix(domain, ".int"):
			return ReliabilityOfficial, labelForScore(ReliabilityOfficial)
		case strings.HasSuffix(domain, ".edu") || strings.HasSuffix(domain, ".edu.br"):
			return ReliabilityEstablished, labelForScore(ReliabilityEstablished)
		}
		idx := strings.IndexByte(domain, '.')
		if idx < 0 {
			break
		}
		domain = domain[idx+1:]
	}
	return ReliabilityLow, labelForScore(ReliabilityLow)
}

// ReliabilityLabel returns the human label for a reliability score.
func ReliabilityLabel(score int) string {
	return labelForScore(score)
}

// DefaultReliabilityScores exposes the built-in domain ranking (read-only).
func DefaultReliabilityScores() map[string]int {
	out := make(map[string]int, len(defaultReliabilityScores))
	for k, v := range defaultReliabilityScores {
		out[k] = v
	}
	return out
}

func labelForScore(score int) string {
	switch {
	case score >= 90:
		return "verified"
	case score >= 75:
		return "official"
	case score >= 55:
		return "established"
	case score >= 30:
		return "low"
	default:
		return "unknown"
	}
}
