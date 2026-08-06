package translation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/ai"
	"nexora/internal/kernel"
	"nexora/internal/modules/posts"
	"nexora/internal/modules/publisher"
	"nexora/internal/modules/research"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	"nexora/internal/pkg/logger"
)

type Service struct {
	cfg            *config.Config
	log            *logger.Logger
	db             *database.Database
	cache          *cache.Cache
	eventBus       *kernel.EventBus
	auditLog       *audit.Logger
	aiManager      *ai.Manager
	qualityChecker ai.QualityChecker
	postsSvc       *posts.Service
	publisherSvc   *publisher.Service
	researchSvc    *research.Service
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache) *Service {
	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}
	return &Service{
		cfg:      cfg,
		log:      log,
		db:       db,
		cache:    ch,
		auditLog: audit.New(pool, log),
	}
}

func (s *Service) SetEventBus(bus *kernel.EventBus) {
	s.eventBus = bus
}

func (s *Service) SetAIManager(m *ai.Manager) {
	s.aiManager = m
}

func (s *Service) SetQualityChecker(qc ai.QualityChecker) {
	s.qualityChecker = qc
}

func (s *Service) SetPostsSvc(svc *posts.Service) {
	s.postsSvc = svc
}

func (s *Service) SetPublisherSvc(svc *publisher.Service) {
	s.publisherSvc = svc
}

// SetResearchSvc wires the research service so the translation pipeline can
// reuse the cached research (fact base + briefing) of the source topic.
func (s *Service) SetResearchSvc(svc *research.Service) {
	s.researchSvc = svc
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

func validLanguage(lang string) bool {
	l := strings.ToLower(lang)
	return l == "pt" || l == "en"
}

func langName(lang string) string {
	if strings.ToLower(lang) == "pt" {
		return "Portuguese"
	}
	return "English"
}

// --- Jobs ---

func (s *Service) CreateJob(ctx context.Context, siteID, userID uuid.UUID, req CreateJobRequest) (*TranslationJob, error) {
	if strings.TrimSpace(req.Title) == "" {
		return nil, ErrTitleRequired
	}
	if req.TargetSiteID == uuid.Nil {
		return nil, ErrTargetSiteRequired
	}
	srcLang := strings.ToLower(req.SourceLanguage)
	if srcLang == "" {
		detected, _ := DetectLanguage(req.Title + " " + req.Content)
		srcLang = detected
	}
	tgtLang := strings.ToLower(req.TargetLanguage)
	if tgtLang == "" {
		tgtLang = "en"
	}
	if !validLanguage(srcLang) || !validLanguage(tgtLang) {
		return nil, ErrInvalidLanguage
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	job := &TranslationJob{
		ID:             uuid.New(),
		SiteID:         siteID,
		ProjectID:      req.ProjectID,
		SourcePostID:   req.SourcePostID,
		TargetSiteID:   req.TargetSiteID,
		SourceLanguage: srcLang,
		TargetLanguage: tgtLang,
		Title:          req.Title,
		Content:        req.Content,
		Status:         JobStatusPending,
		CreatedBy:      &userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err = p.Exec(ctx,
		`INSERT INTO translation_jobs (id, site_id, project_id, source_post_id, target_site_id, source_language, target_language, title, content, status, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		job.ID, siteID, req.ProjectID, req.SourcePostID, req.TargetSiteID, srcLang, tgtLang,
		req.Title, req.Content, string(JobStatusPending), userID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create translation job: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("translation.job.created"),
		EntityType: "translation_job",
		EntityID:   &job.ID,
		Payload:    map[string]interface{}{"title": req.Title, "from": srcLang, "to": tgtLang},
	})

	s.fireEvent(ctx, EventTranslationJobCreated, map[string]interface{}{
		"job_id":    job.ID.String(),
		"site_id":   siteID.String(),
		"title":     req.Title,
		"from":      srcLang,
		"to":        tgtLang,
	}, siteID)

	return job, nil
}

const jobColumns = `id, site_id, project_id, source_post_id, target_site_id, source_language,
	target_language, title, content, status, current_stage, translation_score,
	published_post_id, publication_id, error_message, created_by, created_at, updated_at, completed_at`

func scanJob(row pgx.Row) (*TranslationJob, error) {
	var j TranslationJob
	var currentStage, scoreJSON, errMsg *string
	var projectID, sourcePostID, publishedPostID, publicationID, createdBy *uuid.UUID
	var completedAt *time.Time
	var status string

	err := row.Scan(&j.ID, &j.SiteID, &projectID, &sourcePostID, &j.TargetSiteID,
		&j.SourceLanguage, &j.TargetLanguage, &j.Title, &j.Content, &status,
		&currentStage, &scoreJSON, &publishedPostID, &publicationID, &errMsg,
		&createdBy, &j.CreatedAt, &j.UpdatedAt, &completedAt)
	if err != nil {
		return nil, err
	}
	j.Status = JobStatus(status)
	j.ProjectID = projectID
	j.SourcePostID = sourcePostID
	j.PublishedPostID = publishedPostID
	j.PublicationID = publicationID
	j.CreatedBy = createdBy
	j.CompletedAt = completedAt
	if currentStage != nil {
		st := StageType(*currentStage)
		j.CurrentStage = &st
	}
	if scoreJSON != nil && *scoreJSON != "" {
		var sc TranslationScore
		if err := json.Unmarshal([]byte(*scoreJSON), &sc); err == nil {
			j.TranslationScore = &sc
		}
	}
	if errMsg != nil {
		j.ErrorMessage = *errMsg
	}
	return &j, nil
}

func (s *Service) GetJob(ctx context.Context, siteID, jobID uuid.UUID) (*TranslationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	job, err := scanJob(p.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM translation_jobs WHERE id = $1 AND site_id = $2`, jobID, siteID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	stages, err := s.loadStages(ctx, p, jobID)
	if err != nil {
		return nil, err
	}
	job.Stages = stages
	return job, nil
}

func (s *Service) ListJobs(ctx context.Context, siteID uuid.UUID, status string, limit, offset int) ([]TranslationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	query := `SELECT ` + jobColumns + ` FROM translation_jobs WHERE site_id = $1`
	args := []interface{}{siteID}
	if status != "" {
		query += ` AND status = $2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) +
		` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	jobs := make([]TranslationJob, 0, 16)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobs = append(jobs, *job)
	}
	return jobs, rows.Err()
}

// StartJob validates the job and launches the pipeline asynchronously.
func (s *Service) StartJob(ctx context.Context, siteID, jobID uuid.UUID) (*TranslationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	job, err := scanJob(p.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM translation_jobs WHERE id = $1 AND site_id = $2 FOR UPDATE`, jobID, siteID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	if job.Status == JobStatusRunning {
		return job, ErrInvalidStatus
	}

	_, err = p.Exec(ctx,
		`UPDATE translation_jobs SET status = $3, error_message = NULL, updated_at = NOW() WHERE id = $1 AND site_id = $2`,
		jobID, siteID, string(JobStatusRunning))
	if err != nil {
		return nil, err
	}
	job.Status = JobStatusRunning
	job.ErrorMessage = ""

	s.ensureStages(ctx, p, jobID)

	go s.executePipeline(context.Background(), siteID, jobID)
	return job, nil
}

func (s *Service) CancelJob(ctx context.Context, siteID, jobID uuid.UUID) (*TranslationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	job, err := scanJob(p.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM translation_jobs WHERE id = $1 AND site_id = $2`, jobID, siteID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	if job.Status != JobStatusRunning && job.Status != JobStatusPending {
		return nil, ErrInvalidStatus
	}
	_, err = p.Exec(ctx,
		`UPDATE translation_jobs SET status = $3, updated_at = NOW() WHERE id = $1 AND site_id = $2`,
		jobID, siteID, string(JobStatusCancelled))
	if err != nil {
		return nil, err
	}
	job.Status = JobStatusCancelled
	return job, nil
}

func (s *Service) GetScore(ctx context.Context, siteID, jobID uuid.UUID) (*TranslationScore, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	var scoreJSON *string
	err = p.QueryRow(ctx,
		`SELECT translation_score FROM translation_jobs WHERE id = $1 AND site_id = $2`, jobID, siteID).Scan(&scoreJSON)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	if scoreJSON == nil || *scoreJSON == "" {
		return nil, nil
	}
	var sc TranslationScore
	if err := json.Unmarshal([]byte(*scoreJSON), &sc); err != nil {
		return nil, err
	}
	return &sc, nil
}

// --- Stages ---

func (s *Service) loadStages(ctx context.Context, p database.Pool, jobID uuid.UUID) ([]TranslationStage, error) {
	rows, err := p.Query(ctx,
		`SELECT id, translation_job_id, stage, status, score, attempt, feedback, result, created_at, updated_at, completed_at
		 FROM translation_stages WHERE translation_job_id = $1 ORDER BY created_at, attempt`, jobID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stages := make([]TranslationStage, 0, len(StageOrder))
	for rows.Next() {
		var st TranslationStage
		var stage, status, resultJSON string
		var score *float64
		var feedback *string
		var completedAt *time.Time
		if err := rows.Scan(&st.ID, &st.TranslationJobID, &stage, &status, &score, &st.Attempt,
			&feedback, &resultJSON, &st.CreatedAt, &st.UpdatedAt, &completedAt); err != nil {
			return nil, err
		}
		st.Stage = StageType(stage)
		st.Status = StageStatus(status)
		st.Score = score
		st.CompletedAt = completedAt
		if feedback != nil {
			st.Feedback = *feedback
		}
		if resultJSON != "" {
			var r StageResult
			if err := json.Unmarshal([]byte(resultJSON), &r); err == nil {
				st.Result = r
			}
		}
		stages = append(stages, st)
	}
	return stages, rows.Err()
}

func (s *Service) GetStages(ctx context.Context, siteID, jobID uuid.UUID) ([]TranslationStage, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	var exists bool
	err = p.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM translation_jobs WHERE id = $1 AND site_id = $2)`, jobID, siteID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrJobNotFound
	}
	return s.loadStages(ctx, p, jobID)
}

// ensureStages creates the four pipeline stage rows if they do not exist yet.
func (s *Service) ensureStages(ctx context.Context, p database.Pool, jobID uuid.UUID) {
	existing := make(map[StageType]bool)
	rows, err := p.Query(ctx,
		`SELECT DISTINCT stage FROM translation_stages WHERE translation_job_id = $1`, jobID)
	if err == nil {
		for rows.Next() {
			var stage string
			if rows.Scan(&stage) == nil {
				existing[StageType(stage)] = true
			}
		}
		rows.Close()
	}
	now := time.Now()
	for _, stage := range StageOrder {
		if existing[stage] {
			continue
		}
		p.Exec(ctx,
			`INSERT INTO translation_stages (id, translation_job_id, stage, status, attempt, created_at, updated_at)
			 VALUES ($1, $2, $3, 'pending', 1, $4, $4)`,
			uuid.New(), jobID, string(stage), now)
	}
}

func (s *Service) updateStage(ctx context.Context, p database.Pool, st *TranslationStage) error {
	score := st.Score
	feedback := st.Feedback
	if feedback == "" {
		feedback = ""
	}
	resultJSON := "{}"
	if b, err := json.Marshal(st.Result); err == nil {
		resultJSON = string(b)
	}
	_, err := p.Exec(ctx,
		`UPDATE translation_stages SET status = $3, score = $4, feedback = $5, result = $6::jsonb, completed_at = $7, updated_at = NOW()
		 WHERE id = $1 AND translation_job_id = $2`,
		st.ID, st.TranslationJobID, string(st.Status), score, feedback, resultJSON, st.CompletedAt)
	return err
}

func (s *Service) updateJobStatus(ctx context.Context, p database.Pool, jobID, siteID uuid.UUID,
	status JobStatus, currentStage *StageType, score *TranslationScore,
	publishedPostID, publicationID *uuid.UUID, errMsg string) error {
	var scoreJSON *string
	if score != nil {
		b, err := json.Marshal(score)
		if err == nil {
			s := string(b)
			scoreJSON = &s
		}
	}
	var completedAt *time.Time
	if status == JobStatusCompleted {
		t := time.Now()
		completedAt = &t
	}
	var stageStr *string
	if currentStage != nil {
		s := string(*currentStage)
		stageStr = &s
	}
	var errStr *string
	if errMsg != "" {
		e := errMsg
		errStr = &e
	}
	_, err := p.Exec(ctx,
		`UPDATE translation_jobs SET status = $3, current_stage = $4, translation_score = $5,
		 published_post_id = $6, publication_id = $7, error_message = $8, completed_at = $9, updated_at = NOW()
		 WHERE id = $1 AND site_id = $2`,
		jobID, siteID, string(status), stageStr, scoreJSON, publishedPostID, publicationID, errStr, completedAt)
	return err
}

func (s *Service) findStage(stages []TranslationStage, stageType StageType) *TranslationStage {
	for i := range stages {
		if stages[i].Stage == stageType {
			return &stages[i]
		}
	}
	return nil
}

// ApproveStage accepts a rejected current stage and continues the pipeline to
// the next stage.
func (s *Service) ApproveStage(ctx context.Context, siteID, stageID uuid.UUID) (*TranslationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	var jobID uuid.UUID
	err = p.QueryRow(ctx,
		`SELECT translation_job_id FROM translation_stages WHERE id = $1`, stageID).Scan(&jobID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStageNotFound
		}
		return nil, err
	}

	job, err := scanJob(p.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM translation_jobs WHERE id = $1 AND site_id = $2 FOR UPDATE`, jobID, siteID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	if job.Status != JobStatusReview {
		return nil, ErrInvalidStatus
	}

	_, err = p.Exec(ctx,
		`UPDATE translation_stages SET status = 'completed', feedback = COALESCE(feedback, 'approved'), updated_at = NOW() WHERE id = $1`,
		stageID)
	if err != nil {
		return nil, err
	}

	nextStage := nextStageType(job.CurrentStage)
	if nextStage == nil {
		// Approving the last stage without publish having run: just complete.
		_, err = p.Exec(ctx,
			`UPDATE translation_jobs SET status = 'completed', completed_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2`,
			jobID, siteID)
		if err != nil {
			return nil, err
		}
		job.Status = JobStatusCompleted
		return job, nil
	}

	if err := s.updateJobStatus(ctx, p, jobID, siteID, JobStatusRunning, nextStage, nil, nil, nil, ""); err != nil {
		return nil, err
	}
	go s.executePipeline(context.Background(), siteID, jobID)
	job.Status = JobStatusRunning
	job.CurrentStage = nextStage
	return job, nil
}

// RejectStage rejects the current (rejected) stage and returns the job to the
// previous pipeline stage, which is re-run.
func (s *Service) RejectStage(ctx context.Context, siteID, stageID uuid.UUID, feedback string) (*TranslationJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	var jobID uuid.UUID
	var stageStr string
	err = p.QueryRow(ctx,
		`SELECT translation_job_id, stage FROM translation_stages WHERE id = $1`, stageID).Scan(&jobID, &stageStr)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStageNotFound
		}
		return nil, err
	}

	job, err := scanJob(p.QueryRow(ctx,
		`SELECT `+jobColumns+` FROM translation_jobs WHERE id = $1 AND site_id = $2 FOR UPDATE`, jobID, siteID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrJobNotFound
		}
		return nil, err
	}
	if job.Status != JobStatusReview {
		return nil, ErrInvalidStatus
	}

	if feedback == "" {
		feedback = "rejected by reviewer"
	}
	_, err = p.Exec(ctx,
		`UPDATE translation_stages SET status = 'rejected', feedback = $3, updated_at = NOW() WHERE id = $1`,
		stageID, jobID, feedback)
	if err != nil {
		return nil, err
	}

	current := StageType(stageStr)
	prev := previousStageType(current)
	if prev == nil {
		return nil, ErrInvalidStatus
	}

	// Re-open the previous stage with a fresh attempt.
	_, err = p.Exec(ctx,
		`UPDATE translation_stages SET status = 'pending', attempt = attempt + 1, score = NULL, feedback = NULL, result = '{}'::jsonb, updated_at = NOW()
		 WHERE translation_job_id = $1 AND stage = $2`,
		jobID, string(*prev))
	if err != nil {
		return nil, err
	}

	if err := s.updateJobStatus(ctx, p, jobID, siteID, JobStatusRunning, prev, nil, nil, nil, ""); err != nil {
		return nil, err
	}

	s.fireEvent(ctx, EventTranslationStageRejected, map[string]interface{}{
		"job_id":   jobID.String(),
		"site_id":  siteID.String(),
		"stage":    stageStr,
		"feedback": feedback,
	}, siteID)

	go s.executePipeline(context.Background(), siteID, jobID)
	job.Status = JobStatusRunning
	job.CurrentStage = prev
	return job, nil
}

func nextStageType(current *StageType) *StageType {
	if current == nil {
		c := StageTranslate
		return &c
	}
	for i, st := range StageOrder {
		if st == *current && i+1 < len(StageOrder) {
			c := StageOrder[i+1]
			return &c
		}
	}
	return nil
}

func previousStageType(current StageType) *StageType {
	for i, st := range StageOrder {
		if st == current && i > 0 {
			c := StageOrder[i-1]
			return &c
		}
	}
	return nil
}

// --- Glossary ---

func (s *Service) CreateGlossaryTerm(ctx context.Context, siteID, userID uuid.UUID, req CreateGlossaryTermRequest) (*GlossaryTerm, error) {
	source := normalizeTerm(req.SourceTerm)
	target := normalizeTerm(req.TargetTerm)
	if source == "" || target == "" {
		return nil, ErrInvalidGlossary
	}
	srcLang := strings.ToLower(req.SourceLanguage)
	tgtLang := strings.ToLower(req.TargetLanguage)
	if srcLang == "" {
		srcLang = "pt"
	}
	if tgtLang == "" {
		tgtLang = "en"
	}
	if !validLanguage(srcLang) || !validLanguage(tgtLang) {
		return nil, ErrInvalidLanguage
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var exists bool
	err = p.QueryRow(ctx,
		`SELECT EXISTS(
			SELECT 1 FROM glossary_terms
			WHERE site_id = $1 AND project_id IS NOT DISTINCT FROM $2
			  AND LOWER(source_term) = LOWER($3) AND source_language = $4 AND target_language = $5)`,
		siteID, req.ProjectID, source, srcLang, tgtLang).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrGlossaryDuplicate
	}

	now := time.Now()
	term := &GlossaryTerm{
		ID:             uuid.New(),
		SiteID:         siteID,
		ProjectID:      req.ProjectID,
		SourceTerm:     source,
		TargetTerm:     target,
		SourceLanguage: srcLang,
		TargetLanguage: tgtLang,
		Forbidden:      req.Forbidden,
		Description:    req.Description,
		CreatedBy:      &userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err = p.Exec(ctx,
		`INSERT INTO glossary_terms (id, site_id, project_id, source_term, target_term, source_language, target_language, forbidden, description, created_by, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $11)`,
		term.ID, siteID, req.ProjectID, source, target, srcLang, tgtLang, req.Forbidden, req.Description, userID, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create glossary term: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("translation.glossary.created"),
		EntityType: "glossary_term",
		EntityID:   &term.ID,
		Payload:    map[string]interface{}{"source": source, "target": target, "forbidden": req.Forbidden},
	})

	s.fireEvent(ctx, EventGlossaryCreated, map[string]interface{}{
		"term_id":   term.ID.String(),
		"site_id":   siteID.String(),
		"source":    source,
		"target":    target,
	}, siteID)

	return term, nil
}

func (s *Service) ListGlossaryTerms(ctx context.Context, siteID uuid.UUID, projectID *uuid.UUID) ([]GlossaryTerm, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}
	rows, err := p.Query(ctx,
		`SELECT id, site_id, project_id, source_term, target_term, source_language, target_language, forbidden, description, created_by, created_at, updated_at
		 FROM glossary_terms
		 WHERE site_id = $1 AND project_id IS NOT DISTINCT FROM $2
		 ORDER BY source_term`, siteID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	terms := make([]GlossaryTerm, 0, 16)
	for rows.Next() {
		var t GlossaryTerm
		var desc, createdBy *string
		if err := rows.Scan(&t.ID, &t.SiteID, &t.ProjectID, &t.SourceTerm, &t.TargetTerm,
			&t.SourceLanguage, &t.TargetLanguage, &t.Forbidden, &desc, &createdBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		if desc != nil {
			t.Description = *desc
		}
		terms = append(terms, t)
	}
	return terms, rows.Err()
}

func (s *Service) UpdateGlossaryTerm(ctx context.Context, siteID, termID uuid.UUID, req UpdateGlossaryTermRequest) (*GlossaryTerm, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var exists bool
	err = p.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM glossary_terms WHERE id = $1 AND site_id = $2)`, termID, siteID).Scan(&exists)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrGlossaryNotFound
	}

	if req.SourceTerm != nil && strings.TrimSpace(*req.SourceTerm) == "" {
		return nil, ErrInvalidGlossary
	}
	if req.TargetTerm != nil && strings.TrimSpace(*req.TargetTerm) == "" {
		return nil, ErrInvalidGlossary
	}

	_, err = p.Exec(ctx,
		`UPDATE glossary_terms SET
			source_term = COALESCE($3, source_term),
			target_term = COALESCE($4, target_term),
			source_language = COALESCE($5, source_language),
			target_language = COALESCE($6, target_language),
			forbidden = COALESCE($7, forbidden),
			description = COALESCE($8, description),
			updated_at = NOW()
		 WHERE id = $1 AND site_id = $2`,
		termID, siteID, req.SourceTerm, req.TargetTerm, req.SourceLanguage, req.TargetLanguage,
		req.Forbidden, req.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to update glossary term: %w", err)
	}

	var t GlossaryTerm
	var desc *string
	err = p.QueryRow(ctx,
		`SELECT id, site_id, project_id, source_term, target_term, source_language, target_language, forbidden, description, created_by, created_at, updated_at
		 FROM glossary_terms WHERE id = $1 AND site_id = $2`, termID, siteID).
		Scan(&t.ID, &t.SiteID, &t.ProjectID, &t.SourceTerm, &t.TargetTerm, &t.SourceLanguage,
			&t.TargetLanguage, &t.Forbidden, &desc, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if desc != nil {
		t.Description = *desc
	}

	s.fireEvent(ctx, EventGlossaryUpdated, map[string]interface{}{
		"term_id": termID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return &t, nil
}

func (s *Service) DeleteGlossaryTerm(ctx context.Context, siteID, termID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}
	tag, err := p.Exec(ctx,
		`DELETE FROM glossary_terms WHERE id = $1 AND site_id = $2`, termID, siteID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrGlossaryNotFound
	}
	s.fireEvent(ctx, EventGlossaryDeleted, map[string]interface{}{
		"term_id": termID.String(),
		"site_id": siteID.String(),
	}, siteID)
	return nil
}
