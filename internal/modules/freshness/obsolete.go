package freshness

import (
	"regexp"
	"strconv"
	"strings"
)

// tokenRE captures version-ish tokens in a text: GPT-6, GPT 6, GPT6,
// Gemini 2.5, v4.1, versão 2.3.
var tokenRE = regexp.MustCompile(`(?i)([a-z0-9]{2,})[\s\-]?(v?\d+(?:\.\d+){0,2})`)

// parseVersionSegment splits a version string into (major, minor).
func parseVersionSegment(v string) (int, int, bool) {
	v = strings.TrimPrefix(strings.ToLower(v), "v")
	parts := strings.Split(strings.TrimSpace(v), ".")
	if len(parts) == 0 {
		return 0, 0, false
	}
	maj, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return 0, 0, false
	}
	minor := 0
	if len(parts) > 1 {
		if m, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
			minor = m
		}
	}
	return maj, minor, true
}

// compareEntities compares two version strings ("2.5" vs "6"), missing → equal.
func compareEntities(a, b string) int {
	ma, mia, oka := parseVersionSegment(a)
	mb, mib, okb := parseVersionSegment(b)
	if !oka || !okb {
		return strings.Compare(a, b)
	}
	if ma != mb {
		return sign(ma - mb)
	}
	return sign(mia - mib)
}

func sign(n int) int {
	if n < 0 {
		return -1
	}
	if n > 0 {
		return 1
	}
	return 0
}

// CheckObsolete scans text for mentions of a known entity whose version is
// older than its current known version. Obsolete info must never be used as
// the primary source. Deterministic.
func CheckObsolete(text string, ev EntityVersion) ObsoleteCheck {
	res := ObsoleteCheck{Entity: ev.Entity, CurrentVersion: ev.Current}
	ent := strings.ToLower(strings.TrimSpace(ev.Entity))
	cur := strings.TrimSpace(ev.Current)
	if ent == "" || cur == "" {
		return res
	}
	low := strings.ToLower(text)
	mention := ""
	for _, m := range tokenRE.FindAllStringSubmatch(low, -1) {
		// Brand must appear adjacent to the version token.
		if strings.HasPrefix(m[1], ent) || strings.HasPrefix(ent, m[1]) {
			mention = m[2]
			break
		}
	}
	if mention == "" {
		return res
	}
	res.MentionedVersion = mention
	if cmp := compareEntities(mention, cur); cmp < 0 {
		res.Obsolete = true
		res.Confidence = 0.92
	}
	return res
}

// DetectObsoleteText
// DetectObsoleteSources flags every entity mention that is outdated.
func DetectObsoleteSources(text string, entities []EntityVersion) []ObsoleteCheck {
	var out []ObsoleteCheck
	for _, e := range entities {
		c := CheckObsolete(text, e)
		if c.Obsolete {
			out = append(out, c)
		}
	}
	return out
}

// DeriveMainEntity returns the topic's most prominent token (single word) used
// for official-domain matching. Deterministic.
func DeriveMainEntity(topic string) string {
	t := strings.ToLower(strings.TrimSpace(topic))
	t = strings.ReplaceAll(t, ":", " ")
	t = strings.ReplaceAll(t, ",", " ")
	stop := map[string]bool{
		"o": true, "a": true, "as": true, "os": true, "de": true, "do": true,
		"da": true, "em": true, "com": true, "para": true, "e": true,
		"how": true, "to": true, "the": true, "what": true, "is": true,
		"of": true, "for": true, "in": true, "and": true, "an": true,
		"nova": true, "novo": true, "guia": true, "como": true, "fazer": true,
	}
	for _, w := range strings.Fields(t) {
		w = strings.Trim(w, "[](){}")
		if !stop[w] && len(w) > 2 {
			return strings.ReplaceAll(w, "-", "")
		}
	}
	return ""
}