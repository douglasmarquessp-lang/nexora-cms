package humanwriter

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

const ModuleName = "humanwriter"

type ProfileSlug string

const (
	ProfileJournalist       ProfileSlug = "journalist"
	ProfileTechWriter       ProfileSlug = "tech_writer"
	ProfileSoftwareReviewer ProfileSlug = "software_reviewer"
	ProfileNewsReporter     ProfileSlug = "news_reporter"
	ProfileTutorialAuthor   ProfileSlug = "tutorial_author"
	ProfileEditorialWriter  ProfileSlug = "editorial_writer"
	ProfileOpinionWriter    ProfileSlug = "opinion_writer"
	ProfileEvergreenWriter  ProfileSlug = "evergreen_writer"
)

var AllProfiles = []ProfileSlug{
	ProfileJournalist, ProfileTechWriter, ProfileSoftwareReviewer,
	ProfileNewsReporter, ProfileTutorialAuthor, ProfileEditorialWriter,
	ProfileOpinionWriter, ProfileEvergreenWriter,
}

var ProfileDefaults = map[ProfileSlug]map[string]interface{}{
	ProfileJournalist: {
		"name":                     "Journalist",
		"tone":                     "neutral",
		"perspective":              "third_person",
		"audience":                 "general_public",
		"expertise_level":          "general",
		"preferred_sentence_length": "medium",
		"paragraph_size_min":        3,
		"paragraph_size_max":        7,
		"vocabulary_tags":           []string{"jargon", "formal"},
		"allowed_connectors":        []string{"addition", "contrast", "cause", "sequence"},
	},
	ProfileTechWriter: {
		"name":                     "Technology Writer",
		"tone":                     "informative",
		"perspective":              "third_person",
		"audience":                 "technical",
		"expertise_level":          "intermediate",
		"preferred_sentence_length": "medium",
		"paragraph_size_min":        2,
		"paragraph_size_max":        6,
		"vocabulary_tags":           []string{"technical", "precise"},
		"allowed_connectors":        []string{"addition", "cause", "sequence", "clarification"},
	},
	ProfileSoftwareReviewer: {
		"name":                     "Software Reviewer",
		"tone":                     "conversational",
		"perspective":              "first_person",
		"audience":                 "developers",
		"expertise_level":          "advanced",
		"preferred_sentence_length": "short",
		"paragraph_size_min":        2,
		"paragraph_size_max":        5,
		"vocabulary_tags":           []string{"technical", "comparative"},
		"allowed_connectors":        []string{"contrast", "addition", "example", "conclusion"},
	},
	ProfileNewsReporter: {
		"name":                     "News Reporter",
		"tone":                     "factual",
		"perspective":              "third_person",
		"audience":                 "general_public",
		"expertise_level":          "general",
		"preferred_sentence_length": "short",
		"paragraph_size_min":        2,
		"paragraph_size_max":        4,
		"vocabulary_tags":           []string{"concise", "direct"},
		"allowed_connectors":        []string{"sequence", "cause", "contrast", "time"},
	},
	ProfileTutorialAuthor: {
		"name":                     "Tutorial Author",
		"tone":                     "instructional",
		"perspective":              "second_person",
		"audience":                 "learners",
		"expertise_level":          "beginner",
		"preferred_sentence_length": "short",
		"paragraph_size_min":        1,
		"paragraph_size_max":        4,
		"vocabulary_tags":           []string{"simple", "action_oriented"},
		"allowed_connectors":        []string{"sequence", "addition", "example", "result"},
	},
	ProfileEditorialWriter: {
		"name":                     "Editorial Writer",
		"tone":                     "authoritative",
		"perspective":              "third_person",
		"audience":                 "informed_public",
		"expertise_level":          "advanced",
		"preferred_sentence_length": "long",
		"paragraph_size_min":        4,
		"paragraph_size_max":        10,
		"vocabulary_tags":           []string{"sophisticated", "persuasive"},
		"allowed_connectors":        []string{"contrast", "cause", "addition", "concession"},
	},
	ProfileOpinionWriter: {
		"name":                     "Opinion Writer",
		"tone":                     "passionate",
		"perspective":              "first_person",
		"audience":                 "engaged_readers",
		"expertise_level":          "intermediate",
		"preferred_sentence_length": "mixed",
		"paragraph_size_min":        2,
		"paragraph_size_max":        6,
		"vocabulary_tags":           []string{"emotive", "rhetorical"},
		"allowed_connectors":        []string{"contrast", "addition", "concession", "emphasis"},
	},
	ProfileEvergreenWriter: {
		"name":                     "Evergreen Writer",
		"tone":                     "timeless",
		"perspective":              "third_person",
		"audience":                 "general_public",
		"expertise_level":          "general",
		"preferred_sentence_length": "medium",
		"paragraph_size_min":        3,
		"paragraph_size_max":        8,
		"vocabulary_tags":           []string{"universal", "durable"},
		"allowed_connectors":        []string{"addition", "contrast", "cause", "sequence", "example"},
	},
}

var ProfileDisplayNames = map[ProfileSlug]string{
	ProfileJournalist:       "Journalist",
	ProfileTechWriter:       "Technology Writer",
	ProfileSoftwareReviewer: "Software Reviewer",
	ProfileNewsReporter:     "News Reporter",
	ProfileTutorialAuthor:   "Tutorial Author",
	ProfileEditorialWriter:  "Editorial Writer",
	ProfileOpinionWriter:    "Opinion Writer",
	ProfileEvergreenWriter:  "Evergreen Writer",
}

var RuleKeys = struct {
	AvoidAICliches          string
	AvoidRepetitiveOpenings string
	AvoidRepetitiveConclusions string
	NaturalParagraphSizes   string
	VariableSentenceLengths string
	NaturalConnectorRotation string
	QuoteInsertionSupport   string
	StatisticInsertionSupport string
	ExpertOpinionPlaceholders string
}{
	AvoidAICliches:             "avoid_ai_cliches",
	AvoidRepetitiveOpenings:    "avoid_repetitive_openings",
	AvoidRepetitiveConclusions: "avoid_repetitive_conclusions",
	NaturalParagraphSizes:      "natural_paragraph_sizes",
	VariableSentenceLengths:    "variable_sentence_lengths",
	NaturalConnectorRotation:   "natural_connector_rotation",
	QuoteInsertionSupport:      "quote_insertion_support",
	StatisticInsertionSupport:  "statistic_insertion_support",
	ExpertOpinionPlaceholders:  "expert_opinion_placeholders",
}

var AllRuleKeys = []string{
	RuleKeys.AvoidAICliches, RuleKeys.AvoidRepetitiveOpenings,
	RuleKeys.AvoidRepetitiveConclusions, RuleKeys.NaturalParagraphSizes,
	RuleKeys.VariableSentenceLengths, RuleKeys.NaturalConnectorRotation,
	RuleKeys.QuoteInsertionSupport, RuleKeys.StatisticInsertionSupport,
	RuleKeys.ExpertOpinionPlaceholders,
}

var RuleCategories = map[string]string{
	RuleKeys.AvoidAICliches:             "style",
	RuleKeys.AvoidRepetitiveOpenings:    "structure",
	RuleKeys.AvoidRepetitiveConclusions: "structure",
	RuleKeys.NaturalParagraphSizes:      "readability",
	RuleKeys.VariableSentenceLengths:    "readability",
	RuleKeys.NaturalConnectorRotation:   "flow",
	RuleKeys.QuoteInsertionSupport:      "enrichment",
	RuleKeys.StatisticInsertionSupport:  "enrichment",
	RuleKeys.ExpertOpinionPlaceholders:  "enrichment",
}

type WritingProfile struct {
	ID                      uuid.UUID              `json:"id"`
	SiteID                  uuid.UUID              `json:"site_id"`
	Slug                    string                 `json:"slug"`
	Name                    string                 `json:"name"`
	Description             string                 `json:"description,omitempty"`
	Tone                    string                 `json:"tone,omitempty"`
	Perspective             string                 `json:"perspective,omitempty"`
	Audience                string                 `json:"audience,omitempty"`
	ExpertiseLevel          string                 `json:"expertise_level,omitempty"`
	Language                string                 `json:"language"`
	VocabularyTags          []string               `json:"vocabulary_tags,omitempty"`
	AllowedConnectors       []string               `json:"allowed_connectors,omitempty"`
	PreferredSentenceLength string                 `json:"preferred_sentence_length,omitempty"`
	ParagraphSizeMin        int                    `json:"paragraph_size_min"`
	ParagraphSizeMax        int                    `json:"paragraph_size_max"`
	IsActive                bool                   `json:"is_active"`
	Metadata                map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy               *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt               time.Time              `json:"created_at"`
	UpdatedAt               time.Time              `json:"updated_at"`
}

type WritingRule struct {
	ID          uuid.UUID              `json:"id"`
	SiteID      uuid.UUID              `json:"site_id"`
	ProfileID   *uuid.UUID             `json:"profile_id,omitempty"`
	RuleKey     string                 `json:"rule_key"`
	Category    string                 `json:"category"`
	Enabled     bool                   `json:"enabled"`
	Priority    int                    `json:"priority"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Description string                 `json:"description,omitempty"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type WritingPersona struct {
	ID              uuid.UUID              `json:"id"`
	SiteID          uuid.UUID              `json:"site_id"`
	ProfileID       *uuid.UUID             `json:"profile_id,omitempty"`
	Name            string                 `json:"name"`
	Title           string                 `json:"title,omitempty"`
	Bio             string                 `json:"bio,omitempty"`
	VoiceTraits     []string               `json:"voice_traits,omitempty"`
	VocabularyStyle []string               `json:"vocabulary_style,omitempty"`
	SentencePatterns []string              `json:"sentence_patterns,omitempty"`
	ExpertiseAreas  []string               `json:"expertise_areas,omitempty"`
	Language        string                 `json:"language"`
	IsActive        bool                   `json:"is_active"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy       *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type VocabularySet struct {
	ID           uuid.UUID              `json:"id"`
	SiteID       uuid.UUID              `json:"site_id"`
	Name         string                 `json:"name"`
	Category     string                 `json:"category,omitempty"`
	Words        []string               `json:"words"`
	Replacements [][]string             `json:"replacements,omitempty"`
	Language     string                 `json:"language"`
	Tags         []string               `json:"tags,omitempty"`
	IsActive     bool                   `json:"is_active"`
	CreatedBy    *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type TransitionPhrase struct {
	ID         uuid.UUID              `json:"id"`
	SiteID     uuid.UUID              `json:"site_id"`
	Category   string                 `json:"category"`
	Phrase     string                 `json:"phrase"`
	Language   string                 `json:"language"`
	Formality  string                 `json:"formality,omitempty"`
	UsageCount int                    `json:"usage_count"`
	IsActive   bool                   `json:"is_active"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
}

type StylePattern struct {
	ID                uuid.UUID              `json:"id"`
	SiteID            uuid.UUID              `json:"site_id"`
	ProfileID         *uuid.UUID             `json:"profile_id,omitempty"`
	Name              string                 `json:"name"`
	PatternType       string                 `json:"pattern_type"`
	Pattern           string                 `json:"pattern"`
	Language          string                 `json:"language"`
	Tags              []string               `json:"tags,omitempty"`
	EffectivenessScore float64                `json:"effectiveness_score"`
	UsageCount        int                    `json:"usage_count"`
	IsActive          bool                   `json:"is_active"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

type SentenceTemplate struct {
	ID        uuid.UUID              `json:"id"`
	SiteID    uuid.UUID              `json:"site_id"`
	ProfileID *uuid.UUID             `json:"profile_id,omitempty"`
	Name      string                 `json:"name"`
	Template  string                 `json:"template"`
	Category  string                 `json:"category,omitempty"`
	Variables []string               `json:"variables,omitempty"`
	Language  string                 `json:"language"`
	Formality string                 `json:"formality,omitempty"`
	UsageCount int                   `json:"usage_count"`
	IsActive  bool                   `json:"is_active"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

type HumanizationRecord struct {
	ID                uuid.UUID              `json:"id"`
	SiteID            uuid.UUID              `json:"site_id"`
	ProfileID         *uuid.UUID             `json:"profile_id,omitempty"`
	SourceText        string                 `json:"source_text"`
	HumanizedText     string                 `json:"humanized_text"`
	BurstinessScore   float64                `json:"burstiness_score"`
	PerplexityScore   float64                `json:"perplexity_score"`
	RepetitionScore   float64                `json:"repetition_score"`
	PassiveVoiceScore float64                `json:"passive_voice_score"`
	RhythmScore       float64                `json:"rhythm_score"`
	FlowScore         float64                `json:"flow_score"`
	RulesApplied      []string               `json:"rules_applied,omitempty"`
	Transformations   []map[string]interface{} `json:"transformations,omitempty"`
	Language          string                 `json:"language"`
	WordCountOriginal int                    `json:"word_count_original"`
	WordCountHumanized int                   `json:"word_count_humanized"`
	DurationMs        int                    `json:"duration_ms"`
	CreatedBy         *uuid.UUID             `json:"created_by,omitempty"`
	CreatedAt         time.Time              `json:"created_at"`
}

type HumanizationResult struct {
	HumanizedText     string                   `json:"humanized_text"`
	BurstinessScore   float64                  `json:"burstiness_score"`
	PerplexityScore   float64                  `json:"perplexity_score"`
	RepetitionScore   float64                  `json:"repetition_score"`
	PassiveVoiceScore float64                  `json:"passive_voice_score"`
	RhythmScore       float64                  `json:"rhythm_score"`
	FlowScore         float64                  `json:"flow_score"`
	RulesApplied      []string                 `json:"rules_applied,omitempty"`
	Transformations   []map[string]interface{} `json:"transformations,omitempty"`
	WordCountOriginal int                      `json:"word_count_original"`
	WordCountHumanized int                     `json:"word_count_humanized"`
}

type AnalyzeRequest struct {
	Text     string `json:"text"`
	Language string `json:"language,omitempty"`
}

type AnalyzeResult struct {
	BurstinessScore   float64 `json:"burstiness_score"`
	PerplexityScore   float64 `json:"perplexity_score"`
	RepetitionScore   float64 `json:"repetition_score"`
	PassiveVoiceScore float64 `json:"passive_voice_score"`
	RhythmScore       float64 `json:"rhythm_score"`
	FlowScore         float64 `json:"flow_score"`
	SentenceCount     int     `json:"sentence_count"`
	ParagraphCount    int     `json:"paragraph_count"`
	WordCount         int     `json:"word_count"`
	AvgSentenceLength float64 `json:"avg_sentence_length"`
	AvgParagraphSize  float64 `json:"avg_paragraph_size"`
	VocabularyDensity float64 `json:"vocabulary_density"`
}

type ProfileCreateRequest struct {
	Slug                    string                 `json:"slug"`
	Name                    string                 `json:"name"`
	Description             string                 `json:"description,omitempty"`
	Tone                    string                 `json:"tone,omitempty"`
	Perspective             string                 `json:"perspective,omitempty"`
	Audience                string                 `json:"audience,omitempty"`
	ExpertiseLevel          string                 `json:"expertise_level,omitempty"`
	Language                string                 `json:"language,omitempty"`
	VocabularyTags          []string               `json:"vocabulary_tags,omitempty"`
	AllowedConnectors       []string               `json:"allowed_connectors,omitempty"`
	PreferredSentenceLength string                 `json:"preferred_sentence_length,omitempty"`
	ParagraphSizeMin        int                    `json:"paragraph_size_min,omitempty"`
	ParagraphSizeMax        int                    `json:"paragraph_size_max,omitempty"`
	Metadata                map[string]interface{} `json:"metadata,omitempty"`
}

type ProfileUpdateRequest struct {
	Name                    *string    `json:"name,omitempty"`
	Description             *string    `json:"description,omitempty"`
	Tone                    *string    `json:"tone,omitempty"`
	Perspective             *string    `json:"perspective,omitempty"`
	Audience                *string    `json:"audience,omitempty"`
	ExpertiseLevel          *string    `json:"expertise_level,omitempty"`
	VocabularyTags          *[]string  `json:"vocabulary_tags,omitempty"`
	AllowedConnectors       *[]string  `json:"allowed_connectors,omitempty"`
	PreferredSentenceLength *string    `json:"preferred_sentence_length,omitempty"`
	ParagraphSizeMin        *int       `json:"paragraph_size_min,omitempty"`
	ParagraphSizeMax        *int       `json:"paragraph_size_max,omitempty"`
	IsActive                *bool      `json:"is_active,omitempty"`
	Metadata                *map[string]interface{} `json:"metadata,omitempty"`
}

type HumanizeRequest struct {
	Text      string  `json:"text"`
	ProfileID *uuid.UUID `json:"profile_id,omitempty"`
	Slug      string  `json:"slug,omitempty"`
	Language  string  `json:"language,omitempty"`
}

type BatchHumanizeRequest struct {
	Texts     []string  `json:"texts"`
	ProfileID *uuid.UUID `json:"profile_id,omitempty"`
	Slug      string    `json:"slug,omitempty"`
	Language  string    `json:"language,omitempty"`
}

type BatchHumanizeResult struct {
	Results []HumanizationResult `json:"results"`
}

type HumanWriterMetrics struct {
	TotalRequests    int64   `json:"total_requests"`
	AvgBurstiness    float64 `json:"avg_burstiness"`
	AvgPerplexity    float64 `json:"avg_perplexity"`
	AvgRepetition    float64 `json:"avg_repetition"`
	AvgPassiveVoice  float64 `json:"avg_passive_voice"`
	AvgRhythm        float64 `json:"avg_rhythm"`
	AvgFlow          float64 `json:"avg_flow"`
	ProfileCount     int     `json:"profile_count"`
	RuleCount        int     `json:"rule_count"`
	PersonaCount     int     `json:"persona_count"`
	VocabularyCount  int     `json:"vocabulary_count"`
	TransitionCount  int     `json:"transition_count"`
	TemplateCount    int     `json:"template_count"`
	PatternCount     int     `json:"pattern_count"`
}

const (
	EventHumanCreated     kernel.EventType = "humanwriter.created"
	EventHumanized        kernel.EventType = "humanwriter.humanized"
	EventProfileCreated   kernel.EventType = "humanwriter.profile.created"
	EventProfileUpdated   kernel.EventType = "humanwriter.profile.updated"
	EventProfileDeleted   kernel.EventType = "humanwriter.profile.deleted"
	EventRuleToggled      kernel.EventType = "humanwriter.rule.toggled"
	EventPersonaCreated   kernel.EventType = "humanwriter.persona.created"
	EventPersonaUpdated   kernel.EventType = "humanwriter.persona.updated"
	EventPersonaDeleted   kernel.EventType = "humanwriter.persona.deleted"
	EventBatchHumanized   kernel.EventType = "humanwriter.batch.humanized"
)

var (
	ErrProfileNotFound       = errors.New("writing profile not found")
	ErrRuleNotFound          = errors.New("writing rule not found")
	ErrPersonaNotFound       = errors.New("writing persona not found")
	ErrVocabularyNotFound    = errors.New("vocabulary set not found")
	ErrTransitionNotFound    = errors.New("transition phrase not found")
	ErrPatternNotFound       = errors.New("style pattern not found")
	ErrTemplateNotFound      = errors.New("sentence template not found")
	ErrHistoryNotFound       = errors.New("humanization history not found")
	ErrInvalidSlug           = errors.New("invalid profile slug")
	ErrInvalidText           = errors.New("text is required")
	ErrInvalidLanguage       = errors.New("language must be 'pt' or 'en'")
	ErrProfileExists         = errors.New("profile slug already exists")
	ErrDatabaseNotAvail      = errors.New("database not available")
)
