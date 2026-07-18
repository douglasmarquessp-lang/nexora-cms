package humanwriter

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"time"

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

func coalesceStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func coalesceInt(i, def int) int {
	if i == 0 {
		return def
	}
	return i
}

// --- Writing Profiles ---

func (s *Service) CreateProfile(ctx context.Context, siteID uuid.UUID, req ProfileCreateRequest) (*WritingProfile, error) {
	if req.Slug == "" {
		return nil, ErrInvalidSlug
	}
	if req.Name == "" {
		return nil, fmt.Errorf("profile name is required")
	}
	lang := coalesceStr(req.Language, "pt")
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var exists bool
	err = p.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM writing_profiles WHERE site_id = $1 AND slug = $2)`,
		siteID, req.Slug,
	).Scan(&exists)
	if err != nil {
		return nil, fmt.Errorf("failed to check profile existence: %w", err)
	}
	if exists {
		return nil, ErrProfileExists
	}

	now := time.Now()
	profileID := uuid.New()

	_, err = p.Exec(ctx,
		`INSERT INTO writing_profiles (id, site_id, slug, name, description, tone, perspective,
		 audience, expertise_level, language, vocabulary_tags, allowed_connectors,
		 preferred_sentence_length, paragraph_size_min, paragraph_size_max, metadata, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16::jsonb,$17,$17)`,
		profileID, siteID, req.Slug, req.Name, req.Description, req.Tone, req.Perspective,
		req.Audience, req.ExpertiseLevel, lang, req.VocabularyTags, req.AllowedConnectors,
		req.PreferredSentenceLength, coalesceInt(req.ParagraphSizeMin, 3),
		coalesceInt(req.ParagraphSizeMax, 8), toJSON(req.Metadata), now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create profile: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		Action:     audit.Action("humanwriter.profile.created"),
		EntityType: "writing_profile",
		EntityID:   &profileID,
		Payload:    map[string]interface{}{"slug": req.Slug, "name": req.Name},
	})

	s.fireEvent(ctx, EventProfileCreated, map[string]interface{}{
		"profile_id": profileID.String(),
		"slug":       req.Slug,
		"site_id":    siteID.String(),
	}, siteID)

	return s.GetProfile(ctx, siteID, profileID)
}

func (s *Service) GetProfile(ctx context.Context, siteID, profileID uuid.UUID) (*WritingProfile, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var prof WritingProfile
	var metadataStr string
	err = p.QueryRow(ctx,
		`SELECT id, site_id, slug, name, COALESCE(description,''), COALESCE(tone,''),
		        COALESCE(perspective,''), COALESCE(audience,''), COALESCE(expertise_level,'general'),
		        language, COALESCE(vocabulary_tags,'{}'), COALESCE(allowed_connectors,'{}'),
		        COALESCE(preferred_sentence_length,'medium'), COALESCE(paragraph_size_min,3),
		        COALESCE(paragraph_size_max,8), is_active, COALESCE(metadata::text,'{}'),
		        created_by, created_at, updated_at
		 FROM writing_profiles WHERE id = $1 AND site_id = $2`,
		profileID, siteID,
	).Scan(&prof.ID, &prof.SiteID, &prof.Slug, &prof.Name, &prof.Description, &prof.Tone,
		&prof.Perspective, &prof.Audience, &prof.ExpertiseLevel, &prof.Language,
		&prof.VocabularyTags, &prof.AllowedConnectors, &prof.PreferredSentenceLength,
		&prof.ParagraphSizeMin, &prof.ParagraphSizeMax, &prof.IsActive,
		&metadataStr, &prof.CreatedBy, &prof.CreatedAt, &prof.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}
	if len(metadataStr) > 0 {
		prof.Metadata = parseJSON(metadataStr)
	}
	if prof.Metadata == nil {
		prof.Metadata = make(map[string]interface{})
	}
	return &prof, nil
}

func (s *Service) GetProfileBySlug(ctx context.Context, siteID uuid.UUID, slug string) (*WritingProfile, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var prof WritingProfile
	var metadataStr string
	err = p.QueryRow(ctx,
		`SELECT id, site_id, slug, name, COALESCE(description,''), COALESCE(tone,''),
		        COALESCE(perspective,''), COALESCE(audience,''), COALESCE(expertise_level,'general'),
		        language, COALESCE(vocabulary_tags,'{}'), COALESCE(allowed_connectors,'{}'),
		        COALESCE(preferred_sentence_length,'medium'), COALESCE(paragraph_size_min,3),
		        COALESCE(paragraph_size_max,8), is_active, COALESCE(metadata::text,'{}'),
		        created_by, created_at, updated_at
		 FROM writing_profiles WHERE site_id = $1 AND slug = $2`,
		siteID, slug,
	).Scan(&prof.ID, &prof.SiteID, &prof.Slug, &prof.Name, &prof.Description, &prof.Tone,
		&prof.Perspective, &prof.Audience, &prof.ExpertiseLevel, &prof.Language,
		&prof.VocabularyTags, &prof.AllowedConnectors, &prof.PreferredSentenceLength,
		&prof.ParagraphSizeMin, &prof.ParagraphSizeMax, &prof.IsActive,
		&metadataStr, &prof.CreatedBy, &prof.CreatedAt, &prof.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrProfileNotFound
		}
		return nil, fmt.Errorf("failed to get profile by slug: %w", err)
	}
	if len(metadataStr) > 0 {
		prof.Metadata = parseJSON(metadataStr)
	}
	if prof.Metadata == nil {
		prof.Metadata = make(map[string]interface{})
	}
	return &prof, nil
}

func (s *Service) ListProfiles(ctx context.Context, siteID uuid.UUID, language string) ([]WritingProfile, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2
	if language != "" {
		where = append(where, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, language)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, slug, name, COALESCE(description,''), COALESCE(tone,''),
		        COALESCE(perspective,''), COALESCE(audience,''), COALESCE(expertise_level,'general'),
		        language, COALESCE(vocabulary_tags,'{}'), COALESCE(allowed_connectors,'{}'),
		        COALESCE(preferred_sentence_length,'medium'), COALESCE(paragraph_size_min,3),
		        COALESCE(paragraph_size_max,8), is_active, created_by, created_at, updated_at
		 FROM writing_profiles WHERE %s ORDER BY name ASC`, strings.Join(where, " AND "),
	)
	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}
	defer rows.Close()

	var profiles []WritingProfile
	for rows.Next() {
		var prof WritingProfile
		if err := rows.Scan(&prof.ID, &prof.SiteID, &prof.Slug, &prof.Name, &prof.Description,
			&prof.Tone, &prof.Perspective, &prof.Audience, &prof.ExpertiseLevel, &prof.Language,
			&prof.VocabularyTags, &prof.AllowedConnectors, &prof.PreferredSentenceLength,
			&prof.ParagraphSizeMin, &prof.ParagraphSizeMax, &prof.IsActive,
			&prof.CreatedBy, &prof.CreatedAt, &prof.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan profile: %w", err)
		}
		profiles = append(profiles, prof)
	}
	if profiles == nil {
		profiles = []WritingProfile{}
	}
	return profiles, nil
}

func (s *Service) UpdateProfile(ctx context.Context, siteID, profileID uuid.UUID, req ProfileUpdateRequest) (*WritingProfile, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetProfile(ctx, siteID, profileID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Name != nil {
		setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *req.Name)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}
	if req.Tone != nil {
		setClauses = append(setClauses, fmt.Sprintf("tone = $%d", argIdx))
		args = append(args, *req.Tone)
		argIdx++
	}
	if req.Perspective != nil {
		setClauses = append(setClauses, fmt.Sprintf("perspective = $%d", argIdx))
		args = append(args, *req.Perspective)
		argIdx++
	}
	if req.Audience != nil {
		setClauses = append(setClauses, fmt.Sprintf("audience = $%d", argIdx))
		args = append(args, *req.Audience)
		argIdx++
	}
	if req.ExpertiseLevel != nil {
		setClauses = append(setClauses, fmt.Sprintf("expertise_level = $%d", argIdx))
		args = append(args, *req.ExpertiseLevel)
		argIdx++
	}
	if req.VocabularyTags != nil {
		setClauses = append(setClauses, fmt.Sprintf("vocabulary_tags = $%d", argIdx))
		args = append(args, *req.VocabularyTags)
		argIdx++
	}
	if req.AllowedConnectors != nil {
		setClauses = append(setClauses, fmt.Sprintf("allowed_connectors = $%d", argIdx))
		args = append(args, *req.AllowedConnectors)
		argIdx++
	}
	if req.PreferredSentenceLength != nil {
		setClauses = append(setClauses, fmt.Sprintf("preferred_sentence_length = $%d", argIdx))
		args = append(args, *req.PreferredSentenceLength)
		argIdx++
	}
	if req.ParagraphSizeMin != nil {
		setClauses = append(setClauses, fmt.Sprintf("paragraph_size_min = $%d", argIdx))
		args = append(args, *req.ParagraphSizeMin)
		argIdx++
	}
	if req.ParagraphSizeMax != nil {
		setClauses = append(setClauses, fmt.Sprintf("paragraph_size_max = $%d", argIdx))
		args = append(args, *req.ParagraphSizeMax)
		argIdx++
	}
	if req.IsActive != nil {
		setClauses = append(setClauses, fmt.Sprintf("is_active = $%d", argIdx))
		args = append(args, *req.IsActive)
		argIdx++
	}
	if req.Metadata != nil {
		setClauses = append(setClauses, fmt.Sprintf("metadata = $%d::jsonb", argIdx))
		args = append(args, toJSON(*req.Metadata))
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.GetProfile(ctx, siteID, profileID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE writing_profiles SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, profileID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	s.fireEvent(ctx, EventProfileUpdated, map[string]interface{}{
		"profile_id": profileID.String(),
		"site_id":    siteID.String(),
	}, siteID)

	return s.GetProfile(ctx, siteID, profileID)
}

func (s *Service) DeleteProfile(ctx context.Context, siteID, profileID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	tag, err := p.Exec(ctx,
		`DELETE FROM writing_profiles WHERE id = $1 AND site_id = $2`,
		profileID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrProfileNotFound
	}

	s.fireEvent(ctx, EventProfileDeleted, map[string]interface{}{
		"profile_id": profileID.String(),
		"site_id":    siteID.String(),
	}, siteID)
	return nil
}

// --- Writing Rules ---

func (s *Service) ListRules(ctx context.Context, siteID uuid.UUID, profileID *uuid.UUID) ([]WritingRule, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2
	if profileID != nil {
		where = append(where, fmt.Sprintf("(profile_id = $%d OR profile_id IS NULL)", argIdx))
		args = append(args, *profileID)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, profile_id, rule_key, category, enabled, priority, config::text,
		        COALESCE(description,''), created_at, updated_at
		 FROM writing_rules WHERE %s ORDER BY priority ASC, rule_key ASC`,
		strings.Join(where, " AND "),
	)
	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer rows.Close()

	var rules []WritingRule
	for rows.Next() {
		var r WritingRule
		var configStr string
		if err := rows.Scan(&r.ID, &r.SiteID, &r.ProfileID, &r.RuleKey, &r.Category,
			&r.Enabled, &r.Priority, &configStr, &r.Description, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		if len(configStr) > 0 {
			r.Config = parseJSON(configStr)
		}
		if r.Config == nil {
			r.Config = make(map[string]interface{})
		}
		rules = append(rules, r)
	}
	if rules == nil {
		rules = []WritingRule{}
	}
	return rules, nil
}

func (s *Service) ToggleRule(ctx context.Context, siteID, ruleID uuid.UUID, enabled bool) (*WritingRule, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	tag, err := p.Exec(ctx,
		`UPDATE writing_rules SET enabled = $1, updated_at = NOW() WHERE id = $2 AND site_id = $3`,
		enabled, ruleID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to toggle rule: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil, ErrRuleNotFound
	}

	var r WritingRule
	var configStr string
	err = p.QueryRow(ctx,
		`SELECT id, site_id, profile_id, rule_key, category, enabled, priority, config::text,
		        COALESCE(description,''), created_at, updated_at
		 FROM writing_rules WHERE id = $1`,
		ruleID,
	).Scan(&r.ID, &r.SiteID, &r.ProfileID, &r.RuleKey, &r.Category,
		&r.Enabled, &r.Priority, &configStr, &r.Description, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated rule: %w", err)
	}
	if len(configStr) > 0 {
		r.Config = parseJSON(configStr)
	}
	if r.Config == nil {
		r.Config = make(map[string]interface{})
	}

	s.fireEvent(ctx, EventRuleToggled, map[string]interface{}{
		"rule_id": r.ID.String(),
		"rule_key": r.RuleKey,
		"enabled":  enabled,
	}, siteID)

	return &r, nil
}

// --- Writing Personas ---

func (s *Service) CreatePersona(ctx context.Context, siteID uuid.UUID, req interface{}) (*WritingPersona, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	personaID := uuid.New()
	lang := "pt"

	if m, ok := req.(map[string]interface{}); ok {
		if l, ok := m["language"].(string); ok {
			lang = l
		}
	}

	_, err = p.Exec(ctx,
		`INSERT INTO writing_personas (id, site_id, name, title, bio, voice_traits,
		 vocabulary_style, sentence_patterns, expertise_areas, language, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)`,
		personaID, siteID, "New Persona", "", "", []string{},
		[]string{}, []string{}, []string{}, lang, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create persona: %w", err)
	}

	return s.GetPersona(ctx, siteID, personaID)
}

func (s *Service) GetPersona(ctx context.Context, siteID, personaID uuid.UUID) (*WritingPersona, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var per WritingPersona
	var metadataStr string
	err = p.QueryRow(ctx,
		`SELECT id, site_id, profile_id, name, COALESCE(title,''), COALESCE(bio,''),
		        COALESCE(voice_traits,'{}'), COALESCE(vocabulary_style,'{}'),
		        COALESCE(sentence_patterns,'{}'), COALESCE(expertise_areas,'{}'),
		        language, is_active, COALESCE(metadata::text,'{}'), created_by, created_at, updated_at
		 FROM writing_personas WHERE id = $1 AND site_id = $2`,
		personaID, siteID,
	).Scan(&per.ID, &per.SiteID, &per.ProfileID, &per.Name, &per.Title, &per.Bio,
		&per.VoiceTraits, &per.VocabularyStyle, &per.SentencePatterns, &per.ExpertiseAreas,
		&per.Language, &per.IsActive, &metadataStr, &per.CreatedBy, &per.CreatedAt, &per.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPersonaNotFound
		}
		return nil, fmt.Errorf("failed to get persona: %w", err)
	}
	if len(metadataStr) > 0 {
		per.Metadata = parseJSON(metadataStr)
	}
	if per.Metadata == nil {
		per.Metadata = make(map[string]interface{})
	}
	return &per, nil
}

func (s *Service) ListPersonas(ctx context.Context, siteID uuid.UUID, profileID *uuid.UUID, language string) ([]WritingPersona, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2
	if profileID != nil {
		where = append(where, fmt.Sprintf("profile_id = $%d", argIdx))
		args = append(args, *profileID)
		argIdx++
	}
	if language != "" {
		where = append(where, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, language)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, profile_id, name, COALESCE(title,''), COALESCE(bio,''),
		        COALESCE(voice_traits,'{}'), COALESCE(vocabulary_style,'{}'),
		        COALESCE(sentence_patterns,'{}'), COALESCE(expertise_areas,'{}'),
		        language, is_active, created_by, created_at, updated_at
		 FROM writing_personas WHERE %s ORDER BY name ASC`, strings.Join(where, " AND "),
	)
	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list personas: %w", err)
	}
	defer rows.Close()

	var personas []WritingPersona
	for rows.Next() {
		var per WritingPersona
		if err := rows.Scan(&per.ID, &per.SiteID, &per.ProfileID, &per.Name, &per.Title,
			&per.Bio, &per.VoiceTraits, &per.VocabularyStyle, &per.SentencePatterns,
			&per.ExpertiseAreas, &per.Language, &per.IsActive,
			&per.CreatedBy, &per.CreatedAt, &per.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan persona: %w", err)
		}
		personas = append(personas, per)
	}
	if personas == nil {
		personas = []WritingPersona{}
	}
	return personas, nil
}

func (s *Service) UpdatePersona(ctx context.Context, siteID, personaID uuid.UUID, updates map[string]interface{}) (*WritingPersona, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	_, err = s.GetPersona(ctx, siteID, personaID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for key, val := range updates {
		if key == "metadata" {
			setClauses = append(setClauses, fmt.Sprintf("metadata = $%d::jsonb", argIdx))
			args = append(args, toJSON(val))
		} else {
			setClauses = append(setClauses, fmt.Sprintf("%s = $%d", key, argIdx))
			args = append(args, val)
		}
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.GetPersona(ctx, siteID, personaID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE writing_personas SET %s WHERE id = $%d AND site_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, personaID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update persona: %w", err)
	}

	s.fireEvent(ctx, EventPersonaUpdated, map[string]interface{}{
		"persona_id": personaID.String(),
		"site_id":    siteID.String(),
	}, siteID)

	return s.GetPersona(ctx, siteID, personaID)
}

func (s *Service) DeletePersona(ctx context.Context, siteID, personaID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	tag, err := p.Exec(ctx,
		`DELETE FROM writing_personas WHERE id = $1 AND site_id = $2`,
		personaID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete persona: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPersonaNotFound
	}

	s.fireEvent(ctx, EventPersonaDeleted, map[string]interface{}{
		"persona_id": personaID.String(),
		"site_id":    siteID.String(),
	}, siteID)
	return nil
}

// --- Vocabulary Sets ---

func (s *Service) ListVocabularySets(ctx context.Context, siteID uuid.UUID, category, language string) ([]VocabularySet, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2
	if category != "" {
		where = append(where, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, category)
		argIdx++
	}
	if language != "" {
		where = append(where, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, language)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, name, COALESCE(category,'general'), words, replacements,
		        language, COALESCE(tags,'{}'), is_active, created_by, created_at, updated_at
		 FROM vocabulary_sets WHERE %s ORDER BY name ASC`, strings.Join(where, " AND "),
	)
	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list vocabulary sets: %w", err)
	}
	defer rows.Close()

	var sets []VocabularySet
	for rows.Next() {
		var vs VocabularySet
		if err := rows.Scan(&vs.ID, &vs.SiteID, &vs.Name, &vs.Category, &vs.Words,
			&vs.Replacements, &vs.Language, &vs.Tags, &vs.IsActive,
			&vs.CreatedBy, &vs.CreatedAt, &vs.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan vocabulary set: %w", err)
		}
		sets = append(sets, vs)
	}
	if sets == nil {
		sets = []VocabularySet{}
	}
	return sets, nil
}

// --- Transition Library ---

func (s *Service) ListTransitions(ctx context.Context, siteID uuid.UUID, category, language string) ([]TransitionPhrase, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2
	if category != "" {
		where = append(where, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, category)
		argIdx++
	}
	if language != "" {
		where = append(where, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, language)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, category, phrase, language, COALESCE(formality,'neutral'),
		        COALESCE(usage_count,0), is_active, created_at, updated_at
		 FROM transition_library WHERE %s ORDER BY category ASC, phrase ASC`,
		strings.Join(where, " AND "),
	)
	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list transitions: %w", err)
	}
	defer rows.Close()

	var transitions []TransitionPhrase
	for rows.Next() {
		var t TransitionPhrase
		if err := rows.Scan(&t.ID, &t.SiteID, &t.Category, &t.Phrase, &t.Language,
			&t.Formality, &t.UsageCount, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan transition: %w", err)
		}
		transitions = append(transitions, t)
	}
	if transitions == nil {
		transitions = []TransitionPhrase{}
	}
	return transitions, nil
}

// --- Style Patterns ---

func (s *Service) ListPatterns(ctx context.Context, siteID uuid.UUID, patternType, language string) ([]StylePattern, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2
	if patternType != "" {
		where = append(where, fmt.Sprintf("pattern_type = $%d", argIdx))
		args = append(args, patternType)
		argIdx++
	}
	if language != "" {
		where = append(where, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, language)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, profile_id, name, pattern_type, pattern, language,
		        COALESCE(tags,'{}'), COALESCE(effectiveness_score,0), COALESCE(usage_count,0),
		        is_active, created_at, updated_at
		 FROM style_patterns WHERE %s ORDER BY effectiveness_score DESC`,
		strings.Join(where, " AND "),
	)
	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list patterns: %w", err)
	}
	defer rows.Close()

	var patterns []StylePattern
	for rows.Next() {
		var sp StylePattern
		if err := rows.Scan(&sp.ID, &sp.SiteID, &sp.ProfileID, &sp.Name, &sp.PatternType,
			&sp.Pattern, &sp.Language, &sp.Tags, &sp.EffectivenessScore, &sp.UsageCount,
			&sp.IsActive, &sp.CreatedAt, &sp.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan pattern: %w", err)
		}
		patterns = append(patterns, sp)
	}
	if patterns == nil {
		patterns = []StylePattern{}
	}
	return patterns, nil
}

// --- Sentence Templates ---

func (s *Service) ListTemplates(ctx context.Context, siteID uuid.UUID, category, language string) ([]SentenceTemplate, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2
	if category != "" {
		where = append(where, fmt.Sprintf("category = $%d", argIdx))
		args = append(args, category)
		argIdx++
	}
	if language != "" {
		where = append(where, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, language)
		argIdx++
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, profile_id, name, template, COALESCE(category,'general'),
		        COALESCE(variables,'{}'), language, COALESCE(formality,'neutral'),
		        COALESCE(usage_count,0), is_active, created_at, updated_at
		 FROM sentence_templates WHERE %s ORDER BY category ASC, name ASC`,
		strings.Join(where, " AND "),
	)
	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}
	defer rows.Close()

	var templates []SentenceTemplate
	for rows.Next() {
		var st SentenceTemplate
		if err := rows.Scan(&st.ID, &st.SiteID, &st.ProfileID, &st.Name, &st.Template,
			&st.Category, &st.Variables, &st.Language, &st.Formality,
			&st.UsageCount, &st.IsActive, &st.CreatedAt, &st.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan template: %w", err)
		}
		templates = append(templates, st)
	}
	if templates == nil {
		templates = []SentenceTemplate{}
	}
	return templates, nil
}

// --- Humanization History ---

func (s *Service) ListHistory(ctx context.Context, siteID uuid.UUID, profileID *uuid.UUID, language string, limit, offset int) ([]HumanizationRecord, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2
	if profileID != nil {
		where = append(where, fmt.Sprintf("profile_id = $%d", argIdx))
		args = append(args, *profileID)
		argIdx++
	}
	if language != "" {
		where = append(where, fmt.Sprintf("language = $%d", argIdx))
		args = append(args, language)
		argIdx++
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, profile_id, source_text, humanized_text,
		        COALESCE(burstiness_score,0), COALESCE(perplexity_score,0),
		        COALESCE(repetition_score,0), COALESCE(passive_voice_score,0),
		        COALESCE(rhythm_score,0), COALESCE(flow_score,0),
		        COALESCE(rules_applied,'{}'), COALESCE(transformations::text,'[]'),
		        language, COALESCE(word_count_original,0), COALESCE(word_count_humanized,0),
		        COALESCE(duration_ms,0), created_by, created_at
		 FROM humanization_history WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list history: %w", err)
	}
	defer rows.Close()

	var records []HumanizationRecord
	for rows.Next() {
		var rec HumanizationRecord
		var transStr string
		if err := rows.Scan(&rec.ID, &rec.SiteID, &rec.ProfileID, &rec.SourceText,
			&rec.HumanizedText, &rec.BurstinessScore, &rec.PerplexityScore,
			&rec.RepetitionScore, &rec.PassiveVoiceScore, &rec.RhythmScore,
			&rec.FlowScore, &rec.RulesApplied, &transStr,
			&rec.Language, &rec.WordCountOriginal, &rec.WordCountHumanized,
			&rec.DurationMs, &rec.CreatedBy, &rec.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		if len(transStr) > 0 {
			var trans []map[string]interface{}
			if err := json.Unmarshal([]byte(transStr), &trans); err == nil {
				rec.Transformations = trans
			}
		}
		records = append(records, rec)
	}
	if records == nil {
		records = []HumanizationRecord{}
	}
	return records, nil
}

// --- Humanization Engine ---

func (s *Service) Humanize(ctx context.Context, siteID uuid.UUID, req HumanizeRequest) (*HumanizationResult, error) {
	if req.Text == "" {
		return nil, ErrInvalidText
	}
	lang := coalesceStr(req.Language, "pt")
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}

	start := time.Now()

	var profile *WritingProfile
	if req.ProfileID != nil {
		var err error
		profile, err = s.GetProfile(ctx, siteID, *req.ProfileID)
		if err != nil {
			return nil, err
		}
	} else if req.Slug != "" {
		var err error
		profile, err = s.GetProfileBySlug(ctx, siteID, req.Slug)
		if err != nil {
			_ = err
		}
	}

	rulesApplied := []string{}
	transformations := []map[string]interface{}{}

	text := req.Text
	origWords := countWords(text)

	if profile != nil {
		text = s.applyParagraphNormalization(text, profile)
		rulesApplied = append(rulesApplied, RuleKeys.NaturalParagraphSizes)
		transformations = append(transformations, map[string]interface{}{
			"type": "paragraph_normalization", "profile": profile.Slug,
		})
	}

	sentences := splitSentences(text)
	sentenceVaried := s.applySentenceVariation(sentences, lang)
	if sentenceVaried != text {
		rulesApplied = append(rulesApplied, RuleKeys.VariableSentenceLengths)
		transformations = append(transformations, map[string]interface{}{
			"type": "sentence_variation",
		})
		text = sentenceVaried
	}

	text = s.applyConnectorRotation(text, lang, profile)
	rulesApplied = append(rulesApplied, RuleKeys.NaturalConnectorRotation)
	transformations = append(transformations, map[string]interface{}{
		"type": "connector_rotation",
	})

	text = s.applyVocabularyDiversity(text, lang, profile)
	rulesApplied = append(rulesApplied, "vocabulary_diversity")
	transformations = append(transformations, map[string]interface{}{
		"type": "vocabulary_diversity",
	})

	text = s.removeAICliches(text, lang)
	rulesApplied = append(rulesApplied, RuleKeys.AvoidAICliches)
	transformations = append(transformations, map[string]interface{}{
		"type": "remove_ai_cliches",
	})

	text = s.varyOpenings(text, lang)
	rulesApplied = append(rulesApplied, RuleKeys.AvoidRepetitiveOpenings)
	transformations = append(transformations, map[string]interface{}{
		"type": "vary_openings",
	})

	text = s.varyConclusions(text, lang)
	rulesApplied = append(rulesApplied, RuleKeys.AvoidRepetitiveConclusions)
	transformations = append(transformations, map[string]interface{}{
		"type": "vary_conclusions",
	})

	text = s.insertPlaceholders(text, lang)
	rulesApplied = append(rulesApplied, RuleKeys.QuoteInsertionSupport)
	rulesApplied = append(rulesApplied, RuleKeys.StatisticInsertionSupport)
	rulesApplied = append(rulesApplied, RuleKeys.ExpertOpinionPlaceholders)
	transformations = append(transformations, map[string]interface{}{
		"type": "insert_placeholders",
	})

	humanizedWords := countWords(text)
	durationMs := int(time.Since(start).Milliseconds())

	result := &HumanizationResult{
		HumanizedText:     text,
		BurstinessScore:   s.calcBurstiness(text),
		PerplexityScore:   s.calcPerplexity(text, lang),
		RepetitionScore:   s.detectRepetition(text),
		PassiveVoiceScore: s.detectPassiveVoice(text, lang),
		RhythmScore:       s.analyzeRhythm(text),
		FlowScore:         s.analyzeFlow(text),
		RulesApplied:      rulesApplied,
		Transformations:   transformations,
		WordCountOriginal: origWords,
		WordCountHumanized: humanizedWords,
	}

	if s.db != nil && s.db.Pool != nil {
		s.saveHistory(ctx, siteID, profile, req, result, durationMs)
	}

	s.fireEvent(ctx, EventHumanized, map[string]interface{}{
		"site_id":          siteID.String(),
		"burstiness":       result.BurstinessScore,
		"perplexity":       result.PerplexityScore,
		"rules_applied":    len(rulesApplied),
		"word_count_change": humanizedWords - origWords,
	}, siteID)

	return result, nil
}

func (s *Service) BatchHumanize(ctx context.Context, siteID uuid.UUID, req BatchHumanizeRequest) (*BatchHumanizeResult, error) {
	if len(req.Texts) == 0 {
		return nil, ErrInvalidText
	}

	results := make([]HumanizationResult, 0, len(req.Texts))
	for _, text := range req.Texts {
		r, err := s.Humanize(ctx, siteID, HumanizeRequest{
			Text:      text,
			ProfileID: req.ProfileID,
			Slug:      req.Slug,
			Language:  req.Language,
		})
		if err != nil {
			return nil, err
		}
		results = append(results, *r)
	}

	s.fireEvent(ctx, EventBatchHumanized, map[string]interface{}{
		"site_id": siteID.String(),
		"count":   len(results),
	}, siteID)

	return &BatchHumanizeResult{Results: results}, nil
}

func (s *Service) AnalyzeText(ctx context.Context, req AnalyzeRequest) (*AnalyzeResult, error) {
	if req.Text == "" {
		return nil, ErrInvalidText
	}
	lang := coalesceStr(req.Language, "pt")

	words := splitWords(req.Text)
	sentences := splitSentences(req.Text)
	paragraphs := splitParagraphs(req.Text)

	wordCount := len(words)
	sentenceCount := len(sentences)
	paragraphCount := len(paragraphs)

	avgSentenceLen := 0.0
	if sentenceCount > 0 {
		avgSentenceLen = float64(wordCount) / float64(sentenceCount)
	}

	avgParagraphSize := 0.0
	if paragraphCount > 0 {
		avgParagraphSize = float64(sentenceCount) / float64(paragraphCount)
	}

	uniqueWords := make(map[string]bool)
	for _, w := range words {
		w = strings.ToLower(strings.Trim(w, ".,!?;:\"'()[]{}"))
		if w != "" {
			uniqueWords[w] = true
		}
	}
	vocabDensity := 0.0
	if wordCount > 0 {
		vocabDensity = float64(len(uniqueWords)) / float64(wordCount)
	}

	return &AnalyzeResult{
		BurstinessScore:   s.calcBurstiness(req.Text),
		PerplexityScore:   s.calcPerplexity(req.Text, lang),
		RepetitionScore:   s.detectRepetition(req.Text),
		PassiveVoiceScore: s.detectPassiveVoice(req.Text, lang),
		RhythmScore:       s.analyzeRhythm(req.Text),
		FlowScore:         s.analyzeFlow(req.Text),
		SentenceCount:     sentenceCount,
		ParagraphCount:    paragraphCount,
		WordCount:         wordCount,
		AvgSentenceLength: avgSentenceLen,
		AvgParagraphSize:  avgParagraphSize,
		VocabularyDensity: vocabDensity,
	}, nil
}

func (s *Service) GetMetrics(ctx context.Context, siteID uuid.UUID) (*HumanWriterMetrics, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var metrics HumanWriterMetrics

	err = p.QueryRow(ctx,
		`SELECT COUNT(*) FROM humanization_history WHERE site_id = $1`, siteID,
	).Scan(&metrics.TotalRequests)
	if err != nil {
		return nil, fmt.Errorf("failed to count requests: %w", err)
	}

	if metrics.TotalRequests > 0 {
		p.QueryRow(ctx,
			`SELECT COALESCE(AVG(burstiness_score),0), COALESCE(AVG(perplexity_score),0),
			        COALESCE(AVG(repetition_score),0), COALESCE(AVG(passive_voice_score),0),
			        COALESCE(AVG(rhythm_score),0), COALESCE(AVG(flow_score),0)
			 FROM humanization_history WHERE site_id = $1`, siteID,
		).Scan(&metrics.AvgBurstiness, &metrics.AvgPerplexity, &metrics.AvgRepetition,
			&metrics.AvgPassiveVoice, &metrics.AvgRhythm, &metrics.AvgFlow)
	}

	p.QueryRow(ctx, `SELECT COUNT(*) FROM writing_profiles WHERE site_id = $1`, siteID).Scan(&metrics.ProfileCount)
	p.QueryRow(ctx, `SELECT COUNT(*) FROM writing_rules WHERE site_id = $1`, siteID).Scan(&metrics.RuleCount)
	p.QueryRow(ctx, `SELECT COUNT(*) FROM writing_personas WHERE site_id = $1`, siteID).Scan(&metrics.PersonaCount)
	p.QueryRow(ctx, `SELECT COUNT(*) FROM vocabulary_sets WHERE site_id = $1`, siteID).Scan(&metrics.VocabularyCount)
	p.QueryRow(ctx, `SELECT COUNT(*) FROM transition_library WHERE site_id = $1`, siteID).Scan(&metrics.TransitionCount)
	p.QueryRow(ctx, `SELECT COUNT(*) FROM sentence_templates WHERE site_id = $1`, siteID).Scan(&metrics.TemplateCount)
	p.QueryRow(ctx, `SELECT COUNT(*) FROM style_patterns WHERE site_id = $1`, siteID).Scan(&metrics.PatternCount)

	return &metrics, nil
}

func (s *Service) saveHistory(ctx context.Context, siteID uuid.UUID, profile *WritingProfile, req HumanizeRequest, result *HumanizationResult, durationMs int) {
	p, err := s.pool()
	if err != nil {
		return
	}

	var profileID *uuid.UUID
	if profile != nil {
		profileID = &profile.ID
	}

	transJSON := "[]"
	if len(result.Transformations) > 0 {
		transJSON = toJSON(result.Transformations)
	}

	p.Exec(ctx,
		`INSERT INTO humanization_history (id, site_id, profile_id, source_text, humanized_text,
		 burstiness_score, perplexity_score, repetition_score, passive_voice_score,
		 rhythm_score, flow_score, rules_applied, transformations, language,
		 word_count_original, word_count_humanized, duration_ms, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14,$15,$16,$17,NOW())`,
		uuid.New(), siteID, profileID, req.Text, result.HumanizedText,
		result.BurstinessScore, result.PerplexityScore, result.RepetitionScore,
		result.PassiveVoiceScore, result.RhythmScore, result.FlowScore,
		result.RulesApplied, transJSON, req.Language,
		result.WordCountOriginal, result.WordCountHumanized, durationMs,
	)
}

func (s *Service) applyParagraphNormalization(text string, profile *WritingProfile) string {
	paragraphs := splitParagraphs(text)
	if len(paragraphs) <= 1 {
		return text
	}

	var result []string
	for _, para := range paragraphs {
		sentences := splitSentences(para)
		if len(sentences) == 0 {
			result = append(result, para)
			continue
		}
		minSize := profile.ParagraphSizeMin
		if minSize < 1 {
			minSize = 2
		}
		maxSize := profile.ParagraphSizeMax
		if maxSize < minSize {
			maxSize = minSize + 3
		}
		if len(sentences) < minSize {
			result = append(result, para)
		} else if len(sentences) > maxSize {
			for i := 0; i < len(sentences); i += maxSize {
				end := i + maxSize
				if end > len(sentences) {
					end = len(sentences)
				}
				result = append(result, strings.Join(sentences[i:end], " "))
			}
		} else {
			result = append(result, para)
		}
	}
	return strings.Join(result, "\n\n")
}

func (s *Service) applySentenceVariation(sentences []string, lang string) string {
	if len(sentences) < 3 {
		return strings.Join(sentences, " ")
	}

	var varied []string
	prevLen := 0
	for i, sent := range sentences {
		wordCount := len(strings.Fields(sent))
		if i > 0 && prevLen > 0 && wordCount == prevLen {
			adj := s.adjustSentenceLength(sent, wordCount, lang)
			varied = append(varied, adj)
			prevLen = len(strings.Fields(adj))
		} else {
			varied = append(varied, sent)
			prevLen = wordCount
		}
	}
	return strings.Join(varied, " ")
}

func (s *Service) adjustSentenceLength(sentence string, wordCount int, lang string) string {
	if wordCount < 5 {
		expansion := s.getExpansionPhrase(lang)
		parts := strings.SplitN(sentence, " ", 2)
		if len(parts) == 2 {
			return parts[0] + " " + expansion + " " + parts[1]
		}
	} else if wordCount > 15 {
		compression := s.getCompressionPhrase(lang)
		parts := strings.SplitN(sentence, " ", 2)
		if len(parts) == 2 {
			return parts[0] + " " + compression + " " + parts[1]
		}
	}
	return sentence
}

func (s *Service) getExpansionPhrase(lang string) string {
	pt := []string{"na verdade", "sem dúvida", "como se sabe", "é importante mencionar que",
		"vale ressaltar que", "naturalmente", "certamente"}
	en := []string{"in fact", "undoubtedly", "as we know", "it is worth noting that",
		"importantly", "naturally", "certainly"}
	if lang == "pt" {
		return pt[rand.Intn(len(pt))]
	}
	return en[rand.Intn(len(en))]
}

func (s *Service) getCompressionPhrase(lang string) string {
	pt := []string{"ou seja", "isto é", "em suma", "em resumo", "resumidamente"}
	en := []string{"i.e.", "that is", "in short", "briefly", "in summary"}
	if lang == "pt" {
		return pt[rand.Intn(len(pt))]
	}
	return en[rand.Intn(len(en))]
}

func (s *Service) applyConnectorRotation(text string, lang string, profile *WritingProfile) string {
	ptReplacements := map[string][]string{
		"Além disso":  {"Além disso", "Ademais", "Outrossim", "Também", "Da mesma forma"},
		"Portanto":    {"Portanto", "Dessa forma", "Desse modo", "Assim", "Consequentemente"},
		"Porém":       {"Porém", "Contudo", "Todavia", "No entanto", "Entretanto"},
		"Por exemplo": {"Por exemplo", "Como exemplo", "A título de exemplo", "Ilustrando"},
		"Primeiramente": {"Primeiramente", "Em primeiro lugar", "Inicialmente", "Antes de mais nada"},
		"Finalmente": {"Finalmente", "Por fim", "Por último", "Em conclusão"},
		"Portanto,":    {"Portanto,", "Dessa forma,", "Desse modo,", "Assim,", "Consequentemente,"},
	}
	enReplacements := map[string][]string{
		"Furthermore": {"Furthermore", "Moreover", "Additionally", "Also", "In addition"},
		"Therefore":   {"Therefore", "Thus", "Hence", "Consequently", "As a result"},
		"However":     {"However", "Nevertheless", "Nonetheless", "On the other hand", "Yet"},
		"For example": {"For example", "For instance", "As an illustration", "To illustrate"},
		"Firstly":     {"Firstly", "First of all", "To begin", "Initially"},
		"Finally":     {"Finally", "Lastly", "In conclusion", "To conclude"},
		"Therefore,":  {"Therefore,", "Thus,", "Hence,", "Consequently,", "As a result,"},
	}

	replacements := enReplacements
	if lang == "pt" {
		replacements = ptReplacements
	}

	result := text
	for original, options := range replacements {
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(original) + `\b`)
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			return options[rand.Intn(len(options))]
		})
	}
	return result
}

func (s *Service) applyVocabularyDiversity(text string, lang string, profile *WritingProfile) string {
	ptReplacements := map[string][]string{
		"bom":    {"bom", "excelente", "notável", "extraordinário", "positivo"},
		"ruim":   {"ruim", "deficiente", "insatisfatório", "problemático", "precário"},
		"grande": {"grande", "enorme", "vasto", "amplo", "significativo"},
		"novo":   {"novo", "recente", "inovador", "moderno", "contemporâneo"},
		"importante": {"importante", "fundamental", "essencial", "crucial", "vital"},
		"muito": {"muito", "bastante", "consideravelmente", "extremamente", "altamente"},
		"interessante": {"interessante", "cativante", "fascinante", "envolvente", "intrigante"},
	}
	enReplacements := map[string][]string{
		"good":     {"good", "excellent", "notable", "outstanding", "positive"},
		"bad":      {"bad", "deficient", "unsatisfactory", "problematic", "poor"},
		"big":      {"big", "enormous", "vast", "extensive", "significant"},
		"new":      {"new", "recent", "innovative", "modern", "contemporary"},
		"important": {"important", "fundamental", "essential", "crucial", "vital"},
		"very":     {"very", "quite", "considerably", "extremely", "highly"},
		"interesting": {"interesting", "captivating", "fascinating", "engaging", "intriguing"},
	}

	replacements := enReplacements
	if lang == "pt" {
		replacements = ptReplacements
	}

	result := text
	for original, options := range replacements {
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(original) + `\b`)
		result = pattern.ReplaceAllStringFunc(result, func(match string) string {
			if rand.Float64() < 0.5 {
				return match
			}
			return options[rand.Intn(len(options))]
		})
	}
	return result
}

var aiCliches = map[string][]string{
	"pt": {
		"no mundo atual", "num mundo cada vez mais",
		"com o avanço da tecnologia", "na era digital",
		"é fundamental destacar", "é importante salientar",
		"vale mencionar", "como mencionado anteriormente",
		"diante desse cenário", "neste contexto",
		"em outras palavras", "em suma",
	},
	"en": {
		"in today's world", "in an increasingly",
		"with the advancement of technology", "in the digital era",
		"it is worth noting", "it is important to highlight",
		"it is worth mentioning", "as previously mentioned",
		"against this backdrop", "in this context",
		"in other words", "in summary",
	},
}

func (s *Service) removeAICliches(text string, lang string) string {
	cliches, ok := aiCliches[lang]
	if !ok {
		return text
	}
	result := text
	for _, cliche := range cliches {
		pattern := regexp.MustCompile(`(?i)` + regexp.QuoteMeta(cliche))
		result = pattern.ReplaceAllString(result, "...")
	}
	result = strings.ReplaceAll(result, "...", "")
	result = strings.ReplaceAll(result, "  ", " ")
	return strings.TrimSpace(result)
}

var openingVariations = map[string][]string{
	"pt": {
		"Você já parou para pensar", "Imagine", "Considere por um momento",
		"Um fato interessante", "Vamos refletir", "É notável como",
		"Vale a pena observar", "Pense nisto",
	},
	"en": {
		"Have you ever wondered", "Imagine", "Consider for a moment",
		"An interesting fact", "Let's reflect on", "It's remarkable how",
		"It's worth noting", "Think about this",
	},
}

var conclusionVariations = map[string][]string{
	"pt": {
		"Em conclusão", "Para finalizar", "Fica claro que",
		"Fica evidente que", "Portanto", "Resumindo",
		"Em resumo", "Diante do exposto",
	},
	"en": {
		"In conclusion", "To sum up", "It is clear that",
		"It is evident that", "Therefore", "In summary",
		"To summarize", "Given the above",
	},
}

func (s *Service) varyOpenings(text string, lang string) string {
	paragraphs := splitParagraphs(text)
	vars, ok := openingVariations[lang]
	if !ok {
		return text
	}
	for i, para := range paragraphs {
		if i == 0 {
			continue
		}
		sentences := splitSentences(para)
		if len(sentences) == 0 {
			continue
		}
		first := sentences[0]
		for _, cliche := range []string{"Além disso", "Além disto", "Além de", "Também",
			"Moreover", "Furthermore", "Additionally"} {
			if strings.HasPrefix(strings.TrimSpace(first), cliche) {
				v := vars[rand.Intn(len(vars))]
				rest := strings.TrimSpace(first[len(cliche):])
				sentences[0] = v + rest
				paragraphs[i] = strings.Join(sentences, " ")
				break
			}
		}
	}
	return strings.Join(paragraphs, "\n\n")
}

func (s *Service) varyConclusions(text string, lang string) string {
	paragraphs := splitParagraphs(text)
	if len(paragraphs) == 0 {
		return text
	}
	last := paragraphs[len(paragraphs)-1]
	sentences := splitSentences(last)
	if len(sentences) < 2 {
		return text
	}
	vars, ok := conclusionVariations[lang]
	if !ok {
		return text
	}
	lastSentence := strings.TrimSpace(sentences[len(sentences)-1])
	conclusionStarts := []string{"Em conclusão", "Para finalizar", "Portanto",
		"In conclusion", "To sum up", "Therefore", "Finally"}
	for _, cs := range conclusionStarts {
		if strings.HasPrefix(lastSentence, cs) {
			v := vars[rand.Intn(len(vars))]
			rest := strings.TrimSpace(lastSentence[len(cs):])
			sentences[len(sentences)-1] = v + " " + rest
			paragraphs[len(paragraphs)-1] = strings.Join(sentences, " ")
			break
		}
	}
	return strings.Join(paragraphs, "\n\n")
}

var placeholderPatterns = map[string][]string{
	"pt": {
		`\[citação\]`, `\[quote\]`, `\[aspas\]`,
		`\[estatística\]`, `\[dado\]`, `\[número\]`,
		`\[especialista\]`, `\[opinião de especialista\]`,
	},
	"en": {
		`\[citation\]`, `\[quote\]`, `\[statistic\]`, `\[data\]`,
		`\[expert\]`, `\[expert opinion\]`,
	},
}

func (s *Service) insertPlaceholders(text string, lang string) string {
	patterns, ok := placeholderPatterns[lang]
	if !ok {
		return text
	}
	result := text
	for _, pat := range patterns {
		re := regexp.MustCompile(`(?i)` + pat)
		if strings.Contains(pat, "citação") || strings.Contains(pat, "citation") || strings.Contains(pat, "quote") {
			result = re.ReplaceAllString(result, s.randomQuote(lang)+" — "+s.randomSource(lang))
		} else if strings.Contains(pat, "estatística") || strings.Contains(pat, "statistic") || strings.Contains(pat, "data") || strings.Contains(pat, "número") {
			result = re.ReplaceAllString(result, s.randomStatistic(lang))
		} else if strings.Contains(pat, "especialista") || strings.Contains(pat, "expert") {
			result = re.ReplaceAllString(result, s.randomExpertOpinion(lang))
		}
	}
	return result
}

func (s *Service) randomQuote(lang string) string {
	pt := []string{"Segundo especialistas da área", "Conforme aponta a pesquisa recente",
		"De acordo com estudos realizados", "Conforme relatado por pesquisadores"}
	en := []string{"According to field experts", "As recent research points out",
		"According to conducted studies", "As reported by researchers"}
	if lang == "pt" {
		return pt[rand.Intn(len(pt))]
	}
	return en[rand.Intn(len(en))]
}

func (s *Service) randomSource(lang string) string {
	pt := []string{"especialista consultado", "fonte oficial", "relatório do setor",
		"documento de referência", "autoridade no tema"}
	en := []string{"consulted specialist", "official source", "industry report",
		"reference document", "subject matter authority"}
	if lang == "pt" {
		return pt[rand.Intn(len(pt))]
	}
	return en[rand.Intn(len(en))]
}

func (s *Service) randomStatistic(lang string) string {
	pt := []string{"de acordo com dados recentes, aproximadamente 70% dos casos indicam que",
		"estudos mostram que cerca de 65% dos usuários preferem",
		"pesquisas indicam que mais de 80% dos profissionais concordam que",
		"dados do setor revelam que aproximadamente 75% das empresas adotam"}
	en := []string{"according to recent data, approximately 70% of cases indicate that",
		"studies show that about 65% of users prefer",
		"research indicates that over 80% of professionals agree that",
		"industry data reveals that approximately 75% of companies adopt"}
	if lang == "pt" {
		return pt[rand.Intn(len(pt))]
	}
	return en[rand.Intn(len(en))]
}

func (s *Service) randomExpertOpinion(lang string) string {
	pt := []string{"especialistas consultados recomendam que",
		"analistas do setor sugerem que",
		"profissionais experientes apontam que",
		"consultores especializados indicam que"}
	en := []string{"consulted experts recommend that",
		"industry analysts suggest that",
		"experienced professionals point out that",
		"specialized consultants indicate that"}
	if lang == "pt" {
		return pt[rand.Intn(len(pt))]
	}
	return en[rand.Intn(len(en))]
}

func (s *Service) calcBurstiness(text string) float64 {
	sentences := splitSentences(text)
	if len(sentences) < 2 {
		return 0
	}

	lengths := make([]float64, len(sentences))
	for i, s := range sentences {
		lengths[i] = float64(len(strings.Fields(s)))
	}

	mean := 0.0
	for _, l := range lengths {
		mean += l
	}
	mean /= float64(len(lengths))

	variance := 0.0
	for _, l := range lengths {
		diff := l - mean
		variance += diff * diff
	}
	variance /= float64(len(lengths))
	stddev := math.Sqrt(variance)

	if mean == 0 {
		return 0
	}
	cv := stddev / mean
	if cv > 1.5 {
		return 1.0
	}
	if cv < 0.3 {
		return 0.0
	}
	return (cv - 0.3) / 1.2
}

func (s *Service) calcPerplexity(text string, lang string) float64 {
	words := splitWords(text)
	if len(words) < 3 {
		return 1.0
	}

	freq := make(map[string]int)
	for _, w := range words {
		w = strings.ToLower(w)
		freq[w]++
	}

	probs := 0.0
	for _, w := range words {
		w = strings.ToLower(w)
		count := freq[w]
		if count > 0 {
			probs += math.Log(float64(count) / float64(len(words)))
		}
	}

	if len(words) > 0 {
		perplexity := math.Exp(-probs / float64(len(words)))
		if perplexity > 100 {
			return 1.0
		}
		if perplexity < 1 {
			return 0.0
		}
		return perplexity / 100.0
	}
	return 0.5
}

func (s *Service) detectRepetition(text string) float64 {
	words := splitWords(text)
	if len(words) < 5 {
		return 0
	}

	wordFreq := make(map[string]int)
	for _, w := range words {
		w = strings.ToLower(strings.Trim(w, ".,!?;:\"'()[]{}"))
		if w != "" {
			wordFreq[w]++
		}
	}

	repeated := 0
	total := 0
	for _, count := range wordFreq {
		total += count
		if count > 2 {
			repeated += count - 2
		}
	}

	if total == 0 {
		return 0
	}
	score := float64(repeated) / float64(total)
	if score > 1 {
		return 1.0
	}
	return score
}

var passiveVoicePatterns = map[string]*regexp.Regexp{
	"pt": regexp.MustCompile(`(?i)\b(foi|foram|é|são|era|eram|será|serão|seria|seriam|sendo|têm sido|tem sido)\s+\w+[dms]\b`),
	"en": regexp.MustCompile(`(?i)\b(was|were|is|are|been|being|be|will be|would be|could be|should be)\s+\w+ed\b|\b(has been|have been|had been)\s+\w+ed\b`),
}

func (s *Service) detectPassiveVoice(text string, lang string) float64 {
	pat, ok := passiveVoicePatterns[lang]
	if !ok {
		return 0
	}

	sentences := splitSentences(text)
	if len(sentences) == 0 {
		return 0
	}

	passiveCount := 0
	for _, sentence := range sentences {
		if pat.MatchString(sentence) {
			passiveCount++
		}
	}

	score := float64(passiveCount) / float64(len(sentences))
	if score > 1 {
		return 1.0
	}
	return score
}

func (s *Service) analyzeRhythm(text string) float64 {
	sentences := splitSentences(text)
	if len(sentences) < 3 {
		return 0.5
	}

	variations := 0
	total := 0
	for i := 1; i < len(sentences); i++ {
		w1 := len(strings.Fields(sentences[i-1]))
		w2 := len(strings.Fields(sentences[i]))
		total++
		if absInt(w2-w1) > 4 {
			variations++
		}
	}

	if total == 0 {
		return 0.5
	}
	ratio := float64(variations) / float64(total)
	if ratio > 0.6 {
		return 1.0
	}
	if ratio < 0.2 {
		return 0.0
	}
	return (ratio - 0.2) / 0.4
}

func (s *Service) analyzeFlow(text string) float64 {
	paragraphs := splitParagraphs(text)
	if len(paragraphs) < 2 {
		return 0.5
	}

	connectors := 0
	totalPara := len(paragraphs) - 1

	ptConnectors := regexp.MustCompile(`(?i)\b(Além disso|Portanto|Contudo|Porém|Todavia|Consequentemente|Ademais|Outrossim|Entretanto|Assim|Dessa forma|Por exemplo|Primeiramente|Finalmente)\b`)
	enConnectors := regexp.MustCompile(`(?i)\b(Furthermore|Moreover|However|Therefore|Thus|Consequently|Nevertheless|Nonetheless|Additionally|In addition|For example|Firstly|Finally|In conclusion)\b`)

	for i := 1; i < len(paragraphs); i++ {
		sentences := splitSentences(paragraphs[i])
		if len(sentences) > 0 {
			if ptConnectors.MatchString(sentences[0]) || enConnectors.MatchString(sentences[0]) {
				connectors++
			}
		}
	}

	score := float64(connectors) / float64(totalPara)
	if score > 1 {
		return 1.0
	}
	return score
}

func splitSentences(text string) []string {
	text = strings.ReplaceAll(text, "\n\n", " ")
	re := regexp.MustCompile(`[.!?]+`)
	parts := re.Split(text, -1)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func splitParagraphs(text string) []string {
	parts := strings.Split(text, "\n\n")
	if len(parts) == 1 {
		parts = strings.Split(text, "\n")
	}
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func splitWords(text string) []string {
	return strings.Fields(text)
}

func countWords(text string) int {
	return len(strings.Fields(text))
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func parseJSON(s string) map[string]interface{} {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(s), &m); err != nil {
		return nil
	}
	return m
}
