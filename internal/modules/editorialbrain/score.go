package editorialbrain

import (
	"time"
)

// ComputeEditorialScore computes the weighted final editorial note and the
// gate decision. Weights: SEO 0.20, EEAT 0.20, Freshness 0.15, Coverage 0.20,
// Naturalness 0.15, Confidence 0.10 (sum = 1.0). Fully deterministic.
func ComputeEditorialScore(seo, eeat, freshness, coverage, naturalness, confidence, threshold float64) EditorialScore {
	if threshold <= 0 {
		threshold = DefaultMinFinalScore
	}
	final := round2(seo*weightSEO + eeat*weightEEAT + freshness*weightFreshness +
		coverage*weightCoverage + naturalness*weightNaturalness + confidence*weightConfidence)
	decision := DecisionApproved
	if final < threshold {
		decision = DecisionNeedsReview
	}
	return EditorialScore{
		SEO:         round2(seo),
		EEAT:        round2(eeat),
		Freshness:   round2(freshness),
		Coverage:    round2(coverage),
		Naturalness: round2(naturalness),
		Confidence:  round2(confidence),
		Final:       final,
		Decision:    decision,
		Threshold:   round2(threshold),
	}
}

// SourceFreshnessScore scores a single source by its publish/update age.
// Deterministic; nil date → neutral 60.
func SourceFreshnessScore(publishedAt, updatedAt *time.Time) float64 {
	ref := publishedAt
	if updatedAt != nil {
		ref = updatedAt
	}
	if ref == nil {
		return 60
	}
	days := time.Since(*ref).Hours() / 24
	switch {
	case days < 0:
		return 100
	case days <= 1:
		return 100
	case days <= 7:
		return 95
	case days <= 30:
		return 85
	case days <= 90:
		return 70
	default:
		return 50
	}
}

// SourcesFreshnessScore averages the freshness of the research sources used
// by the article. No sources → neutral 60.
func SourcesFreshnessScore(sources []SourceRef) float64 {
	if len(sources) == 0 {
		return 60
	}
	total := 0.0
	for _, s := range sources {
		total += SourceFreshnessScore(s.PublishedAt, nil)
	}
	return round2(total / float64(len(sources)))
}
