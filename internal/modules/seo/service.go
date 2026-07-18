package seo //nolint:gocritic

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/kernel"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type Service struct {
	log      *logger.Logger
	db       *database.Database
	cache    *cache.Cache
	eventBus *kernel.EventBus
	auditLog *audit.Logger
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *Service {
	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}
	return &Service{
		log:      log,
		db:       db,
		cache:    ch,
		auditLog: audit.New(pool, log),
	}
}

func (s *Service) SetEventBus(bus *kernel.EventBus) {
	s.eventBus = bus
}

func (s *Service) fireEvent(ctx context.Context, eventType kernel.EventType, payload interface{}, siteID uuid.UUID) {
	if s.eventBus != nil {
		s.eventBus.EmitAsync(ctx, eventType, payload, siteID.String())
	}
}

func (s *Service) pool() (database.Pool, error) {
	if s.db == nil || s.db.Pool == nil {
		return nil, ErrDatabaseNotAvail
	}
	return s.db.Pool, nil
}

// ============================================================
// Project CRUD
// ============================================================

func (s *Service) CreateProject(ctx context.Context, siteID, userID uuid.UUID, req CreateProjectRequest) (*SEOProject, error) {
	if req.Title == "" {
		return nil, ErrEmptyTitle
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	projID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO seo_projects (id, site_id, user_id, title, target_url, post_id, language,
		 status, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,'pending',$8,$9,$9)`,
		projID, siteID, &userID, req.Title, req.TargetURL, req.PostID, lang, &userID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create seo project: %w", err)
	}

	_, _ = p.Exec(ctx,
		`INSERT INTO seo_scores (id, site_id, seo_project_id, post_id, language, scored_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$6)`,
		uuid.New(), siteID, projID, req.PostID, lang, now,
	)

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     "seo.project.created",
		EntityType: "seo_project",
		EntityID:   &projID,
	})

	return s.GetProject(ctx, siteID, projID)
}

func (s *Service) GetProject(ctx context.Context, siteID, projID uuid.UUID) (*SEOProject, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var proj SEOProject
	err = p.QueryRow(ctx,
		`SELECT id, site_id, user_id, title, COALESCE(target_url,''), post_id,
		        language, status, COALESCE(seo_score,0), COALESCE(readability_score,0),
		        COALESCE(keyword_density,0), COALESCE(content_quality,0),
		        COALESCE(technical_score,0), COALESCE(recommendations,'{}'),
		        started_at, completed_at, COALESCE(error_message,''), created_by,
		        created_at, updated_at
		 FROM seo_projects WHERE id = $1 AND site_id = $2`,
		projID, siteID,
	).Scan(&proj.ID, &proj.SiteID, &proj.UserID, &proj.Title, &proj.TargetURL, &proj.PostID,
		&proj.Language, &proj.Status, &proj.SEOScore, &proj.ReadabilityScore,
		&proj.KeywordDensity, &proj.ContentQuality, &proj.TechnicalScore,
		&proj.Recommendations, &proj.StartedAt, &proj.CompletedAt, &proj.ErrorMessage,
		&proj.CreatedBy, &proj.CreatedAt, &proj.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("failed to get seo project: %w", err)
	}
	return &proj, nil
}

func (s *Service) ListProjects(ctx context.Context, siteID uuid.UUID, status string, limit, offset int) ([]SEOProject, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2

	if status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, user_id, title, COALESCE(target_url,''), post_id,
		        language, status, COALESCE(seo_score,0), COALESCE(readability_score,0),
		        COALESCE(keyword_density,0), COALESCE(content_quality,0),
		        COALESCE(technical_score,0), COALESCE(recommendations,'{}'),
		        started_at, completed_at, COALESCE(error_message,''), created_by,
		        created_at, updated_at
		 FROM seo_projects WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list seo projects: %w", err)
	}
	defer rows.Close()

	var projects []SEOProject
	for rows.Next() {
		var proj SEOProject
		if err := rows.Scan(&proj.ID, &proj.SiteID, &proj.UserID, &proj.Title, &proj.TargetURL, &proj.PostID,
			&proj.Language, &proj.Status, &proj.SEOScore, &proj.ReadabilityScore,
			&proj.KeywordDensity, &proj.ContentQuality, &proj.TechnicalScore,
			&proj.Recommendations, &proj.StartedAt, &proj.CompletedAt, &proj.ErrorMessage,
			&proj.CreatedBy, &proj.CreatedAt, &proj.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan seo project: %w", err)
		}
		projects = append(projects, proj)
	}
	if projects == nil {
		projects = []SEOProject{}
	}
	return projects, nil
}

func (s *Service) DeleteProject(ctx context.Context, siteID, projID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}
	tag, err := p.Exec(ctx,
		`DELETE FROM seo_projects WHERE id = $1 AND site_id = $2`,
		projID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete seo project: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProjectNotFound
	}
	return nil
}

// ============================================================
// Keyword Engine
// ============================================================

func (s *Service) AnalyzeKeywords(ctx context.Context, siteID uuid.UUID, req KeywordAnalysisRequest) ([]Keyword, error) {
	if req.Content == "" {
		return nil, ErrEmptyContent
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}

	words := tokenize(req.Content)
	wordFreq := wordFrequency(words)
	totalWords := len(words)

	var kwList []Keyword

	if req.PrimaryKW != "" {
		freq := wordFreq[strings.ToLower(req.PrimaryKW)]
		density := calcDensity(freq, totalWords)
		kwList = append(kwList, Keyword{
			ID:           uuid.New(),
			SiteID:       siteID,
			Keyword:      req.PrimaryKW,
			KeywordType:  KWPrimary,
			SearchIntent: classifyIntent(req.PrimaryKW),
			Frequency:    freq,
			Density:      density,
			Prominence:   calcProminence(req.Content, req.PrimaryKW, totalWords),
			Language:     lang,
			CreatedAt:    time.Now(),
		})
	}

	for _, sk := range req.SecondaryKW {
		freq := wordFreq[strings.ToLower(sk)]
		density := calcDensity(freq, totalWords)
		kwList = append(kwList, Keyword{
			ID:           uuid.New(),
			SiteID:       siteID,
			Keyword:      sk,
			KeywordType:  KWSecondary,
			SearchIntent: classifyIntent(sk),
			Frequency:    freq,
			Density:      density,
			Prominence:   calcProminence(req.Content, sk, totalWords),
			Language:     lang,
			CreatedAt:    time.Now(),
		})
	}

	semanticKW := extractSemantic(req.Content, req.PrimaryKW, lang)
	for _, sk := range semanticKW {
		freq := wordFreq[strings.ToLower(sk)]
		density := calcDensity(freq, totalWords)
		kwList = append(kwList, Keyword{
			ID:           uuid.New(),
			SiteID:       siteID,
			Keyword:      sk,
			KeywordType:  KWSemantic,
			SearchIntent: IntentInformational,
			Frequency:    freq,
			Density:      density,
			Prominence:   0,
			Language:     lang,
			CreatedAt:    time.Now(),
		})
	}

	entities := extractEntities(req.Content, lang)
	for i := range kwList {
		kwList[i].Entities = entities
	}

	s.fireEvent(ctx, EventSEOKeywords, map[string]interface{}{
		"site_id":      siteID.String(),
		"keyword_count": len(kwList),
	}, siteID)

	return kwList, nil
}

func (s *Service) SaveKeywords(ctx context.Context, projectID uuid.UUID, keywords []Keyword) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	for _, kw := range keywords {
		_, err = p.Exec(ctx,
			`INSERT INTO seo_keywords (id, site_id, seo_project_id, keyword, keyword_type,
			 search_intent, volume, difficulty, density, frequency, prominence, entities, language, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14)`,
			uuid.New(), kw.SiteID, projectID, kw.Keyword, kw.KeywordType,
			kw.SearchIntent, kw.Volume, kw.Difficulty, kw.Density, kw.Frequency,
			kw.Prominence, kw.Entities, kw.Language, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to save keyword: %w", err)
		}
	}
	return nil
}

// ============================================================
// SEO Analyzer
// ============================================================

func (s *Service) AnalyzeSEO(ctx context.Context, siteID uuid.UUID, req SEOAnalysisRequest) (*Audit, error) {
	if req.Content == "" {
		return nil, ErrEmptyContent
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}

	now := time.Now()

	titleScore := scoreTitle(req.Title, req.PrimaryKW)
	metaScore := scoreMetaDescription(req.MetaDesc, req.PrimaryKW)
	headingScore := scoreHeadings(req.Content, req.PrimaryKW)
	paragraphScore := scoreParagraphs(req.Content)
	readScore := scoreReadability(req.Content, lang)
	passiveScore := scorePassiveVoice(req.Content, lang)
	sentenceVarScore := scoreSentenceVariation(req.Content)
	dupScore := scoreDuplicates(req.Content)

	overall := (titleScore*0.15 + metaScore*0.12 + headingScore*0.13 +
		paragraphScore*0.10 + readScore*0.15 + passiveScore*0.08 +
		sentenceVarScore*0.07 + dupScore*0.08) / 0.88

	issues := detectIssues(titleScore, metaScore, headingScore, paragraphScore,
		readScore, passiveScore, sentenceVarScore, dupScore, req.PrimaryKW)

	audit := &Audit{
		ID:                   uuid.New(),
		SiteID:               siteID,
		URL:                  req.Content,
		TitleScore:           titleScore,
		MetaScore:            metaScore,
		HeadingScore:         headingScore,
		ParagraphScore:       paragraphScore,
		ReadabilityScore:     readScore,
		PassiveVoiceScore:    passiveScore,
		SentenceVariationScore: sentenceVarScore,
		DuplicateScore:       dupScore,
		OverallScore:         math.Round(overall*100) / 100,
		Issues:               issues,
		Recommendations:      buildRecommendations(titleScore, metaScore, headingScore, readScore, passiveScore, req.PrimaryKW),
		Language:             lang,
		AuditedAt:            now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}

	s.fireEvent(ctx, EventSEOAnalyzed, map[string]interface{}{
		"site_id":  siteID.String(),
		"score":    audit.OverallScore,
	}, siteID)

	return audit, nil
}

func (s *Service) SaveAudit(ctx context.Context, projectID uuid.UUID, audit *Audit) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	issuesJSON, _ := json.Marshal(audit.Issues)

	_, err = p.Exec(ctx,
		`INSERT INTO seo_audits (id, site_id, seo_project_id, post_id, url,
		 title_score, meta_score, heading_score, paragraph_score,
		 readability_score, passive_voice_score, sentence_variation_score,
		 duplicate_score, overall_score, issues, recommendations, language, audited_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15::jsonb,$16,$17,$18,$18,$18)`,
		audit.ID, audit.SiteID, projectID, audit.PostID, audit.URL,
		audit.TitleScore, audit.MetaScore, audit.HeadingScore, audit.ParagraphScore,
		audit.ReadabilityScore, audit.PassiveVoiceScore, audit.SentenceVariationScore,
		audit.DuplicateScore, audit.OverallScore, string(issuesJSON),
		audit.Recommendations, audit.Language, audit.AuditedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save audit: %w", err)
	}
	return nil
}

// ============================================================
// Internal Linking
// ============================================================

func (s *Service) SuggestLinks(ctx context.Context, siteID uuid.UUID, req LinkSuggestionRequest) ([]InternalLink, error) {
	if req.Content == "" {
		return nil, ErrEmptyContent
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}

	words := tokenize(req.Content)
	wordFreq := wordFrequency(words)

	suggestedKW := make([]string, 0, 10)
	for w, f := range wordFreq {
		if f >= 2 && len(w) > 3 && len(suggestedKW) < 10 {
			suggestedKW = append(suggestedKW, w)
		}
	}

	var links []InternalLink
	for _, kw := range suggestedKW {
		links = append(links, InternalLink{
			ID:         uuid.New(),
			SiteID:     siteID,
			SourceURL:  req.SiteURL,
			AnchorText: kw,
			LinkType:   LinkSuggestion,
			Relevance:  math.Round(float64(wordFreq[kw])/float64(len(words))*1000) / 10,
			Language:   lang,
			CreatedAt:  time.Now(),
		})
	}

	s.fireEvent(ctx, EventSEOLinks, map[string]interface{}{
		"site_id":    siteID.String(),
		"link_count": len(links),
	}, siteID)

	return links, nil
}

func (s *Service) SaveLinks(ctx context.Context, projectID uuid.UUID, links []InternalLink) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	for _, l := range links {
		_, err = p.Exec(ctx,
			`INSERT INTO seo_internal_links (id, site_id, seo_project_id, source_url, target_url,
			 anchor_text, link_type, relevance, is_existing, is_implemented, language, created_at, updated_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$12)`,
			uuid.New(), l.SiteID, projectID, l.SourceURL, l.TargetURL,
			l.AnchorText, l.LinkType, l.Relevance, l.IsExisting, l.IsImplemented, l.Language, time.Now(),
		)
		if err != nil {
			return fmt.Errorf("failed to save link: %w", err)
		}
	}
	return nil
}

// ============================================================
// Technical SEO
// ============================================================

func (s *Service) GenerateTechnicalSEO(ctx context.Context, siteID uuid.UUID, req TechnicalSEORequest) (*Metadata, error) {
	if req.Title == "" {
		return nil, ErrEmptyTitle
	}
	lang := req.Language
	if lang == "" {
		lang = "pt"
	}

	now := time.Now()

	hreflang := []map[string]interface{}{
		{"rel": "canonical", "href": req.URL, "hreflang": lang},
	}
	if req.AltLang != "" {
		altURL := strings.Replace(req.URL, "/"+lang+"/", "/"+req.AltLang+"/", 1)
		if altURL == req.URL {
			altURL = req.URL + "?lang=" + req.AltLang
		}
		hreflang = append(hreflang, map[string]interface{}{
			"rel": "alternate", "href": altURL, "hreflang": req.AltLang,
		})
	}

	fullURL := req.URL
	if !strings.HasPrefix(fullURL, "http") {
		fullURL = "https://" + fullURL
	}

	articleSchema := map[string]interface{}{
		"@context":       "https://schema.org",
		"@type":          "Article",
		"headline":       req.Title,
		"description":    req.MetaDesc,
		"url":            fullURL,
		"mainEntityOfPage": fullURL,
		"datePublished":  now.Format(time.RFC3339),
		"dateModified":   now.Format(time.RFC3339),
	}
	if req.ImageURL != "" {
		articleSchema["image"] = req.ImageURL
	}
	if req.Author != "" {
		articleSchema["author"] = map[string]interface{}{
			"@type": "Person",
			"name":  req.Author,
		}
	}
	if req.SiteName != "" {
		articleSchema["publisher"] = map[string]interface{}{
			"@type": "Organization",
			"name":  req.SiteName,
		}
	}

	var faqSchema []map[string]interface{}
	for _, faq := range req.FAQs {
		q, _ := faq["question"]
		a, _ := faq["answer"]
		faqSchema = append(faqSchema, map[string]interface{}{
			"@type": "Question",
			"name":  q,
			"acceptedAnswer": map[string]interface{}{
				"@type": "Answer",
				"text":  a,
			},
		})
	}

	breadcrumbSchema := []map[string]interface{}{
		{"@type": "ListItem", "position": 1, "name": "Home", "item": fullURL},
	}
	for i, cat := range req.Categories {
		breadcrumbSchema = append(breadcrumbSchema, map[string]interface{}{
			"@type": "ListItem",
			"position": i + 2,
			"name":  cat,
		})
	}

	meta := &Metadata{
		ID:               uuid.New(),
		SiteID:           siteID,
		TitleTag:         buildTitleTag(req.Title, req.SiteName),
		MetaDescription:  truncateStr(req.MetaDesc, 160),
		CanonicalURL:     fullURL,
		OGTitle:          req.Title,
		OGDescription:    truncateStr(req.MetaDesc, 200),
		OGImage:          req.ImageURL,
		TwitterTitle:     req.Title,
		TwitterDescription: truncateStr(req.MetaDesc, 200),
		TwitterImage:     req.ImageURL,
		JSONLD:           articleSchema,
		FAQSchema:        faqSchema,
		BreadcrumbSchema: breadcrumbSchema,
		ArticleSchema:    articleSchema,
		Hreflang:         hreflang,
		RobotsDirectives: []string{"index", "follow"},
		Language:         lang,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	s.fireEvent(ctx, EventSEOMetadataGen, map[string]interface{}{
		"site_id": siteID.String(),
	}, siteID)

	return meta, nil
}

func (s *Service) SaveMetadata(ctx context.Context, projectID uuid.UUID, meta *Metadata) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	jsonldJSON, _ := json.Marshal(meta.JSONLD)
	faqJSON, _ := json.Marshal(meta.FAQSchema)
	bcJSON, _ := json.Marshal(meta.BreadcrumbSchema)
	articleJSON, _ := json.Marshal(meta.ArticleSchema)
	hreflangJSON, _ := json.Marshal(meta.Hreflang)

	_, err = p.Exec(ctx,
		`INSERT INTO seo_metadata (id, site_id, seo_project_id, post_id,
		 title_tag, meta_description, canonical_url,
		 og_title, og_description, og_image,
		 twitter_title, twitter_description, twitter_image,
		 json_ld, faq_schema, breadcrumb_schema, article_schema, hreflang,
		 robots_directives, language, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15::jsonb,$16::jsonb,$17::jsonb,$18::jsonb,$19,$20,$21,$21)`,
		meta.ID, meta.SiteID, projectID, meta.PostID,
		meta.TitleTag, meta.MetaDescription, meta.CanonicalURL,
		meta.OGTitle, meta.OGDescription, meta.OGImage,
		meta.TwitterTitle, meta.TwitterDescription, meta.TwitterImage,
		string(jsonldJSON), string(faqJSON), string(bcJSON), string(articleJSON), string(hreflangJSON),
		meta.RobotsDirectives, meta.Language, meta.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save metadata: %w", err)
	}
	return nil
}

// ============================================================
// Content Scoring
// ============================================================

func (s *Service) ScoreContent(ctx context.Context, keywords []Keyword, audit *Audit, meta *Metadata, links []InternalLink) *ContentScoreResult {
	kwScore := calcKeywordScore(keywords)
	contentScore := calcContentScore(audit)
	techScore := calcTechnicalScore(meta)
	linkScore := calcLinkingScore(links)
	readScore := audit.ReadabilityScore
	metaScore := calcMetaScore(meta)

	total := math.Round((kwScore*0.20 + contentScore*0.25 + techScore*0.15 +
		linkScore*0.10 + readScore*0.15 + metaScore*0.15) * 100) / 100

	recs := generateScoreRecommendations(kwScore, contentScore, techScore, linkScore, readScore, metaScore)

	result := &ContentScoreResult{
		TotalScore:       total,
		KeywordScore:     kwScore,
		ContentScore:     contentScore,
		TechnicalScore:   techScore,
		LinkingScore:     linkScore,
		ReadabilityScore: readScore,
		MetadataScore:    metaScore,
		Breakdown: map[string]float64{
			"keyword":    kwScore,
			"content":    contentScore,
			"technical":  techScore,
			"linking":    linkScore,
			"readability": readScore,
			"metadata":   metaScore,
		},
		Recommendations: recs,
	}

	return result
}

func (s *Service) SaveScore(ctx context.Context, projectID uuid.UUID, result *ContentScoreResult, lang string) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`UPDATE seo_scores SET total_score = $1, keyword_score = $2, content_score = $3,
		 technical_score = $4, linking_score = $5, readability_score = $6,
		 metadata_score = $7, scored_at = $8
		 WHERE seo_project_id = $9`,
		result.TotalScore, result.KeywordScore, result.ContentScore,
		result.TechnicalScore, result.LinkingScore, result.ReadabilityScore,
		result.MetadataScore, time.Now(), projectID,
	)
	if err != nil {
		return fmt.Errorf("failed to save score: %w", err)
	}

	_, err = p.Exec(ctx,
		`UPDATE seo_projects SET seo_score = $1, readability_score = $2,
		 content_quality = $3, technical_score = $4, updated_at = $5
		 WHERE id = $6`,
		result.TotalScore, result.ReadabilityScore, result.ContentScore,
		result.TechnicalScore, time.Now(), projectID,
	)
	if err != nil {
		return fmt.Errorf("failed to update project scores: %w", err)
	}

	s.fireEvent(ctx, EventSEOScored, map[string]interface{}{
		"project_id": projectID.String(),
		"score":      result.TotalScore,
	}, uuid.Nil)

	return nil
}

// ============================================================
// Dashboard
// ============================================================

func (s *Service) GetDashboard(ctx context.Context, siteID uuid.UUID) (*DashboardResponse, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	projects, _ := s.ListProjects(ctx, siteID, "", 5, 0)

	var topIssues []string
	_ = p.QueryRow(ctx,
		`SELECT COALESCE(
			(SELECT recommendations[1:5] FROM seo_audits
			 WHERE site_id = $1 AND array_length(recommendations,1) > 0
			 ORDER BY created_at DESC LIMIT 1), '{}'::text[])`,
		siteID,
	).Scan(&topIssues)

	var avgSEO, avgRead float64
	var pending, completedToday int

	_ = p.QueryRow(ctx,
		`SELECT COALESCE(AVG(seo_score),0), COALESCE(AVG(readability_score),0) FROM seo_projects WHERE site_id = $1`,
		siteID,
	).Scan(&avgSEO, &avgRead)

	_ = p.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END),0),
		        COALESCE(SUM(CASE WHEN status = 'completed' AND completed_at::date = CURRENT_DATE THEN 1 ELSE 0 END),0)
		 FROM seo_projects WHERE site_id = $1`,
		siteID,
	).Scan(&pending, &completedToday)

	return &DashboardResponse{
		Projects:        projects,
		TopIssues:       topIssues,
		AvgSEOScore:     avgSEO,
		AvgReadability:  avgRead,
		PendingProjects: pending,
		CompletedToday:  completedToday,
	}, nil
}

func (s *Service) GetMetrics(ctx context.Context, siteID uuid.UUID) (*SEOMetrics, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var m SEOMetrics
	err = p.QueryRow(ctx,
		`SELECT COALESCE(COUNT(*),0),
		        COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END),0),
		        COALESCE(AVG(seo_score),0),
		        COALESCE(AVG(content_quality),0),
		        COALESCE(AVG(readability_score),0),
		        COALESCE(AVG(technical_score),0),
		        (SELECT COALESCE(COUNT(*),0) FROM seo_keywords WHERE site_id = $1),
		        (SELECT COALESCE(COUNT(*),0) FROM seo_internal_links WHERE site_id = $1)
		 FROM seo_projects WHERE site_id = $1`,
		siteID,
	).Scan(&m.TotalProjects, &m.CompletedProjects, &m.AvgScore,
		&m.AvgKeywordScore, &m.AvgReadability, &m.AvgTechnical,
		&m.TotalKeywords, &m.TotalLinks)
	if err != nil {
		return nil, fmt.Errorf("failed to get seo metrics: %w", err)
	}
	return &m, nil
}

// ============================================================
// Scoring helpers
// ============================================================

func scoreTitle(title, primaryKW string) float64 {
	if title == "" {
		return 0
	}
	score := 60.0
	length := len([]rune(title))
	if length >= 30 && length <= 60 {
		score += 20
	} else if length < 30 {
		score += 5
	} else {
		score -= 10
	}
	if primaryKW != "" && strings.Contains(strings.ToLower(title), strings.ToLower(primaryKW)) {
		score += 20
	}
	return math.Min(100, math.Max(0, score))
}

func scoreMetaDescription(meta, primaryKW string) float64 {
	if meta == "" {
		return 0
	}
	score := 50.0
	length := len([]rune(meta))
	if length >= 120 && length <= 160 {
		score += 30
	} else if length > 160 {
		score += 10
	} else if length >= 80 {
		score += 15
	}
	if primaryKW != "" && strings.Contains(strings.ToLower(meta), strings.ToLower(primaryKW)) {
		score += 20
	}
	return math.Min(100, math.Max(0, score))
}

func scoreHeadings(content, primaryKW string) float64 {
	h2Count := strings.Count(content, "\n## ")
	h3Count := strings.Count(content, "\n### ")
	total := h2Count + h3Count
	if total == 0 {
		return 20
	}
	score := 50.0
	if total >= 3 && total <= 8 {
		score += 30
	} else if total >= 1 {
		score += 15
	}
	if primaryKW != "" {
		for _, line := range strings.Split(content, "\n") {
			if strings.HasPrefix(line, "#") && strings.Contains(strings.ToLower(line), strings.ToLower(primaryKW)) {
				score += 20
				break
			}
		}
	}
	return math.Min(100, math.Max(0, score))
}

func scoreParagraphs(content string) float64 {
	paragraphs := strings.Split(content, "\n\n")
	if len(paragraphs) == 0 {
		return 30
	}
	longEnough := 0
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if len([]rune(trimmed)) >= 50 && len([]rune(trimmed)) <= 300 {
			longEnough++
		}
	}
	ratio := float64(longEnough) / float64(len(paragraphs))
	return math.Min(100, ratio*100)
}

func scoreReadability(content, lang string) float64 {
	sentences := splitSentences(content)
	if len(sentences) == 0 {
		return 50
	}
	totalSyllables := 0
	totalWords := 0
	for _, s := range sentences {
		words := tokenize(s)
		totalWords += len(words)
		for _, w := range words {
			totalSyllables += countSyllables(w)
		}
	}
	if totalWords == 0 {
		return 50
	}
	avgSyllables := float64(totalSyllables) / float64(totalWords)
	avgWordsPerSentence := float64(totalWords) / float64(len(sentences))

	score := 100.0
	if avgSyllables > 1.8 {
		score -= 20
	} else if avgSyllables > 1.5 {
		score -= 10
	}
	if avgWordsPerSentence > 25 {
		score -= 15
	} else if avgWordsPerSentence > 20 {
		score -= 5
	} else if avgWordsPerSentence < 8 {
		score -= 10
	}
	return math.Max(0, score)
}

func scorePassiveVoice(content, lang string) float64 {
	passivePatterns := []string{
		"was ", "were ", "been ", "being ",
		"is ", "are ", "was ", "were ",
	}
	if lang == "pt" {
		passivePatterns = []string{
			"foi ", "foram ", "era ", "eram ",
			"ser ", "sido ", "sendo ",
		}
	}
	sentences := splitSentences(content)
	if len(sentences) == 0 {
		return 80
	}
	passiveCount := 0
	for _, s := range sentences {
		lower := strings.ToLower(s)
		for _, p := range passivePatterns {
			if strings.Contains(lower, p) {
				passiveCount++
				break
			}
		}
	}
	ratio := float64(passiveCount) / float64(len(sentences))
	if ratio <= 0.1 {
		return 100
	}
	if ratio <= 0.2 {
		return 80
	}
	if ratio <= 0.3 {
		return 60
	}
	return 40
}

func scoreSentenceVariation(content string) float64 {
	sentences := splitSentences(content)
	if len(sentences) < 3 {
		return 50
	}
	lengths := make([]int, len(sentences))
	for i, s := range sentences {
		lengths[i] = len(tokenize(s))
	}
	if len(lengths) < 2 {
		return 50
	}
	variation := 0
	for i := 1; i < len(lengths); i++ {
		diff := lengths[i] - lengths[i-1]
		if diff < 0 {
			diff = -diff
		}
		if diff >= 3 && diff <= 15 {
			variation++
		}
	}
	ratio := float64(variation) / float64(len(lengths)-1)
	return math.Min(100, ratio*100)
}

func scoreDuplicates(content string) float64 {
	paragraphs := strings.Split(content, "\n\n")
	if len(paragraphs) < 2 {
		return 100
	}
	seen := make(map[string]bool)
	dups := 0
	for _, p := range paragraphs {
		trimmed := strings.TrimSpace(p)
		if len(trimmed) < 20 {
			continue
		}
		key := strings.ToLower(trimmed)
		if seen[key] {
			dups++
		}
		seen[key] = true
	}
	if dups == 0 {
		return 100
	}
	ratio := float64(dups) / float64(len(paragraphs))
	return math.Max(0, 100-ratio*200)
}

func detectIssues(titleScore, metaScore, headingScore, paragraphScore, readScore, passiveScore, sentenceVarScore, dupScore float64, primaryKW string) []map[string]interface{} {
	var issues []map[string]interface{}
	addIssue := func(category, severity, message string, score float64) {
		issues = append(issues, map[string]interface{}{
			"category": category,
			"severity": severity,
			"message":  message,
			"score":    score,
		})
	}
	if titleScore < 60 {
		addIssue("title", "high", "Title is too short, too long, or missing primary keyword", titleScore)
	}
	if metaScore < 50 {
		addIssue("meta", "high", "Meta description is missing or too short", metaScore)
	}
	if headingScore < 40 {
		addIssue("headings", "medium", "Content lacks proper heading hierarchy (H2/H3)", headingScore)
	}
	if paragraphScore < 50 {
		addIssue("paragraphs", "medium", "Paragraphs are too short or too long", paragraphScore)
	}
	if readScore < 50 {
		addIssue("readability", "high", "Content readability needs improvement", readScore)
	}
	if passiveScore < 60 {
		addIssue("passive_voice", "low", "Excessive use of passive voice", passiveScore)
	}
	if sentenceVarScore < 50 {
		addIssue("sentence_variation", "low", "Sentence lengths are too uniform", sentenceVarScore)
	}
	if dupScore < 80 {
		addIssue("duplicates", "medium", "Duplicate paragraphs detected", dupScore)
	}
	if issues == nil {
		issues = []map[string]interface{}{}
	}
	return issues
}

func buildRecommendations(titleScore, metaScore, headingScore, readScore, passiveScore float64, primaryKW string) []string {
	var recs []string
	if titleScore < 70 {
		recs = append(recs, "Optimize title: keep 30-60 characters and include primary keyword")
	}
	if metaScore < 60 {
		recs = append(recs, "Write a meta description between 120-160 characters with primary keyword")
	}
	if headingScore < 50 {
		recs = append(recs, "Add more H2/H3 headings to structure content and include keywords")
	}
	if readScore < 60 {
		recs = append(recs, "Improve readability: use shorter sentences and simpler words")
	}
	if passiveScore < 70 {
		recs = append(recs, "Reduce passive voice usage for more direct content")
	}
	return recs
}

func generateScoreRecommendations(kwScore, contentScore, techScore, linkScore, readScore, metaScore float64) []string {
	var recs []string
	if kwScore < 60 {
		recs = append(recs, "Improve keyword usage: increase primary keyword density and add semantic keywords")
	}
	if contentScore < 60 {
		recs = append(recs, "Enhance content quality: add more headings, improve paragraph structure")
	}
	if techScore < 60 {
		recs = append(recs, "Add structured data markup (Article schema, FAQ schema, breadcrumbs)")
	}
	if linkScore < 50 {
		recs = append(recs, "Add more internal links to related content")
	}
	if readScore < 60 {
		recs = append(recs, "Improve readability for better user engagement")
	}
	if metaScore < 60 {
		recs = append(recs, "Optimize metadata: title tag, meta description, OG tags")
	}
	if len(recs) == 0 {
		recs = append(recs, "Excellent SEO score — maintain current optimization")
	}
	return recs
}

func calcKeywordScore(keywords []Keyword) float64 {
	if len(keywords) == 0 {
		return 20
	}
	score := 50.0
	hasPrimary := false
	for _, kw := range keywords {
		if kw.KeywordType == KWPrimary {
			hasPrimary = true
			if kw.Density >= 0.5 && kw.Density <= 3.0 {
				score += 20
			} else if kw.Density > 0 {
				score += 5
			}
			if kw.Prominence > 0.5 {
				score += 10
			}
		}
		if kw.KeywordType == KWSecondary {
			score += 5
		}
		if kw.KeywordType == KWSemantic {
			score += 3
		}
	}
	if !hasPrimary {
		score -= 20
	}
	return math.Min(100, math.Max(0, score))
}

func calcContentScore(audit *Audit) float64 {
	if audit == nil {
		return 0
	}
	return (audit.HeadingScore*0.25 + audit.ParagraphScore*0.25 +
		audit.PassiveVoiceScore*0.15 + audit.SentenceVariationScore*0.15 +
		audit.DuplicateScore*0.20)
}

func calcTechnicalScore(meta *Metadata) float64 {
	if meta == nil {
		return 0
	}
	score := 30.0
	if meta.CanonicalURL != "" {
		score += 10
	}
	if meta.OGTitle != "" {
		score += 10
	}
	if meta.TwitterTitle != "" {
		score += 5
	}
	if meta.ArticleSchema != nil && len(meta.ArticleSchema) > 0 {
		score += 20
	}
	if len(meta.FAQSchema) > 0 {
		score += 10
	}
	if len(meta.BreadcrumbSchema) > 0 {
		score += 10
	}
	if len(meta.Hreflang) > 0 {
		score += 5
	}
	return math.Min(100, score)
}

func calcLinkingScore(links []InternalLink) float64 {
	if len(links) == 0 {
		return 20
	}
	score := 30.0
	implemented := 0
	for _, l := range links {
		if l.IsImplemented {
			implemented++
		}
		if l.Relevance > 60 {
			score += 5
		}
	}
	if implemented > 0 {
		score += float64(implemented) * 10
	}
	return math.Min(100, score)
}

func calcMetaScore(meta *Metadata) float64 {
	if meta == nil {
		return 0
	}
	score := 20.0
	if meta.TitleTag != "" {
		score += 15
	}
	if meta.MetaDescription != "" {
		score += 15
	}
	if meta.OGTitle != "" {
		score += 10
	}
	if meta.OGDescription != "" {
		score += 10
	}
	if meta.TwitterTitle != "" {
		score += 5
	}
	if meta.TwitterDescription != "" {
		score += 5
	}
	if meta.CanonicalURL != "" {
		score += 10
	}
	if len(meta.RobotsDirectives) > 0 {
		score += 10
	}
	return math.Min(100, score)
}

// ============================================================
// Text utilities
// ============================================================

func tokenize(text string) []string {
	var words []string
	current := strings.Builder{}
	for _, r := range text {
		if unicode.IsLetter(r) || r == '\'' {
			current.WriteRune(unicode.ToLower(r))
		} else if current.Len() > 0 {
			words = append(words, current.String())
			current.Reset()
		}
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

func wordFrequency(words []string) map[string]int {
	freq := make(map[string]int)
	for _, w := range words {
		freq[w]++
	}
	return freq
}

func calcDensity(freq, total int) float64 {
	if total == 0 {
		return 0
	}
	return math.Round(float64(freq)/float64(total)*10000) / 100
}

func calcProminence(content, keyword string, totalWords int) float64 {
	lower := strings.ToLower(content)
	kw := strings.ToLower(keyword)
	idx := strings.Index(lower, kw)
	if idx == -1 {
		return 0
	}
	before := lower[:idx]
	beforeWords := len(tokenize(before))
	if totalWords == 0 {
		return 0
	}
	return math.Max(0, 100-float64(beforeWords)/float64(totalWords)*100)
}

func classifyIntent(keyword string) SearchIntent {
	lower := strings.ToLower(keyword)
	transactional := []string{"comprar", "buy", "preço", "price", "order", "encomendar", "assinar", "subscribe"}
	navigational := []string{"login", "sign in", "entrar", "site", "website", "home"}
	commercial := []string{"best", "melhor", "review", "avaliação", "comparar", "compare", "top", "vs"}

	for _, w := range transactional {
		if strings.Contains(lower, w) {
			return IntentTransactional
		}
	}
	for _, w := range navigational {
		if strings.Contains(lower, w) {
			return IntentNavigational
		}
	}
	for _, w := range commercial {
		if strings.Contains(lower, w) {
			return IntentCommercial
		}
	}
	return IntentInformational
}

func extractSemantic(content, primaryKW, lang string) []string {
	words := tokenize(content)
	stopWords := stopWordsEN
	if lang == "pt" {
		stopWords = stopWordsPT
	}
	stopSet := make(map[string]bool)
	for _, sw := range stopWords {
		stopSet[sw] = true
	}
	freq := wordFrequency(words)

	var semantic []string
	for w, f := range freq {
		if f >= 2 && len(w) > 3 && !stopSet[w] && w != strings.ToLower(primaryKW) {
			semantic = append(semantic, w)
			if len(semantic) >= 10 {
				break
			}
		}
	}
	return semantic
}

func extractEntities(content, lang string) []string {
	words := tokenize(content)
	var entities []string
	for _, w := range words {
		if len(w) > 1 && unicode.IsUpper([]rune(w)[0]) {
			entities = append(entities, w)
		}
	}
	if entities == nil {
		entities = []string{}
	}
	return entities
}

var stopWordsEN = []string{
	"the", "a", "an", "and", "or", "but", "in", "on", "at", "to", "for",
	"of", "with", "by", "from", "is", "are", "was", "were", "be", "been",
	"being", "have", "has", "had", "do", "does", "did", "will", "would",
	"can", "could", "may", "might", "shall", "should", "this", "that",
}

var stopWordsPT = []string{
	"o", "a", "os", "as", "um", "uma", "uns", "umas", "de", "da", "do",
	"das", "dos", "em", "no", "na", "nos", "nas", "para", "por", "com",
	"sem", "e", "ou", "mas", "que", "se", "como", "mais", "muito", "bem",
	"ao", "aos", "às", "é", "são", "era", "ser", "está", "estão",
}

func splitSentences(text string) []string {
	re := regexp.MustCompile(`[.!?]+`)
	parts := re.Split(text, -1)
	var sentences []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			sentences = append(sentences, trimmed)
		}
	}
	if sentences == nil {
		sentences = []string{}
	}
	return sentences
}

func countSyllables(word string) int {
	word = strings.ToLower(word)
	vowels := "aeiouáéíóúâêîôûãõàèìòù"
	count := 0
	for _, r := range word {
		if strings.ContainsRune(vowels, r) {
			count++
		}
	}
	if count == 0 {
		return 1
	}
	return count
}

func buildTitleTag(title, siteName string) string {
	if siteName != "" {
		return title + " | " + siteName
	}
	return title
}

func truncateStr(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen])
}

// ============================================================
// Sitemap Generation
// ============================================================

func (s *Service) GenerateXMLSitemap(ctx context.Context, siteID uuid.UUID, baseURL string) (string, error) {
	p, err := s.pool()
	if err != nil {
		return "", err
	}

	rows, err := p.Query(ctx,
		`SELECT id, updated_at FROM seo_projects
		 WHERE site_id = $1 AND status = 'completed' ORDER BY updated_at DESC`,
		siteID,
	)
	if err != nil {
		return "", fmt.Errorf("failed to query projects for sitemap: %w", err)
	}
	defer rows.Close()

	var sb strings.Builder
	sb.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	sb.WriteString(`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")

	for rows.Next() {
		var id uuid.UUID
		var updatedAt time.Time
		if err := rows.Scan(&id, &updatedAt); err != nil {
			continue
		}
		url := strings.TrimRight(baseURL, "/") + "/seo/" + id.String()
		sb.WriteString(fmt.Sprintf("  <url>\n    <loc>%s</loc>\n    <lastmod>%s</lastmod>\n    <changefreq>weekly</changefreq>\n    <priority>0.8</priority>\n  </url>\n",
			url, updatedAt.Format("2006-01-02")))
	}

	sb.WriteString(`</urlset>`)
	return sb.String(), nil
}
