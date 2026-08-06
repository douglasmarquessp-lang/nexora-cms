package freshness

import (
	"strings"
	"time"
)

// clampFloat bounds v to [lo, hi].
func clampFloat(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ageDaysBetween returns the whole days elapsed from t to now (0 minimum;
// -1 when t is nil).
func ageDaysBetween(now time.Time, t *time.Time) int {
	if t == nil {
		return -1
	}
	d := now.Sub(*t)
	if d < 0 {
		return 0
	}
	return int(d.Hours() / 24)
}

var priorityMarkers = []struct {
	priority SourcePriority
	markers  []string
}{
	{PriorityChangelog, []string{"changelog", "release note", "release-notes", "release_notes", "releases/"}},
	{PriorityDocs, []string{"docs", "documentation", "/doc/"}},
	{PriorityBlog, []string{"blog"}},
}

// newsAgencies is the deterministic Reuters/AP/Bloomberg/AFP/Wire tier.
var newsAgencies = map[string]bool{
	"reuters.com": true, "apnews.com": true, "ap.org": true,
	"bloomberg.com": true, "afp.com": true, "france24.com": true,
	"ft.com": true, "reuters": true,
}

// specializedSites is a small deterministic "specialized media" tier.
var specializedSites = map[string]bool{
	"theverge.com": true, "techcrunch.com": true, "engadget.com": true,
	"wired.com": true, "techradar.com": true,
}

// SourcePriorityClassify assigns the official-source-first tier for a source.
// targetEntity is the main noun of the topic (optional): when present, a
// registrable-domain match counts as the official site.
func SourcePriorityClassify(title, rawURL, targetEntity string) SourcePriority {
	needle := strings.ToLower(title + " " + rawURL)
	for _, m := range priorityMarkers {
		for _, w := range m.markers {
			if strings.Contains(needle, w) {
				return m.priority
			}
		}
	}
	domain := registrableDomain(rawURL)
	if domain == "" {
		return PriorityUnknown
	}
	if newsAgencies[domain] {
		return PriorityNewsAgency
	}
	if specializedSites[domain] {
		return PrioritySpecialized
	}
	if targetEntity != "" {
		ent := strings.ToLower(stripSpaces(targetEntity))
		if ent != "" && (domain == ent || strings.HasPrefix(domain, ent) || strings.Contains(domain, ent)) {
			return PriorityOfficial
		}
		if ent != "" && firstLabel(rawURL) == ent {
			return PriorityOfficial
		}
	}
	return PriorityOther
}

// firstLabel returns the leftmost hostname label (e.g. "gemini" for
// gemini.google.com).
func firstLabel(raw string) string {
	u := strings.ToLower(strings.TrimSpace(raw))
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	}
	if i := strings.IndexAny(u, "/?#"); i >= 0 {
		u = u[:i]
	}
	u = strings.TrimPrefix(u, "www.")
	if i := strings.IndexByte(u, '.'); i > 0 {
		return u[:i]
	}
	return u
}

func stripSpaces(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == ' ' || r == '-' || r == '_' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// registrableDomain strips scheme/path and keeps the last two labels.
func registrableDomain(raw string) string {
	u := strings.ToLower(strings.TrimSpace(raw))
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+3:]
	}
	if i := strings.IndexAny(u, "/?#"); i >= 0 {
		u = u[:i]
	}
	u = strings.TrimPrefix(u, "www.")
	parts := strings.Split(u, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return u
}

// sourcePriorityRank maps a tier to a 0..100 reputation score.
func sourcePriorityScore(p SourcePriority) float64 {
	switch p {
	case PriorityOfficial:
		return 100
	case PriorityNewsAgency:
		return 98
	case PriorityDocs, PriorityChangelog:
		return 95
	case PriorityBlog:
		return 92
	case PrioritySpecialized:
		return 80
	case PriorityOther:
		return 60
	}
	return 40
}

// ageComponentScore is the date-driven component. When the window is unlimited
// (evergreen/tutorial) date stops being the priority; only extreme age hurts.
func ageComponentScore(ageDays int, win TemporalWindow) float64 {
	if win.MaxDays == 0 {
		if ageDays < 0 {
			return 40
		}
		if ageDays <= 730 {
			return 85
		}
		if ageDays <= 3650 {
			return 70
		}
		return 45
	}
	if ageDays < 0 {
		return 30
	}
	return clampFloat(100-100*float64(ageDays)/float64(win.MaxDays), 15, 100)
}

// updateComponentScore reflects how recently the source was updated.
func updateComponentScore(updateDays int) float64 {
	switch {
	case updateDays < 0:
		return 50 // unknown — neutral
	case updateDays <= 1:
		return 95
	case updateDays <= 7:
		return 90
	case updateDays <= 30:
		return 82
	case updateDays <= 90:
		return 72
	}
	return 60
}

// ComputeSourceFreshness scores one source using the default window for its
// intent. Fully deterministic.
func ComputeSourceFreshness(now time.Time, published, updated *time.Time,
	intent IntentType, title, rawURL, targetEntity string) FreshnessBreakdown {
	return ComputeSourceFreshnessWithWindow(now, published, updated, WindowStrategy(intent), title, rawURL, targetEntity)
}

// ComputeSourceFreshnessWithWindow scores one source using an explicit window.
// Fully deterministic.
func ComputeSourceFreshnessWithWindow(now time.Time, published, updated *time.Time,
	win TemporalWindow, title, rawURL, targetEntity string) FreshnessBreakdown {
	age := ageDaysBetween(now, published)
	upd := ageDaysBetween(now, updated)

	prio := SourcePriorityClassify(title, rawURL, targetEntity)

	ageComp := ageComponentScore(age, win)
	updComp := updateComponentScore(upd)
	srcScore := sourcePriorityScore(prio)

	br := FreshnessBreakdown{
		SourceURL:       rawURL,
		Intent:          win.Intent,
		PublishedAt:     published,
		UpdatedAt:       updated,
		AgeDays:         age,
		AgeComponent:    ageComp,
		UpdateComponent: updComp,
		SourceComponent: srcScore,
		SourcePriority:  prio,
		Usable:          true,
	}
	br.Score = clampFloat(0.50*ageComp+0.30*updComp+0.20*srcScore, 0, 100)

	// Window acceptance rules (e.g. NEWS: max 30 days; never older than 90).
	if win.NeverOlderDays > 0 && age > win.NeverOlderDays {
		br.Usable = false
		br.Score = clampFloat(br.Score-40, 0, 100)
		br.Reasons = append(br.Reasons, "older than the absolute news cutoff")
	} else if win.MaxDays > 0 && age > win.MaxDays {
		br.Usable = false
		br.Reasons = append(br.Reasons, "outside the temporal window")
	}
	if age < 0 {
		br.Reasons = append(br.Reasons, "no publish date")
	}
	return br
}

// SortSourcesByPriorityAndScore returns sources ordered by priority tier then
// by freshness score (deterministic, stable). Obsolete sources rank last.
func SortSourcesByPriorityAndScore(ss []FreshnessBreakdown) []FreshnessBreakdown {
	out := make([]FreshnessBreakdown, len(ss))
	copy(out, ss)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0; j-- {
			a, b := out[j-1], out[j]
			if lessFresh(a, b) {
				out[j-1], out[j] = b, a
			}
		}
	}
	return out
}

// lessFresh reports whether a should rank below b.
func lessFresh(a, b FreshnessBreakdown) bool {
	if a.Obsolete != b.Obsolete {
		return a.Obsolete
	}
	if a.SourcePriority != b.SourcePriority {
		return sourcePriorityRank(a.SourcePriority) < sourcePriorityRank(b.SourcePriority)
	}
	return a.Score < b.Score
}

func sourcePriorityRank(p SourcePriority) int {
	order := []SourcePriority{
		PriorityOfficial, PriorityNewsAgency, PriorityDocs, PriorityChangelog,
		PriorityBlog, PrioritySpecialized, PriorityOther, PriorityUnknown,
	}
	for i, o := range order {
		if o == p {
			return i
		}
	}
	return len(order)
}