// Package editorialbrain implements the AI Editorial Brain: before any
// article is written it builds an intelligent editorial brief (search intent,
// persona, outline, required questions) and before publication it produces a
// full editorial note (coverage, fluency, evidence, per-block confidence,
// semantic SEO) with a weighted final score and an approve/review decision.
package editorialbrain

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

// SearchIntent is the search intent classification. It changes how the whole
// article is written.
type SearchIntent string

const (
	IntentInformational SearchIntent = "informational"
	IntentCommercial    SearchIntent = "commercial"
	IntentNavigational  SearchIntent = "navigational"
	IntentComparison    SearchIntent = "comparison"
	IntentTutorial      SearchIntent = "tutorial"
	IntentBreakingNews  SearchIntent = "breaking_news"
	IntentUpdate        SearchIntent = "update"
)

// IntentLabel returns the bilingual label of a search intent.
func (i SearchIntent) Label(lang string) string {
	switch i {
	case IntentCommercial:
		return b("Comercial", "Commercial").text(lang)
	case IntentNavigational:
		return b("Navegacional", "Navigational").text(lang)
	case IntentComparison:
		return b("Comparação", "Comparison").text(lang)
	case IntentTutorial:
		return b("Tutorial", "Tutorial").text(lang)
	case IntentBreakingNews:
		return b("Breaking News", "Breaking News").text(lang)
	case IntentUpdate:
		return b("Atualização", "Update").text(lang)
	default:
		return b("Informacional", "Informational").text(lang)
	}
}

// Persona is the automatic reader profile detected from the topic.
type Persona string

const (
	PersonaDeveloper Persona = "developer"
	PersonaBusiness  Persona = "business"
	PersonaCreator   Persona = "creator"
	PersonaGeneral   Persona = "general"
)

// AudienceLabel returns the bilingual audience description of a persona.
func (p Persona) AudienceLabel(lang string) string {
	switch p {
	case PersonaDeveloper:
		return b("Desenvolvedores e profissionais técnicos que precisam de detalhes de implementação.", "Developers and technical professionals who need implementation details.").text(lang)
	case PersonaBusiness:
		return b("Empresários e gestores focados em custo, ROI e decisões de negócio.", "Business owners and managers focused on cost, ROI and business decisions.").text(lang)
	case PersonaCreator:
		return b("Criadores de conteúdo e produtores de mídia.", "Content creators and media producers.").text(lang)
	default:
		return b("Público geral sem conhecimento técnico prévio.", "General audience without prior technical knowledge.").text(lang)
	}
}

// IntentResult is the classifier output.
type IntentResult struct {
	Intent      SearchIntent `json:"intent"`
	Confidence  float64      `json:"confidence"`
	Signals     []string     `json:"signals"`
	Language    string       `json:"language"`
	VersionHint bool         `json:"version_hint"`
}

// PersonaResult is the persona detector output.
type PersonaResult struct {
	Persona    Persona  `json:"persona"`
	Confidence float64  `json:"confidence"`
	Audience   string   `json:"audience"`
	Reasons    []string `json:"reasons"`
	Language   string   `json:"language"`
}

// SectionType describes the purpose of an outline section.
type SectionType string

const (
	SecIntro      SectionType = "intro"
	SecWhatIs     SectionType = "what_is"
	SecHowWorks   SectionType = "how_works"
	SecSteps      SectionType = "steps"
	SecCost       SectionType = "cost"
	SecComparison SectionType = "comparison"
	SecTable      SectionType = "table"
	SecChanges    SectionType = "changes"
	SecLimits     SectionType = "limitations"
	SecWhoUses    SectionType = "who_should_use"
	SecAlternates SectionType = "alternatives"
	SecFAQ        SectionType = "faq"
	SecCallout    SectionType = "callout"
	SecConclusion SectionType = "conclusion"
)

// OutlineSection is one section of the generated outline.
type OutlineSection struct {
	Order   int         `json:"order"`
	Type    SectionType `json:"type"`
	Title   string      `json:"title"`
	Purpose string      `json:"purpose"`
}

// EditorialOutline is the intelligent outline generated before the AI writes.
type EditorialOutline struct {
	SuggestedTitle string            `json:"suggested_title"`
	Sections       []OutlineSection  `json:"sections"`
	FAQs           []string          `json:"faqs"`
	NeedsTable     bool              `json:"needs_table"`
	Comparisons    bool              `json:"comparisons"`
	Callouts       bool              `json:"callouts"`
	Rationale      string            `json:"rationale"`
}

// QuestionID identifies one of the questions the article must answer.
type QuestionID string

const (
	QWhatIs     QuestionID = "what_is"
	QHowWorks   QuestionID = "how_works"
	QCost       QuestionID = "cost"
	QWorthIt    QuestionID = "worth_it"
	QWhatChanged QuestionID = "what_changed"
	QWhoCanUse  QuestionID = "who_can_use"
	QLimits     QuestionID = "limitations"
	QAlternates QuestionID = "alternatives"
)

// RequiredQuestion is one required question with its answer verification.
type RequiredQuestion struct {
	ID        QuestionID `json:"id"`
	Question  string     `json:"question"`
	Answered  bool       `json:"answered"`
	Evidence  string     `json:"evidence,omitempty"`
}

// QuestionCheck verifies that every required question was answered.
type QuestionCheck struct {
	Questions       []RequiredQuestion `json:"questions"`
	AnsweredCount   int                `json:"answered_count"`
	Total           int                `json:"total"`
	AnsweredPercent float64            `json:"answered_percent"`
}

// FacetID identifies a coverage facet.
type FacetID string

const (
	FacetPrice        FacetID = "price"
	FacetLimitations  FacetID = "limitations"
	FacetAPI          FacetID = "api"
	FacetUseCases     FacetID = "use_cases"
	FacetAlternatives FacetID = "alternatives"
	FacetInstallation FacetID = "installation"
	FacetRequirements FacetID = "requirements"
	FacetComparison   FacetID = "comparison"
)

// FacetIssue is a coverage gap with a bilingual message.
type FacetIssue struct {
	Facet   FacetID `json:"facet"`
	Message string  `json:"message"`
}

// CoverageReport measures how much of the subject the article explains.
type CoverageReport struct {
	CoveragePercent float64      `json:"coverage_percent"`
	Covered         []FacetID    `json:"covered"`
	Missing         []FacetIssue `json:"missing"`
	TotalFacets     int          `json:"total_facets"`
}

// FluencyIssue is one fluency problem found in the text.
type FluencyIssue struct {
	Kind     string  `json:"kind"`
	Message  string  `json:"message"`
	Severity string  `json:"severity"`
	Score    float64 `json:"score"`
}

// FluencyReport checks reading fluency: repetitions, passive voice,
// huge paragraphs, tiring reading.
type FluencyReport struct {
	OverallScore        float64         `json:"overall_score"`
	ReadabilityScore    float64         `json:"readability_score"`
	SentenceRepetition  float64         `json:"sentence_repetition"`
	WordRepetition      float64         `json:"word_repetition"`
	PassiveVoice        float64         `json:"passive_voice"`
	ParagraphScore      float64         `json:"paragraph_score"`
	AvgSentenceLength   float64         `json:"avg_sentence_length"`
	MaxParagraphWords   int             `json:"max_paragraph_words"`
	RepeatedSentences   int             `json:"repeated_sentences"`
	PassiveCount        int             `json:"passive_count"`
	LongParagraphs      int             `json:"long_paragraphs"`
	Issues              []FluencyIssue  `json:"issues"`
}

// FactEntry is a structured fact used for evidence linking (DB-free shape).
type FactEntry struct {
	FactType   string `json:"fact_type"`
	Entity     string `json:"entity"`
	Value      string `json:"value"`
	SourceURL  string `json:"source_url,omitempty"`
	Confidence int    `json:"confidence"`
}

// SourceRef is a research source used for evidence + freshness (DB-free).
type SourceRef struct {
	Title           string     `json:"title"`
	URL             string     `json:"url"`
	Domain          string     `json:"domain,omitempty"`
	Snippet         string     `json:"snippet,omitempty"`
	ReliabilityScore int       `json:"reliability_score"`
	Language        string     `json:"language,omitempty"`
	PublishedAt     *time.Time `json:"published_at,omitempty"`
	IsVerified      bool       `json:"is_verified,omitempty"`
}

// EvidenceLink ties a claim to the source that supports it.
type EvidenceLink struct {
	Claim         string  `json:"claim"`
	Verified      bool    `json:"verified"`
	SourceTitle   string  `json:"source_title,omitempty"`
	SourceURL     string  `json:"source_url,omitempty"`
	Confidence    float64 `json:"confidence"`
	Note          string  `json:"note"`
}

// EvidenceReport scores how well claims are linked to research sources.
type EvidenceReport struct {
	EvidenceScore  float64        `json:"evidence_score"`
	ClaimsCount    int            `json:"claims_count"`
	VerifiedCount  int            `json:"verified_count"`
	Links          []EvidenceLink `json:"links"`
}

// BlockScore is the per-block confidence of an article.
type BlockScore struct {
	Block         string  `json:"block"`
	Score         float64 `json:"score"`
	EvidenceCount int     `json:"evidence_count"`
	Note          string  `json:"note"`
}

// SemanticIssue is one semantic SEO finding.
type SemanticIssue struct {
	Kind    string `json:"kind"`
	Term    string `json:"term"`
	Message string `json:"message"`
}

// SemanticReport verifies entities, related concepts, missing terms,
// FAQ coverage and natural synonyms.
type SemanticReport struct {
	SemanticScore   float64         `json:"semantic_score"`
	EntitiesFound   []string        `json:"entities_found"`
	EntitiesMissing []string        `json:"entities_missing"`
	ConceptsMissing []string        `json:"concepts_missing"`
	MissingTerms    []string        `json:"missing_terms"`
	FaqCoverage     float64         `json:"faq_coverage"`
	SynonymVariety  float64         `json:"synonym_variety"`
	Issues          []SemanticIssue `json:"issues"`
}

// ReviewDecision is the gate decision of the final editorial note.
type ReviewDecision string

const (
	DecisionApproved     ReviewDecision = "approved"
	DecisionNeedsReview  ReviewDecision = "needs_review"
)

// EditorialScore is the final editorial note (0-100 each).
type EditorialScore struct {
	SEO         float64        `json:"seo"`
	EEAT        float64        `json:"eeat"`
	Freshness   float64        `json:"freshness"`
	Coverage    float64        `json:"coverage"`
	Naturalness float64        `json:"naturalness"`
	Confidence  float64        `json:"confidence"`
	Final       float64        `json:"final"`
	Decision    ReviewDecision `json:"decision"`
	Threshold   float64        `json:"threshold"`
}

// EditorialBrief is the persisted editorial brief.
type EditorialBrief struct {
	ID               uuid.UUID          `json:"id"`
	SiteID           uuid.UUID          `json:"site_id"`
	Topic            string             `json:"topic"`
	TopicHash        string             `json:"topic_hash"`
	Language         string             `json:"language"`
	SearchIntent     SearchIntent       `json:"search_intent"`
	IntentConfidence float64            `json:"intent_confidence"`
	Persona          Persona            `json:"persona"`
	PersonaConfidence float64           `json:"persona_confidence"`
	Audience         string             `json:"audience"`
	Angle            string             `json:"angle"`
	SuggestedTitle   string             `json:"suggested_title"`
	Outline          []OutlineSection   `json:"outline"`
	Questions        []RequiredQuestion `json:"questions"`
	Status           string             `json:"status"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
}

// EditorialReview is the persisted full editorial review.
type EditorialReview struct {
	ID             uuid.UUID       `json:"id"`
	SiteID         uuid.UUID       `json:"site_id"`
	BriefID        *uuid.UUID      `json:"brief_id,omitempty"`
	ArticleID      *uuid.UUID      `json:"article_id,omitempty"`
	ArticleTitle   string          `json:"article_title"`
	ContentHash    string          `json:"content_hash"`
	Scores         EditorialScore  `json:"scores"`
	Coverage       CoverageReport  `json:"coverage"`
	Fluency        FluencyReport   `json:"fluency"`
	Semantic       SemanticReport  `json:"semantic"`
	Blocks         []BlockScore    `json:"blocks"`
	Evidence       []EvidenceLink  `json:"evidence"`
	CreatedAt      time.Time       `json:"created_at"`
}

// EventBus event types emitted by the editorial brain.
const (
	EventBriefCreated  kernel.EventType = "editorial.brief_created"
	EventReviewCreated kernel.EventType = "editorial.review_created"
	EventScoreBlocked  kernel.EventType = "editorial.score_blocked"
)

// Sentinel errors.
var (
	ErrTopicRequired    = errors.New("topic is required")
	ErrContentRequired  = errors.New("content is required")
	ErrInvalidLanguage  = errors.New("language must be pt or en")
	ErrBriefNotFound    = errors.New("editorial brief not found")
	ErrReviewNotFound   = errors.New("editorial review not found")
	ErrDatabaseNotAvail = errors.New("database not available")
	ErrNoResearchData   = errors.New("no research data available")
)

// Score weights of the final editorial note (must sum to 1.0).
const (
	weightSEO         = 0.20
	weightEEAT        = 0.20
	weightFreshness   = 0.15
	weightCoverage    = 0.20
	weightNaturalness = 0.15
	weightConfidence  = 0.10
)

// DefaultMinFinalScore is the default editorial gate threshold.
const DefaultMinFinalScore = 90.0
