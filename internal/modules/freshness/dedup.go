package freshness

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// Candidate describes an existing publication to compare against.
type Candidate struct {
	PublicationID uuid.UUID
	Topic         string
	PublishedAt   *time.Time
}

// CheckDuplicate decides whether a topic was already covered: same subject
// (token overlap ≥ 0.6) + same day → duplicate. When a duplicate is found the
// recommendation is to UPDATE the existing article instead of creating a new
// one. Deterministic.
func CheckDuplicate(topic, primaryText, lang string, now time.Time, existing []Candidate) DedupCandidate {
	fp := Fingerprint(topic, primaryText)
	dc := DedupCandidate{
		Fingerprint: fp,
		Topic:       topic,
		Language:    lang,
		Intent:      IntentEvergreen,
	}
	best := 0.0
	for _, c := range existing {
		ratio := TokenJaccard(topic, c.Topic)
		if ratio > best {
			best = ratio
			dc.ExistingPubID = c.PublicationID
			dc.ExistingDate = c.PublishedAt
		}
		if c.PublishedAt != nil && sameDay(now, *c.PublishedAt) {
			dc.SameDay = true
		}
	}
	dc.MatchRatio = best
	dc.Duplicate = best >= 0.6 && dc.SameDay
	if dc.Duplicate {
		dc.Intent = classifyForDedup(topic, lang)
	}
	return dc
}

func classifyForDedup(topic, lang string) IntentType {
	res, err := ClassifyIntent(topic, "", lang)
	if err != nil {
		return IntentEvergreen
	}
	return res.Intent
}

// sameDay reports whether two times fall on the same calendar day.
func sameDay(a, b time.Time) bool {
	ya, ma, da := a.Date()
	yb, mb, db := b.Date()
	return ya == yb && ma == mb && da == db
}

// shouldBlockTranslation implements the "never translate a BR news literally
// when official EN sources exist" rule: PT content gets a translation block
// flag when English sources exist for the same topic.
func shouldBlockTranslation(lang string, englishSourcesExist bool) bool {
	if lang == "pt" && englishSourcesExist {
		return true
	}
	return false
}

// SourceMatchesRegion scores a source domain for a language strategy:
// pt prefers .br/.pt; en prefers .com/.ca/.co.uk/.com.au.
func SourceMatchesRegion(domain, lang string) bool {
	d := strings.ToLower(domain)
	switch lang {
	case "pt":
		return strings.HasSuffix(d, ".br") || strings.HasSuffix(d, ".pt")
	case "en":
		return strings.HasSuffix(d, ".com") || strings.HasSuffix(d, ".ca") ||
			strings.HasSuffix(d, ".co.uk") || strings.HasSuffix(d, ".com.au")
	}
	return true
}

// SweepDecision is the deterministic outcome of a daily freshness re-evaluation.
type SweepDecision struct {
	PublicationID uuid.UUID `json:"publication_id"`
	Intent        IntentType `json:"intent"`
	OldScore      float64   `json:"old_score"`
	NewScore      float64   `json:"new_score"`
	NeedsUpdate   bool      `json:"needs_update"`
	Reason        string    `json:"reason"`
	Details       []string  `json:"details"`
}

// ReEvaluateArticle is the once-per-day freshness re-check of one article
// using the default temporal window.
func ReEvaluateArticle(articlePublishedAt time.Time, intent IntentType, now time.Time, hasNewerSource bool) SweepDecision {
	return ReEvaluateArticleWithWindow(articlePublishedAt, intent, now, hasNewerSource, WindowStrategy(intent))
}

// ReEvaluateArticleWithWindow is the once-per-day freshness re-check of one
// article with an explicit window. An article is marked Needs Update when a
// newer source for the same topic exists (news_dedup newer than the article)
// or when the article falls outside its own temporal window. Deterministic.
func ReEvaluateArticleWithWindow(articlePublishedAt time.Time, intent IntentType, now time.Time, hasNewerSource bool, win TemporalWindow) SweepDecision {
	age := ageDaysBetween(now, &articlePublishedAt)
	dec := SweepDecision{
		Intent:   intent,
		OldScore: ComputeSourceFreshnessWithWindow(now, &articlePublishedAt, nil, win, "", "", "").Score,
		NewScore: 0,
	}
	if win.MaxDays > 0 && age > win.MaxDays {
		dec.NeedsUpdate = true
		dec.Reason = "outside_temporal_window"
		dec.Details = append(dec.Details, "artigo mais velho que a janela "+win.Label)
	}
	if hasNewerSource {
		dec.NeedsUpdate = true
		dec.Reason = "newer_source_found"
		dec.Details = append(dec.Details, "nova fonte mais recente encontrada para o mesmo tópico")
	}
	if dec.NeedsUpdate {
		dec.NewScore = ComputeSourceFreshnessWithWindow(now, &articlePublishedAt, nil, win, "", "", "").Score
	}
	return dec
}

// DailySweepOnce is the guard for "once per day": a sweep is allowed only when
// the previous run was on a different calendar day.
func DailySweepOnce(lastRun *time.Time, now time.Time) bool {
	if lastRun == nil {
		return true
	}
	return !sameDay(*lastRun, now)
}