package writer

import (
	"context"
	"encoding/json"
	"fmt"
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

func (s *Service) CreateJob(ctx context.Context, siteID, userID uuid.UUID, req CreateArticleJobRequest) (*ArticleJob, error) {
	if req.Headline == "" {
		return nil, ErrHeadlineRequired
	}
	lang := strings.ToLower(req.Language)
	if lang != "pt" && lang != "en" {
		return nil, ErrInvalidLanguage
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	jobID := uuid.New()

	var styleID *uuid.UUID
	styleName := ""
	if req.StyleSlug != "" {
		var sid uuid.UUID
		var sn string
		err = p.QueryRow(ctx,
			`SELECT id, name FROM writing_styles WHERE site_id = $1 AND slug = $2`,
			siteID, req.StyleSlug,
		).Scan(&sid, &sn)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, ErrStyleNotFound
			}
			return nil, fmt.Errorf("failed to lookup style: %w", err)
		}
		styleID = &sid
		styleName = sn
	} else {
		err = p.QueryRow(ctx,
			`SELECT id, name FROM writing_styles WHERE site_id = $1 AND is_default = true LIMIT 1`,
			siteID,
		).Scan(&styleID, &styleName)
		if err != nil && err != pgx.ErrNoRows {
			return nil, fmt.Errorf("failed to lookup default style: %w", err)
		}
	}

	_, err = p.Exec(ctx,
		`INSERT INTO article_jobs (id, site_id, research_job_id, style_id, style_name, language, status,
		 headline, seo_title, slug, meta_description, target_audience, tone, formality, seo_goal, desired_size,
		 created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,'draft',$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`,
		jobID, siteID, req.ResearchJobID, styleID, styleName, lang,
		req.Headline, req.SEOTitle, req.Slug, req.MetaDescription,
		req.TargetAudience, req.Tone, req.Formality, req.SEOGoal, req.DesiredSize,
		userID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create article job: %w", err)
	}

	job := &ArticleJob{
		ID:              jobID,
		SiteID:          siteID,
		ResearchJobID:   req.ResearchJobID,
		StyleID:         styleID,
		StyleName:       styleName,
		Language:        lang,
		Status:          JobStatusDraft,
		Headline:        req.Headline,
		SEOTitle:        req.SEOTitle,
		Slug:            req.Slug,
		MetaDescription: req.MetaDescription,
		TargetAudience:  req.TargetAudience,
		Tone:            req.Tone,
		Formality:       req.Formality,
		SEOGoal:         req.SEOGoal,
		DesiredSize:     req.DesiredSize,
		CreatedBy:       &userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("writer.job.created"),
		EntityType: "article_job",
		EntityID:   &jobID,
		Payload:    map[string]interface{}{"headline": req.Headline, "language": lang, "style": styleName},
	})

	s.fireEvent(ctx, EventWriterJobCreated, map[string]interface{}{
		"job_id":   jobID.String(),
		"site_id":  siteID.String(),
		"headline": req.Headline,
	}, siteID)

	return job, nil
}

func (s *Service) GetJob(ctx context.Context, siteID, jobID uuid.UUID) (*ArticleJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var j ArticleJob
	err = p.QueryRow(ctx,
		`SELECT id, site_id, research_job_id, style_id, COALESCE(style_name,''), language, status,
		        COALESCE(headline,''), COALESCE(seo_title,''), COALESCE(slug,''),
		        COALESCE(meta_description,''), COALESCE(target_audience,''),
		        COALESCE(tone,'neutral'), COALESCE(formality,'neutral'),
		        COALESCE(seo_goal,''), COALESCE(desired_size,'medium'),
		        created_by, completed_at, COALESCE(error_message,''), created_at, updated_at
		 FROM article_jobs WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		jobID, siteID,
	).Scan(&j.ID, &j.SiteID, &j.ResearchJobID, &j.StyleID, &j.StyleName, &j.Language, &j.Status,
		&j.Headline, &j.SEOTitle, &j.Slug, &j.MetaDescription, &j.TargetAudience,
		&j.Tone, &j.Formality, &j.SEOGoal, &j.DesiredSize,
		&j.CreatedBy, &j.CompletedAt, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrWritingJobNotFound
		}
		return nil, fmt.Errorf("failed to get article job: %w", err)
	}

	return &j, nil
}

func (s *Service) GetJobDetail(ctx context.Context, siteID, jobID uuid.UUID) (*ArticleJobDetail, error) {
	job, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	detail := &ArticleJobDetail{ArticleJob: *job}

	outline, _ := s.ListOutline(ctx, jobID)
	detail.Outline = outline

	sections, _ := s.ListSections(ctx, jobID)
	detail.Sections = sections

	versions, _ := s.ListVersions(ctx, jobID)
	detail.Versions = versions

	return detail, nil
}

func (s *Service) ListJobs(ctx context.Context, siteID uuid.UUID, status JobStatus) ([]ArticleJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var rows pgx.Rows
	if status == "" {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, research_job_id, style_id, COALESCE(style_name,''), language, status,
			        COALESCE(headline,''), COALESCE(seo_title,''), COALESCE(slug,''),
			        COALESCE(meta_description,''), COALESCE(target_audience,''),
			        COALESCE(tone,'neutral'), COALESCE(formality,'neutral'),
			        COALESCE(seo_goal,''), COALESCE(desired_size,'medium'),
			        created_by, completed_at, COALESCE(error_message,''), created_at, updated_at
			 FROM article_jobs WHERE site_id = $1 AND deleted_at IS NULL ORDER BY created_at DESC`,
			siteID,
		)
	} else {
		rows, err = p.Query(ctx,
			`SELECT id, site_id, research_job_id, style_id, COALESCE(style_name,''), language, status,
			        COALESCE(headline,''), COALESCE(seo_title,''), COALESCE(slug,''),
			        COALESCE(meta_description,''), COALESCE(target_audience,''),
			        COALESCE(tone,'neutral'), COALESCE(formality,'neutral'),
			        COALESCE(seo_goal,''), COALESCE(desired_size,'medium'),
			        created_by, completed_at, COALESCE(error_message,''), created_at, updated_at
			 FROM article_jobs WHERE site_id = $1 AND status = $2 AND deleted_at IS NULL ORDER BY created_at DESC`,
			siteID, string(status),
		)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to list article jobs: %w", err)
	}
	defer rows.Close()

	var jobs []ArticleJob
	for rows.Next() {
		var j ArticleJob
		if err := rows.Scan(&j.ID, &j.SiteID, &j.ResearchJobID, &j.StyleID, &j.StyleName, &j.Language, &j.Status,
			&j.Headline, &j.SEOTitle, &j.Slug, &j.MetaDescription, &j.TargetAudience,
			&j.Tone, &j.Formality, &j.SEOGoal, &j.DesiredSize,
			&j.CreatedBy, &j.CompletedAt, &j.ErrorMessage, &j.CreatedAt, &j.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan article job: %w", err)
		}
		jobs = append(jobs, j)
	}
	if jobs == nil {
		jobs = []ArticleJob{}
	}
	return jobs, nil
}

func (s *Service) UpdateJob(ctx context.Context, siteID, jobID uuid.UUID, req UpdateArticleJobRequest) (*ArticleJob, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++
		if *req.Status == JobStatusApproved || *req.Status == JobStatusPublished {
			setClauses = append(setClauses, fmt.Sprintf("completed_at = $%d", argIdx))
			args = append(args, time.Now())
			argIdx++
		}
	}
	if req.Headline != nil {
		setClauses = append(setClauses, fmt.Sprintf("headline = $%d", argIdx))
		args = append(args, *req.Headline)
		argIdx++
	}
	if req.SEOTitle != nil {
		setClauses = append(setClauses, fmt.Sprintf("seo_title = $%d", argIdx))
		args = append(args, *req.SEOTitle)
		argIdx++
	}
	if req.Slug != nil {
		setClauses = append(setClauses, fmt.Sprintf("slug = $%d", argIdx))
		args = append(args, *req.Slug)
		argIdx++
	}
	if req.MetaDescription != nil {
		setClauses = append(setClauses, fmt.Sprintf("meta_description = $%d", argIdx))
		args = append(args, *req.MetaDescription)
		argIdx++
	}
	if req.StyleSlug != nil {
		var sid uuid.UUID
		var sn string
		err = p.QueryRow(ctx,
			`SELECT id, name FROM writing_styles WHERE site_id = $1 AND slug = $2`, siteID, *req.StyleSlug,
		).Scan(&sid, &sn)
		if err != nil {
			if err == pgx.ErrNoRows {
				return nil, ErrStyleNotFound
			}
			return nil, fmt.Errorf("failed to lookup style: %w", err)
		}
		setClauses = append(setClauses, fmt.Sprintf("style_id = $%d", argIdx))
		args = append(args, sid)
		argIdx++
		setClauses = append(setClauses, fmt.Sprintf("style_name = $%d", argIdx))
		args = append(args, sn)
		argIdx++
	}
	if req.Tone != nil {
		setClauses = append(setClauses, fmt.Sprintf("tone = $%d", argIdx))
		args = append(args, *req.Tone)
		argIdx++
	}
	if req.Formality != nil {
		setClauses = append(setClauses, fmt.Sprintf("formality = $%d", argIdx))
		args = append(args, *req.Formality)
		argIdx++
	}
	if req.SEOGoal != nil {
		setClauses = append(setClauses, fmt.Sprintf("seo_goal = $%d", argIdx))
		args = append(args, *req.SEOGoal)
		argIdx++
	}
	if req.DesiredSize != nil {
		setClauses = append(setClauses, fmt.Sprintf("desired_size = $%d", argIdx))
		args = append(args, *req.DesiredSize)
		argIdx++
	}

	if len(setClauses) == 0 {
		return existing, nil
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE article_jobs SET %s WHERE id = $%d AND site_id = $%d AND deleted_at IS NULL`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, jobID, siteID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update article job: %w", err)
	}

	s.fireEvent(ctx, EventWriterJobUpdated, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return s.GetJob(ctx, siteID, jobID)
}

func (s *Service) DeleteJob(ctx context.Context, siteID, jobID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	_, err = s.GetJob(ctx, siteID, jobID)
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`UPDATE article_jobs SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		jobID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete article job: %w", err)
	}

	s.fireEvent(ctx, EventWriterJobCreated, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return nil
}

func (s *Service) CompleteJob(ctx context.Context, siteID, jobID uuid.UUID) error {
	_, err := s.UpdateJob(ctx, siteID, jobID, UpdateArticleJobRequest{
		Status: jobStatusPtr(JobStatusApproved),
	})
	if err != nil {
		return err
	}

	s.fireEvent(ctx, EventWriterJobCompleted, map[string]interface{}{
		"job_id":  jobID.String(),
		"site_id": siteID.String(),
	}, siteID)

	return nil
}

func (s *Service) CreateOutline(ctx context.Context, jobID uuid.UUID, req CreateOutlineRequest) ([]ArticleOutline, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var outlines []ArticleOutline
	for _, sec := range req.Sections {
		outlineID := uuid.New()
		now := time.Now()

		_, err = p.Exec(ctx,
			`INSERT INTO article_outlines (id, article_job_id, section_type, title, level, content, position, word_count_target, keywords, created_at)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
			outlineID, jobID, sec.SectionType, sec.Title, sec.Level, sec.Content,
			sec.Position, sec.WordCountTarget, sec.Keywords, now,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create outline section: %w", err)
		}

		outlines = append(outlines, ArticleOutline{
			ID:              outlineID,
			ArticleJobID:    jobID,
			SectionType:     sec.SectionType,
			Title:           sec.Title,
			Level:           sec.Level,
			Content:         sec.Content,
			Position:        sec.Position,
			WordCountTarget: sec.WordCountTarget,
			Keywords:        sec.Keywords,
			CreatedAt:       now,
		})
	}

	return outlines, nil
}

func (s *Service) ListOutline(ctx context.Context, jobID uuid.UUID) ([]ArticleOutline, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, article_job_id, section_type, title, level, COALESCE(content,''),
		        position, COALESCE(word_count_target,0), COALESCE(keywords,''), created_at
		 FROM article_outlines WHERE article_job_id = $1 ORDER BY position ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list outlines: %w", err)
	}
	defer rows.Close()

	var outlines []ArticleOutline
	for rows.Next() {
		var o ArticleOutline
		if err := rows.Scan(&o.ID, &o.ArticleJobID, &o.SectionType, &o.Title, &o.Level, &o.Content,
			&o.Position, &o.WordCountTarget, &o.Keywords, &o.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan outline: %w", err)
		}
		outlines = append(outlines, o)
	}
	if outlines == nil {
		outlines = []ArticleOutline{}
	}
	return outlines, nil
}

func (s *Service) CreateSection(ctx context.Context, jobID uuid.UUID, req CreateSectionRequest) (*ArticleSection, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	sectionID := uuid.New()
	now := time.Now()

	_, err = p.Exec(ctx,
		`INSERT INTO article_sections (id, article_job_id, outline_id, title, content, word_count, status, position, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,0,'pending',$6,$7,$7)`,
		sectionID, jobID, req.OutlineID, req.Title, req.Content, req.Position, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create section: %w", err)
	}

	section := &ArticleSection{
		ID:           sectionID,
		ArticleJobID: jobID,
		OutlineID:    req.OutlineID,
		Title:        req.Title,
		Content:      req.Content,
		WordCount:    0,
		Status:       SectionStatusPending,
		Position:     req.Position,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return section, nil
}

func (s *Service) GetSection(ctx context.Context, jobID, sectionID uuid.UUID) (*ArticleSection, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var sec ArticleSection
	err = p.QueryRow(ctx,
		`SELECT id, article_job_id, outline_id, COALESCE(title,''), COALESCE(content,''),
		        COALESCE(word_count,0), status, position, created_at, updated_at
		 FROM article_sections WHERE id = $1 AND article_job_id = $2`,
		sectionID, jobID,
	).Scan(&sec.ID, &sec.ArticleJobID, &sec.OutlineID, &sec.Title, &sec.Content,
		&sec.WordCount, &sec.Status, &sec.Position, &sec.CreatedAt, &sec.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrSectionNotFound
		}
		return nil, fmt.Errorf("failed to get section: %w", err)
	}

	return &sec, nil
}

func (s *Service) ListSections(ctx context.Context, jobID uuid.UUID) ([]ArticleSection, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, article_job_id, outline_id, COALESCE(title,''), COALESCE(content,''),
		        COALESCE(word_count,0), status, position, created_at, updated_at
		 FROM article_sections WHERE article_job_id = $1 ORDER BY position ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list sections: %w", err)
	}
	defer rows.Close()

	var sections []ArticleSection
	for rows.Next() {
		var sec ArticleSection
		if err := rows.Scan(&sec.ID, &sec.ArticleJobID, &sec.OutlineID, &sec.Title, &sec.Content,
			&sec.WordCount, &sec.Status, &sec.Position, &sec.CreatedAt, &sec.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan section: %w", err)
		}
		sections = append(sections, sec)
	}
	if sections == nil {
		sections = []ArticleSection{}
	}
	return sections, nil
}

func (s *Service) UpdateSection(ctx context.Context, jobID, sectionID uuid.UUID, req UpdateSectionRequest) (*ArticleSection, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.Content != nil {
		setClauses = append(setClauses, fmt.Sprintf("content = $%d", argIdx))
		args = append(args, *req.Content)
		argIdx++
	}
	if req.Status != nil {
		setClauses = append(setClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, string(*req.Status))
		argIdx++
	}
	if req.Position != nil {
		setClauses = append(setClauses, fmt.Sprintf("position = $%d", argIdx))
		args = append(args, *req.Position)
		argIdx++
	}
	if req.Content != nil {
		setClauses = append(setClauses, fmt.Sprintf("word_count = $%d", argIdx))
		args = append(args, countWords(*req.Content))
		argIdx++
	}

	if len(setClauses) == 0 {
		return s.GetSection(ctx, jobID, sectionID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE article_sections SET %s WHERE id = $%d AND article_job_id = $%d`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, sectionID, jobID)

	_, err = p.Exec(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update section: %w", err)
	}

	return s.GetSection(ctx, jobID, sectionID)
}

func (s *Service) CreateVersion(ctx context.Context, jobID, userID uuid.UUID, changeLog string) (*ArticleVersion, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var currentVersion int
	err = p.QueryRow(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM article_versions WHERE article_job_id = $1`, jobID,
	).Scan(&currentVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get current version: %w", err)
	}

	job, err := s.GetJob(ctx, jobID, jobID)
	if err != nil {
		return nil, err
	}

	sections, err := s.ListSections(ctx, jobID)
	if err != nil {
		return nil, err
	}

	sectionsJSON, _ := json.Marshal(sections)
	contentJSON := sectionsJSON
	metadata := map[string]interface{}{
		"style":         job.StyleName,
		"language":      job.Language,
		"tone":          job.Tone,
		"formality":     job.Formality,
		"seo_goal":      job.SEOGoal,
		"desired_size":  job.DesiredSize,
		"target_audience": job.TargetAudience,
		"section_count": len(sections),
	}
	metadataJSON, _ := json.Marshal(metadata)

	versionID := uuid.New()
	newVersion := currentVersion + 1
	now := time.Now()

	_, err = p.Exec(ctx,
		`INSERT INTO article_versions (id, article_job_id, version, headline, seo_title, slug, meta_description,
		 sections, content, metadata, summary, change_log, created_by, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9::jsonb,$10::jsonb,$11,$12,$13,$14)`,
		versionID, jobID, newVersion, job.Headline, job.SEOTitle, job.Slug, job.MetaDescription,
		string(sectionsJSON), string(contentJSON), string(metadataJSON),
		job.Headline, changeLog, userID, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create version: %w", err)
	}

	version := &ArticleVersion{
		ID:              versionID,
		ArticleJobID:    jobID,
		Version:         newVersion,
		Headline:        job.Headline,
		SEOTitle:        job.SEOTitle,
		Slug:            job.Slug,
		MetaDescription: job.MetaDescription,
		Metadata:        metadata,
		Summary:         job.Headline,
		ChangeLog:       changeLog,
		CreatedBy:       &userID,
		CreatedAt:       now,
	}
	if len(sections) > 0 {
		version.Sections = make([]interface{}, len(sections))
		for i, s := range sections {
			version.Sections[i] = s
		}
	}

	s.fireEvent(ctx, EventWriterVersionCreated, map[string]interface{}{
		"job_id":       jobID.String(),
		"version_id":   versionID.String(),
		"version":      newVersion,
		"change_log":   changeLog,
	}, uuid.Nil)

	return version, nil
}

func (s *Service) ListVersions(ctx context.Context, jobID uuid.UUID) ([]ArticleVersion, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, article_job_id, version, COALESCE(headline,''), COALESCE(seo_title,''),
		        COALESCE(slug,''), COALESCE(meta_description,''),
		        COALESCE(sections::text,'[]'), COALESCE(content::text,'[]'),
		        COALESCE(metadata::text,'{}'), COALESCE(summary,''), COALESCE(change_log,''),
		        created_by, created_at
		 FROM article_versions WHERE article_job_id = $1 ORDER BY version DESC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}
	defer rows.Close()

	var versions []ArticleVersion
	for rows.Next() {
		var v ArticleVersion
		var sectionsStr, contentStr, metadataStr string
		if err := rows.Scan(&v.ID, &v.ArticleJobID, &v.Version, &v.Headline, &v.SEOTitle,
			&v.Slug, &v.MetaDescription, &sectionsStr, &contentStr, &metadataStr,
			&v.Summary, &v.ChangeLog, &v.CreatedBy, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan version: %w", err)
		}
		if len(sectionsStr) > 0 {
			_ = json.Unmarshal([]byte(sectionsStr), &v.Sections)
		}
		if len(contentStr) > 0 {
			_ = json.Unmarshal([]byte(contentStr), &v.Content)
		}
		if len(metadataStr) > 0 {
			_ = json.Unmarshal([]byte(metadataStr), &v.Metadata)
		}
		if v.Sections == nil {
			v.Sections = []interface{}{}
		}
		if v.Content == nil {
			v.Content = []interface{}{}
		}
		if v.Metadata == nil {
			v.Metadata = make(map[string]interface{})
		}
		versions = append(versions, v)
	}
	if versions == nil {
		versions = []ArticleVersion{}
	}
	return versions, nil
}

func (s *Service) GetVersion(ctx context.Context, jobID, versionID uuid.UUID) (*ArticleVersion, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var v ArticleVersion
	var sectionsStr, contentStr, metadataStr string
	err = p.QueryRow(ctx,
		`SELECT id, article_job_id, version, COALESCE(headline,''), COALESCE(seo_title,''),
		        COALESCE(slug,''), COALESCE(meta_description,''),
		        COALESCE(sections::text,'[]'), COALESCE(content::text,'[]'),
		        COALESCE(metadata::text,'{}'), COALESCE(summary,''), COALESCE(change_log,''),
		        created_by, created_at
		 FROM article_versions WHERE id = $1 AND article_job_id = $2`,
		versionID, jobID,
	).Scan(&v.ID, &v.ArticleJobID, &v.Version, &v.Headline, &v.SEOTitle,
		&v.Slug, &v.MetaDescription, &sectionsStr, &contentStr, &metadataStr,
		&v.Summary, &v.ChangeLog, &v.CreatedBy, &v.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	if len(sectionsStr) > 0 {
		_ = json.Unmarshal([]byte(sectionsStr), &v.Sections)
	}
	if len(contentStr) > 0 {
		_ = json.Unmarshal([]byte(contentStr), &v.Content)
	}
	if len(metadataStr) > 0 {
		_ = json.Unmarshal([]byte(metadataStr), &v.Metadata)
	}
	if v.Sections == nil {
		v.Sections = []interface{}{}
	}
	if v.Content == nil {
		v.Content = []interface{}{}
	}
	if v.Metadata == nil {
		v.Metadata = make(map[string]interface{})
	}

	return &v, nil
}

func (s *Service) RestoreVersion(ctx context.Context, jobID, versionID, userID uuid.UUID) (*ArticleVersion, error) {
	version, err := s.GetVersion(ctx, jobID, versionID)
	if err != nil {
		return nil, err
	}

	sections, err := s.ListSections(ctx, jobID)
	if err != nil {
		return nil, err
	}

	if len(version.Sections) > 0 {
		data, _ := json.Marshal(version.Sections)
		var restoredSections []ArticleSection
		if err := json.Unmarshal(data, &restoredSections); err == nil {
			for _, rs := range restoredSections {
				_, _ = s.UpdateSection(ctx, jobID, rs.ID, UpdateSectionRequest{
					Title:   &rs.Title,
					Content: &rs.Content,
					Status:  &rs.Status,
				})
			}
		}
		_ = sections
	}

	s.fireEvent(ctx, EventWriterVersionRestored, map[string]interface{}{
		"job_id":       jobID.String(),
		"version_id":   versionID.String(),
		"version":      version.Version,
	}, uuid.Nil)

	return version, nil
}

func (s *Service) ListStyles(ctx context.Context, siteID uuid.UUID) ([]WritingStyle, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT id, site_id, name, slug, COALESCE(description,''), COALESCE(config::text,'{}'),
		        is_default, created_at, updated_at
		 FROM writing_styles WHERE site_id = $1 ORDER BY name ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list styles: %w", err)
	}
	defer rows.Close()

	var styles []WritingStyle
	for rows.Next() {
		var ws WritingStyle
		var configStr string
		if err := rows.Scan(&ws.ID, &ws.SiteID, &ws.Name, &ws.Slug, &ws.Description,
			&configStr, &ws.IsDefault, &ws.CreatedAt, &ws.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan style: %w", err)
		}
		if len(configStr) > 0 {
			_ = json.Unmarshal([]byte(configStr), &ws.Config)
		}
		if ws.Config == nil {
			ws.Config = make(map[string]interface{})
		}
		styles = append(styles, ws)
	}
	if styles == nil {
		styles = []WritingStyle{}
	}
	return styles, nil
}

func countWords(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Fields(s))
}

func jobStatusPtr(s JobStatus) *JobStatus { return &s }
