package freshness

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "freshness"

// IntentType classifies what kind of content a topic requires before research.
type IntentType string

const (
	IntentNews      IntentType = "news"
	IntentEvergreen IntentType = "evergreen"
	IntentUpdate    IntentType = "update"
	IntentReview    IntentType = "review"
	IntentTutorial  IntentType = "tutorial"
)

// ValidIntents lists every supported intent (deterministic order).
var ValidIntents = []IntentType{
	IntentNews, IntentEvergreen, IntentUpdate, IntentReview, IntentTutorial,
}

// IntentResult is the output of the deterministic intent classifier.
type IntentResult struct {
	Intent     IntentType `json:"intent"`
	Confidence float64    `json:"confidence"` // 0..1
	Signals    []string   `json:"signals"`
}

// TemporalWindow is the dynamic research window derived from an intent.
type TemporalWindow struct {
	Intent         IntentType `json:"intent"`
	PriorityDays   int        `json:"priority_days"`   // 0 = no date priority (evergreen)
	RecentDays     int        `json:"recent_days"`     // soft cutoff for sources considered "recent"
	MaxDays        int        `json:"max_days"`        // 0 = unlimited
	NeverOlderDays int        `json:"never_older_days"` // absolute block; 0 = no block
	VersionFirst   bool       `json:"version_first"`   // UPDATE: always prefer the latest version
	Label          string     `json:"label"`
}

// WindowStrategy returns the TemporalWindow for an intent (deterministic).
func WindowStrategy(i IntentType) TemporalWindow {
	switch i {
	case IntentNews:
		return TemporalWindow{Intent: IntentNews, PriorityDays: 1, RecentDays: 7, MaxDays: 30, NeverOlderDays: 90, Label: "últimas 24h → 7 dias → máx. 30 dias; nunca > 90 dias"}
	case IntentUpdate:
		return TemporalWindow{Intent: IntentUpdate, PriorityDays: 7, RecentDays: 30, MaxDays: 90, NeverOlderDays: 0, VersionFirst: true, Label: "sempre priorizar a versão mais recente (changelog/docs/roadmap)"}
	case IntentReview:
		return TemporalWindow{Intent: IntentReview, PriorityDays: 7, RecentDays: 30, MaxDays: 90, NeverOlderDays: 365, Label: "análises recentes, máx. 90 dias, nunca > 1 ano"}
	case IntentTutorial:
		return TemporalWindow{Intent: IntentTutorial, PriorityDays: 0, RecentDays: 90, MaxDays: 0, NeverOlderDays: 0, Label: "sem prioridade de data; preferir guias atuais"}
	case IntentEvergreen:
		return TemporalWindow{Intent: IntentEvergreen, PriorityDays: 0, RecentDays: 0, MaxDays: 0, NeverOlderDays: 0, Label: "sem prioridade de data (docs, papers, artigos antigos)"}
	}
	return TemporalWindow{Intent: IntentEvergreen}
}

// SourcePriority is the official-source-first tier of a source.
type SourcePriority string

const (
	PriorityOfficial     SourcePriority = "official"
	PriorityDocs         SourcePriority = "docs"
	PriorityBlog         SourcePriority = "blog"
	PriorityChangelog    SourcePriority = "changelog"
	PriorityNewsAgency   SourcePriority = "news_agency"
	PrioritySpecialized  SourcePriority = "specialized"
	PriorityOther        SourcePriority = "other"
	PriorityUnknown      SourcePriority = "unknown"
)

// FreshnessBreakdown is the per-component freshness analysis of one source.
type FreshnessBreakdown struct {
	SourceURL       string         `json:"source_url"`
	Intent          IntentType     `json:"intent"`
	PublishedAt     *time.Time     `json:"published_at,omitempty"`
	UpdatedAt       *time.Time     `json:"updated_at,omitempty"`
	AgeDays         int            `json:"age_days"`
	AgeComponent    float64        `json:"age_component"`    // 0..100
	UpdateComponent float64        `json:"update_component"` // 0..100
	SourceComponent float64        `json:"source_component"` // 0..100
	SourcePriority  SourcePriority `json:"source_priority"`
	Score           float64        `json:"score"`       // weighted 0..100
	Usable          bool           `json:"usable"`      // false when outside the temporal window
	Obsolete        bool           `json:"obsolete"`    // true when an older version of a known entity is mentioned
	ObsoleteEntity  string         `json:"obsolete_entity,omitempty"`
	Reasons         []string       `json:"reasons,omitempty"`
}

// EntityVersion is the known-current version of a product/entity used for
// obsolete-information detection.
type EntityVersion struct {
	Entity string `json:"entity"`
	Current string `json:"current"`
}

// ObsoleteCheck is the result of scanning a text for outdated entity versions.
type ObsoleteCheck struct {
	Entity          string `json:"entity"`
	MentionedVersion string `json:"mentioned_version,omitempty"`
	CurrentVersion  string `json:"current_version"`
	Obsolete        bool   `json:"obsolete"`
	Confidence      float64 `json:"confidence"`
}

// VersionRecord is one stored version of an article.
type VersionRecord struct {
	ID            uuid.UUID        `json:"id,omitempty"`
	SiteID        uuid.UUID        `json:"site_id,omitempty"`
	PublicationID uuid.UUID        `json:"publication_id,omitempty"`
	Version       string           `json:"version"`
	Intent        IntentType       `json:"intent"`
	Changes       []string         `json:"changes"`
	Diff          []VersionDiff    `json:"diff"`
	Sources       []string         `json:"sources"`
	CreatedAt     time.Time        `json:"created_at,omitempty"`
}

// DiffFacet is a dimension compared between versions.
type DiffFacet string

const (
	FacetPrice     DiffFacet = "price"
	FacetContext   DiffFacet = "context"
	FacetLimits    DiffFacet = "limits"
	FacetAPI       DiffFacet = "api"
	FacetBenchmark DiffFacet = "benchmark"
	FacetFeatures  DiffFacet = "features"
)

// VersionDiff is a single facet difference between two versions.
type VersionDiff struct {
	Facet  DiffFacet `json:"facet"`
	Before string    `json:"before"`
	After  string    `json:"after"`
	Changed bool     `json:"changed"`
}

// DedupCandidate is the result of checking whether a topic was already
// covered (same subject, same fact, same day) — update instead of duplicate.
type DedupCandidate struct {
	Fingerprint   string      `json:"fingerprint"`
	Topic         string      `json:"topic"`
	Language      string      `json:"language"`
	Intent        IntentType  `json:"intent"`
	Duplicate     bool        `json:"duplicate"`
	ExistingPubID uuid.UUID   `json:"existing_publication_id,omitempty"`
	ExistingDate  *time.Time  `json:"existing_date,omitempty"`
	SameDay       bool        `json:"same_day"`
	MatchRatio    float64     `json:"match_ratio"`
}

// UpdateStatus for content_updates rows.
type UpdateStatus string

const (
	UpdatePending    UpdateStatus = "pending"
	UpdateNeedsWork  UpdateStatus = "needs_update"
	UpdateResolved   UpdateStatus = "resolved"
)

// ContentUpdate flags an article that requires regeneration.
type ContentUpdate struct {
	ID            uuid.UUID    `json:"id,omitempty"`
	SiteID        uuid.UUID    `json:"site_id,omitempty"`
	PublicationID uuid.UUID    `json:"publication_id"`
	Intent        IntentType   `json:"intent"`
	Reason        string       `json:"reason"`
	OldScore      float64      `json:"old_score"`
	NewScore      float64      `json:"new_score"`
	Details       []string     `json:"details"`
	Status        UpdateStatus `json:"status"`
	CreatedAt     time.Time    `json:"created_at,omitempty"`
	ResolvedAt    *time.Time   `json:"resolved_at,omitempty"`
}

// LanguageStrategy encapsulates the region priorities for a content language.
type LanguageStrategy struct {
	Language       string   `json:"language"`
	PrimaryRegions []string `json:"primary_regions"`
	SecondaryRegions []string `json:"secondary_regions"`
	BlockLiteralTranslation bool `json:"block_literal_translation"`
}

// LanguageStrategyFor returns the deterministic search strategy per language.
func LanguageStrategyFor(lang string) LanguageStrategy {
	switch lang {
	case "pt":
		return LanguageStrategy{
			Language: "pt", PrimaryRegions: []string{"Brasil"},
			SecondaryRegions: []string{"Portugal"},
			BlockLiteralTranslation: true,
		}
	case "en":
		return LanguageStrategy{
			Language: "en", PrimaryRegions: []string{"EUA", "Canadá"},
			SecondaryRegions: []string{"Reino Unido", "Austrália"},
			BlockLiteralTranslation: true,
		}
	}
	return LanguageStrategy{Language: lang, PrimaryRegions: nil, SecondaryRegions: nil}
}

// Events emitted by the freshness module.
var (
	EventIntentClassified    = kernel.EventType("freshness.intent.classified")
	EventSourcesScored       = kernel.EventType("freshness.sources.scored")
	EventVersionSaved        = kernel.EventType("freshness.version.saved")
	EventDedupFound          = kernel.EventType("freshness.dedup.found")
	EventContentNeedsUpdate  = kernel.EventType("freshness.content.needs_update")
	EventSweepCompleted      = kernel.EventType("freshness.sweep.completed")
)

// Sentinel errors.
var (
	ErrIntentRequired     = errors.New("topic or content is required for intent classification")
	ErrInvalidIntent      = errors.New("invalid intent")
	ErrInvalidLanguage    = errors.New("language must be 'pt' or 'en'")
	ErrSourceRequired     = errors.New("at least one source is required")
	ErrEntityRequired     = errors.New("entity and current version are required")
	ErrDatabaseNotAvail   = errors.New("database not available")
	ErrVersionNotFound    = errors.New("version record not found")
	ErrDedupLookupFailed  = errors.New("dedup lookup failed")
	ErrSweepAlreadyRun    = errors.New("freshness sweep already ran today")
	ErrSweepDisabled      = errors.New("freshness sweep is disabled")
)
