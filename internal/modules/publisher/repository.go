package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/pkg/database"
)

type Repository struct {
	db database.Pool
}

func NewRepository(db database.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) checkDB() error {
	if r.db == nil {
		return ErrDatabaseNotAvail
	}
	return nil
}

// --- Publications ---

func (r *Repository) CreatePublication(ctx context.Context, p *Publication) error {
	if err := r.checkDB(); err != nil {
		return err
	}

	translationsJSON := "{}"
	if p.Translations != nil {
		b, _ := json.Marshal(p.Translations)
		translationsJSON = string(b)
	}
	multilingualJSON := "{}"
	if p.MultilingualURLs != nil {
		b, _ := json.Marshal(p.MultilingualURLs)
		multilingualJSON = string(b)
	}
	metadataJSON := "{}"
	if p.Metadata != nil {
		b, _ := json.Marshal(p.Metadata)
		metadataJSON = string(b)
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO publications (id, site_id, post_id, title, content, excerpt, slug, url,
		 canonical_url, language, translations, multilingual_urls, status, visibility,
		 author_id, published_by, published_at, unpublished_at, scheduled_at, is_featured,
		 meta_title, meta_description, og_image, featured_image_url, tags, categories,
		 word_count, reading_time, revision, checksum, source, metadata, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12::jsonb,$13,$14,$15,$16,$17,$18,$19,$20,
		 $21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32::jsonb,$33,$34,$34)`,
		p.ID, p.SiteID, p.PostID, p.Title, p.Content, p.Excerpt, p.Slug, p.URL,
		p.CanonicalURL, p.Language, translationsJSON, multilingualJSON, p.Status, p.Visibility,
		p.AuthorID, p.PublishedBy, p.PublishedAt, p.UnpublishedAt, p.ScheduledAt, p.IsFeatured,
		p.MetaTitle, p.MetaDescription, p.OgImage, p.FeaturedImageURL, p.Tags, p.Categories,
		p.WordCount, p.ReadingTime, p.Revision, p.Checksum, p.Source, metadataJSON, p.CreatedBy, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create publication: %w", err)
	}
	return nil
}

func (r *Repository) GetPublicationByID(ctx context.Context, siteID, pubID uuid.UUID) (*Publication, error) {
	if err := r.checkDB(); err != nil {
		return nil, err
	}

	var p Publication
	var translationsStr, multilingualStr, metadataStr string

	err := r.db.QueryRow(ctx,
		`SELECT id, site_id, post_id, title, COALESCE(content,''), COALESCE(excerpt,''), slug, url,
		        COALESCE(canonical_url,''), language, COALESCE(translations::text,'{}'), COALESCE(multilingual_urls::text,'{}'),
		        status, visibility, author_id, published_by, published_at, unpublished_at, scheduled_at, is_featured,
		        COALESCE(meta_title,''), COALESCE(meta_description,''), COALESCE(og_image,''),
		        COALESCE(featured_image_url,''), COALESCE(tags,'{}'), COALESCE(categories,'{}'),
		        COALESCE(word_count,0), COALESCE(reading_time,0), revision, COALESCE(checksum,''),
		        COALESCE(source,'manual'), COALESCE(metadata::text,'{}'), created_by, created_at, updated_at
		 FROM publications WHERE id = $1 AND site_id = $2`,
		pubID, siteID,
	).Scan(&p.ID, &p.SiteID, &p.PostID, &p.Title, &p.Content, &p.Excerpt, &p.Slug, &p.URL,
		&p.CanonicalURL, &p.Language, &translationsStr, &multilingualStr,
		&p.Status, &p.Visibility, &p.AuthorID, &p.PublishedBy, &p.PublishedAt, &p.UnpublishedAt,
		&p.ScheduledAt, &p.IsFeatured, &p.MetaTitle, &p.MetaDescription, &p.OgImage,
		&p.FeaturedImageURL, &p.Tags, &p.Categories, &p.WordCount, &p.ReadingTime, &p.Revision,
		&p.Checksum, &p.Source, &metadataStr, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPublicationNotFound
		}
		return nil, fmt.Errorf("failed to get publication: %w", err)
	}

	if len(translationsStr) > 0 {
		_ = json.Unmarshal([]byte(translationsStr), &p.Translations)
	}
	if len(multilingualStr) > 0 {
		_ = json.Unmarshal([]byte(multilingualStr), &p.MultilingualURLs)
	}
	if len(metadataStr) > 0 {
		_ = json.Unmarshal([]byte(metadataStr), &p.Metadata)
	}
	if p.Translations == nil {
		p.Translations = make(map[string]interface{})
	}
	if p.MultilingualURLs == nil {
		p.MultilingualURLs = make(map[string]interface{})
	}
	if p.Metadata == nil {
		p.Metadata = make(map[string]interface{})
	}

	return &p, nil
}

func (r *Repository) GetPublicationBySlug(ctx context.Context, siteID uuid.UUID, slug string) (*Publication, error) {
	if err := r.checkDB(); err != nil {
		return nil, err
	}

	var p Publication
	var translationsStr, multilingualStr, metadataStr string

	err := r.db.QueryRow(ctx,
		`SELECT id, site_id, post_id, title, COALESCE(content,''), COALESCE(excerpt,''), slug, url,
		        COALESCE(canonical_url,''), language, COALESCE(translations::text,'{}'), COALESCE(multilingual_urls::text,'{}'),
		        status, visibility, author_id, published_by, published_at, unpublished_at, scheduled_at, is_featured,
		        COALESCE(meta_title,''), COALESCE(meta_description,''), COALESCE(og_image,''),
		        COALESCE(featured_image_url,''), COALESCE(tags,'{}'), COALESCE(categories,'{}'),
		        COALESCE(word_count,0), COALESCE(reading_time,0), revision, COALESCE(checksum,''),
		        COALESCE(source,'manual'), COALESCE(metadata::text,'{}'), created_by, created_at, updated_at
		 FROM publications WHERE site_id = $1 AND slug = $2 AND status != 'deleted'`,
		siteID, slug,
	).Scan(&p.ID, &p.SiteID, &p.PostID, &p.Title, &p.Content, &p.Excerpt, &p.Slug, &p.URL,
		&p.CanonicalURL, &p.Language, &translationsStr, &multilingualStr,
		&p.Status, &p.Visibility, &p.AuthorID, &p.PublishedBy, &p.PublishedAt, &p.UnpublishedAt,
		&p.ScheduledAt, &p.IsFeatured, &p.MetaTitle, &p.MetaDescription, &p.OgImage,
		&p.FeaturedImageURL, &p.Tags, &p.Categories, &p.WordCount, &p.ReadingTime, &p.Revision,
		&p.Checksum, &p.Source, &metadataStr, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrPublicationNotFound
		}
		return nil, fmt.Errorf("failed to get publication by slug: %w", err)
	}

	if len(translationsStr) > 0 {
		_ = json.Unmarshal([]byte(translationsStr), &p.Translations)
	}
	if len(multilingualStr) > 0 {
		_ = json.Unmarshal([]byte(multilingualStr), &p.MultilingualURLs)
	}
	if len(metadataStr) > 0 {
		_ = json.Unmarshal([]byte(metadataStr), &p.Metadata)
	}

	return &p, nil
}

func (r *Repository) ListPublications(ctx context.Context, siteID uuid.UUID, status, language string, limit, offset int) ([]Publication, int, error) {
	if err := r.checkDB(); err != nil {
		return nil, 0, err
	}

	where := []string{"site_id = $1"}
	args := []interface{}{siteID}
	argIdx := 2

	if status != "" {
		where = append(where, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, status)
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

	var total int
	err := r.db.QueryRow(ctx,
		fmt.Sprintf(`SELECT COUNT(*) FROM publications WHERE %s`, strings.Join(where, " AND ")),
		args...,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count publications: %w", err)
	}

	query := fmt.Sprintf(
		`SELECT id, site_id, post_id, title, COALESCE(content,''), COALESCE(excerpt,''), slug, url,
		        COALESCE(canonical_url,''), language, COALESCE(translations::text,'{}'), COALESCE(multilingual_urls::text,'{}'),
		        status, visibility, author_id, published_by, published_at, unpublished_at, scheduled_at, is_featured,
		        COALESCE(meta_title,''), COALESCE(meta_description,''), COALESCE(og_image,''),
		        COALESCE(featured_image_url,''), COALESCE(tags,'{}'), COALESCE(categories,'{}'),
		        COALESCE(word_count,0), COALESCE(reading_time,0), revision, COALESCE(checksum,''),
		        COALESCE(source,'manual'), COALESCE(metadata::text,'{}'), created_by, created_at, updated_at
		 FROM publications WHERE %s ORDER BY created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list publications: %w", err)
	}
	defer rows.Close()

	var pubs []Publication
	for rows.Next() {
		var p Publication
		var translationsStr, multilingualStr, metadataStr string
		if err := rows.Scan(&p.ID, &p.SiteID, &p.PostID, &p.Title, &p.Content, &p.Excerpt, &p.Slug, &p.URL,
			&p.CanonicalURL, &p.Language, &translationsStr, &multilingualStr,
			&p.Status, &p.Visibility, &p.AuthorID, &p.PublishedBy, &p.PublishedAt, &p.UnpublishedAt,
			&p.ScheduledAt, &p.IsFeatured, &p.MetaTitle, &p.MetaDescription, &p.OgImage,
			&p.FeaturedImageURL, &p.Tags, &p.Categories, &p.WordCount, &p.ReadingTime, &p.Revision,
			&p.Checksum, &p.Source, &metadataStr, &p.CreatedBy, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("failed to scan publication: %w", err)
		}
		if len(translationsStr) > 0 {
			_ = json.Unmarshal([]byte(translationsStr), &p.Translations)
		}
		if len(multilingualStr) > 0 {
			_ = json.Unmarshal([]byte(multilingualStr), &p.MultilingualURLs)
		}
		if len(metadataStr) > 0 {
			_ = json.Unmarshal([]byte(metadataStr), &p.Metadata)
		}
		if p.Translations == nil {
			p.Translations = make(map[string]interface{})
		}
		if p.MultilingualURLs == nil {
			p.MultilingualURLs = make(map[string]interface{})
		}
		if p.Metadata == nil {
			p.Metadata = make(map[string]interface{})
		}
		pubs = append(pubs, p)
	}
	if pubs == nil {
		pubs = []Publication{}
	}
	return pubs, total, nil
}

func (r *Repository) UpdatePublication(ctx context.Context, pubID uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkDB(); err != nil {
		return err
	}

	if len(updates) == 0 {
		return nil
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for k, v := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, argIdx))
		args = append(args, v)
		argIdx++
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE publications SET %s WHERE id = $%d`,
		strings.Join(setClauses, ", "), argIdx,
	)
	args = append(args, pubID)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update publication: %w", err)
	}
	return nil
}

func (r *Repository) DeletePublication(ctx context.Context, siteID, pubID uuid.UUID) error {
	if err := r.checkDB(); err != nil {
		return err
	}

	tag, err := r.db.Exec(ctx,
		`DELETE FROM publications WHERE id = $1 AND site_id = $2`,
		pubID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete publication: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrPublicationNotFound
	}
	return nil
}

func (r *Repository) CheckDuplicateSlug(ctx context.Context, siteID uuid.UUID, slug string, excludeID *uuid.UUID) (bool, error) {
	if err := r.checkDB(); err != nil {
		return false, err
	}

	var exists bool
	if excludeID != nil {
		err := r.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM publications WHERE site_id = $1 AND slug = $2 AND id != $3 AND status != 'deleted')`,
			siteID, slug, *excludeID,
		).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("failed to check duplicate slug: %w", err)
		}
	} else {
		err := r.db.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM publications WHERE site_id = $1 AND slug = $2 AND status != 'deleted')`,
			siteID, slug,
		).Scan(&exists)
		if err != nil {
			return false, fmt.Errorf("failed to check duplicate slug: %w", err)
		}
	}
	return exists, nil
}

// --- History ---

func (r *Repository) CreateHistory(ctx context.Context, h *PublicationHistory) error {
	if err := r.checkDB(); err != nil {
		return err
	}

	changesJSON := "{}"
	if h.Changes != nil {
		b, _ := json.Marshal(h.Changes)
		changesJSON = string(b)
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO publication_history (id, publication_id, site_id, action, previous_status, new_status,
		 title, slug, changes, reason, performed_by, performed_at, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$12)`,
		h.ID, h.PublicationID, h.SiteID, h.Action, h.PreviousStatus, h.NewStatus,
		h.Title, h.Slug, changesJSON, h.Reason, h.PerformedBy, h.PerformedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create history: %w", err)
	}
	return nil
}

func (r *Repository) ListHistory(ctx context.Context, siteID, pubID uuid.UUID, limit, offset int) ([]PublicationHistory, error) {
	if err := r.checkDB(); err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := r.db.Query(ctx,
		`SELECT id, publication_id, site_id, action, COALESCE(previous_status,''), COALESCE(new_status,''),
		        COALESCE(title,''), COALESCE(slug,''), COALESCE(changes::text,'{}'), COALESCE(reason,''),
		        performed_by, performed_at, created_at
		 FROM publication_history WHERE site_id = $1 AND publication_id = $2
		 ORDER BY performed_at DESC LIMIT $3 OFFSET $4`,
		siteID, pubID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list history: %w", err)
	}
	defer rows.Close()

	var history []PublicationHistory
	for rows.Next() {
		var h PublicationHistory
		var changesStr string
		if err := rows.Scan(&h.ID, &h.PublicationID, &h.SiteID, &h.Action, &h.PreviousStatus, &h.NewStatus,
			&h.Title, &h.Slug, &changesStr, &h.Reason, &h.PerformedBy, &h.PerformedAt, &h.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan history: %w", err)
		}
		if len(changesStr) > 0 {
			_ = json.Unmarshal([]byte(changesStr), &h.Changes)
		}
		if h.Changes == nil {
			h.Changes = make(map[string]interface{})
		}
		history = append(history, h)
	}
	if history == nil {
		history = []PublicationHistory{}
	}
	return history, nil
}

// --- Queue ---

func (r *Repository) CreateQueueItem(ctx context.Context, q *QueueItem) error {
	if err := r.checkDB(); err != nil {
		return err
	}

	metadataJSON := "{}"
	if q.Metadata != nil {
		b, _ := json.Marshal(q.Metadata)
		metadataJSON = string(b)
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO publication_queue (id, site_id, publication_id, action, status, priority,
		 scheduled_for, started_at, completed_at, error_message, retry_count, max_retries,
		 metadata, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb,$14,$15,$15)`,
		q.ID, q.SiteID, q.PublicationID, q.Action, q.Status, q.Priority,
		q.ScheduledFor, q.StartedAt, q.CompletedAt, q.ErrorMessage, q.RetryCount, q.MaxRetries,
		metadataJSON, q.CreatedBy, q.CreatedAt, q.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create queue item: %w", err)
	}
	return nil
}

func (r *Repository) GetQueueItem(ctx context.Context, siteID, itemID uuid.UUID) (*QueueItem, error) {
	if err := r.checkDB(); err != nil {
		return nil, err
	}

	var q QueueItem
	var metadataStr string

	err := r.db.QueryRow(ctx,
		`SELECT id, site_id, publication_id, action, status, priority, scheduled_for,
		        started_at, completed_at, COALESCE(error_message,''), retry_count, max_retries,
		        COALESCE(metadata::text,'{}'), created_by, created_at, updated_at
		 FROM publication_queue WHERE id = $1 AND site_id = $2`,
		itemID, siteID,
	).Scan(&q.ID, &q.SiteID, &q.PublicationID, &q.Action, &q.Status, &q.Priority, &q.ScheduledFor,
		&q.StartedAt, &q.CompletedAt, &q.ErrorMessage, &q.RetryCount, &q.MaxRetries,
		&metadataStr, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrQueueItemNotFound
		}
		return nil, fmt.Errorf("failed to get queue item: %w", err)
	}

	if len(metadataStr) > 0 {
		_ = json.Unmarshal([]byte(metadataStr), &q.Metadata)
	}

	return &q, nil
}

func (r *Repository) ListQueue(ctx context.Context, siteID uuid.UUID, status string, limit, offset int) ([]QueueItem, error) {
	if err := r.checkDB(); err != nil {
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
		`SELECT id, site_id, publication_id, action, status, priority, scheduled_for,
		        started_at, completed_at, COALESCE(error_message,''), retry_count, max_retries,
		        COALESCE(metadata::text,'{}'), created_by, created_at, updated_at
		 FROM publication_queue WHERE %s ORDER BY priority ASC, created_at DESC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list queue: %w", err)
	}
	defer rows.Close()

	var items []QueueItem
	for rows.Next() {
		var q QueueItem
		var metadataStr string
		if err := rows.Scan(&q.ID, &q.SiteID, &q.PublicationID, &q.Action, &q.Status, &q.Priority, &q.ScheduledFor,
			&q.StartedAt, &q.CompletedAt, &q.ErrorMessage, &q.RetryCount, &q.MaxRetries,
			&metadataStr, &q.CreatedBy, &q.CreatedAt, &q.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan queue item: %w", err)
		}
		if len(metadataStr) > 0 {
			_ = json.Unmarshal([]byte(metadataStr), &q.Metadata)
		}
		items = append(items, q)
	}
	if items == nil {
		items = []QueueItem{}
	}
	return items, nil
}

func (r *Repository) UpdateQueueItem(ctx context.Context, itemID uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkDB(); err != nil {
		return err
	}

	if len(updates) == 0 {
		return nil
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for k, v := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, argIdx))
		args = append(args, v)
		argIdx++
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE publication_queue SET %s WHERE id = $%d`,
		strings.Join(setClauses, ", "), argIdx,
	)
	args = append(args, itemID)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update queue item: %w", err)
	}
	return nil
}

// --- Schedule ---

func (r *Repository) CreateSchedule(ctx context.Context, s *Schedule) error {
	if err := r.checkDB(); err != nil {
		return err
	}

	metadataJSON := "{}"
	if s.Metadata != nil {
		b, _ := json.Marshal(s.Metadata)
		metadataJSON = string(b)
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO publication_schedule (id, site_id, publication_id, scheduled_at, action, status,
		 recurrence, recurrence_end, notify_on_publish, notify_users, metadata, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13,$13)`,
		s.ID, s.SiteID, s.PublicationID, s.ScheduledAt, s.Action, s.Status,
		s.Recurrence, s.RecurrenceEnd, s.NotifyOnPublish, s.NotifyUsers, metadataJSON, s.CreatedBy, s.CreatedAt, s.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create schedule: %w", err)
	}
	return nil
}

func (r *Repository) GetSchedule(ctx context.Context, siteID, scheduleID uuid.UUID) (*Schedule, error) {
	if err := r.checkDB(); err != nil {
		return nil, err
	}

	var s Schedule
	var metadataStr string

	err := r.db.QueryRow(ctx,
		`SELECT id, site_id, publication_id, scheduled_at, action, status,
		        COALESCE(recurrence,''), recurrence_end, notify_on_publish, COALESCE(notify_users,'{}'),
		        COALESCE(metadata::text,'{}'), created_by, cancelled_at, COALESCE(cancel_reason,''), created_at, updated_at
		 FROM publication_schedule WHERE id = $1 AND site_id = $2`,
		scheduleID, siteID,
	).Scan(&s.ID, &s.SiteID, &s.PublicationID, &s.ScheduledAt, &s.Action, &s.Status,
		&s.Recurrence, &s.RecurrenceEnd, &s.NotifyOnPublish, &s.NotifyUsers,
		&metadataStr, &s.CreatedBy, &s.CancelledAt, &s.CancelReason, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrScheduleNotFound
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	if len(metadataStr) > 0 {
		_ = json.Unmarshal([]byte(metadataStr), &s.Metadata)
	}

	return &s, nil
}

func (r *Repository) ListSchedules(ctx context.Context, siteID uuid.UUID, status string, limit, offset int) ([]Schedule, error) {
	if err := r.checkDB(); err != nil {
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
		`SELECT id, site_id, publication_id, scheduled_at, action, status,
		        COALESCE(recurrence,''), recurrence_end, notify_on_publish, COALESCE(notify_users,'{}'),
		        COALESCE(metadata::text,'{}'), created_by, cancelled_at, COALESCE(cancel_reason,''), created_at, updated_at
		 FROM publication_schedule WHERE %s ORDER BY scheduled_at ASC LIMIT $%d OFFSET $%d`,
		strings.Join(where, " AND "), argIdx, argIdx+1,
	)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	defer rows.Close()

	var schedules []Schedule
	for rows.Next() {
		var s Schedule
		var metadataStr string
		if err := rows.Scan(&s.ID, &s.SiteID, &s.PublicationID, &s.ScheduledAt, &s.Action, &s.Status,
			&s.Recurrence, &s.RecurrenceEnd, &s.NotifyOnPublish, &s.NotifyUsers,
			&metadataStr, &s.CreatedBy, &s.CancelledAt, &s.CancelReason, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan schedule: %w", err)
		}
		if len(metadataStr) > 0 {
			_ = json.Unmarshal([]byte(metadataStr), &s.Metadata)
		}
		schedules = append(schedules, s)
	}
	if schedules == nil {
		schedules = []Schedule{}
	}
	return schedules, nil
}

func (r *Repository) UpdateSchedule(ctx context.Context, scheduleID uuid.UUID, updates map[string]interface{}) error {
	if err := r.checkDB(); err != nil {
		return err
	}

	if len(updates) == 0 {
		return nil
	}

	setClauses := []string{}
	args := []interface{}{}
	argIdx := 1

	for k, v := range updates {
		setClauses = append(setClauses, fmt.Sprintf("%s = $%d", k, argIdx))
		args = append(args, v)
		argIdx++
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	query := fmt.Sprintf(
		`UPDATE publication_schedule SET %s WHERE id = $%d`,
		strings.Join(setClauses, ", "), argIdx,
	)
	args = append(args, scheduleID)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}
	return nil
}

// --- Metrics ---

func (r *Repository) GetMetrics(ctx context.Context, siteID, pubID uuid.UUID) (*PublicationMetrics, error) {
	if err := r.checkDB(); err != nil {
		return nil, err
	}

	var m PublicationMetrics
	var metadataStr string

	err := r.db.QueryRow(ctx,
		`SELECT id, site_id, publication_id, COALESCE(view_count,0), COALESCE(unique_visitors,0),
		        COALESCE(avg_time_seconds,0), COALESCE(bounce_rate,0), COALESCE(share_count,0),
		        COALESCE(comment_count,0), COALESCE(like_count,0), COALESCE(click_count,0),
		        COALESCE(ctr,0), COALESCE(scroll_depth,0), COALESCE(metadata::text,'{}'),
		        recorded_at, created_at, updated_at
		 FROM publication_metrics WHERE site_id = $1 AND publication_id = $2
		 ORDER BY recorded_at DESC LIMIT 1`,
		siteID, pubID,
	).Scan(&m.ID, &m.SiteID, &m.PublicationID, &m.ViewCount, &m.UniqueVisitors,
		&m.AvgTimeSeconds, &m.BounceRate, &m.ShareCount, &m.CommentCount, &m.LikeCount,
		&m.ClickCount, &m.CTR, &m.ScrollDepth, &metadataStr,
		&m.RecordedAt, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrMetricsNotFound
		}
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	if len(metadataStr) > 0 {
		_ = json.Unmarshal([]byte(metadataStr), &m.Metadata)
	}

	return &m, nil
}

func (r *Repository) UpsertMetrics(ctx context.Context, m *PublicationMetrics) error {
	if err := r.checkDB(); err != nil {
		return err
	}

	metadataJSON := "{}"
	if m.Metadata != nil {
		b, _ := json.Marshal(m.Metadata)
		metadataJSON = string(b)
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO publication_metrics (id, site_id, publication_id, view_count, unique_visitors,
		 avg_time_seconds, bounce_rate, share_count, comment_count, like_count, click_count,
		 ctr, scroll_depth, metadata, recorded_at, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14::jsonb,$15,$15,$15)
		 ON CONFLICT (publication_id, site_id) DO UPDATE SET
		 view_count = EXCLUDED.view_count, unique_visitors = EXCLUDED.unique_visitors,
		 avg_time_seconds = EXCLUDED.avg_time_seconds, bounce_rate = EXCLUDED.bounce_rate,
		 share_count = EXCLUDED.share_count, comment_count = EXCLUDED.comment_count,
		 like_count = EXCLUDED.like_count, click_count = EXCLUDED.click_count,
		 ctr = EXCLUDED.ctr, scroll_depth = EXCLUDED.scroll_depth,
		 metadata = EXCLUDED.metadata, recorded_at = EXCLUDED.recorded_at, updated_at = NOW()`,
		m.ID, m.SiteID, m.PublicationID, m.ViewCount, m.UniqueVisitors,
		m.AvgTimeSeconds, m.BounceRate, m.ShareCount, m.CommentCount, m.LikeCount, m.ClickCount,
		m.CTR, m.ScrollDepth, metadataJSON, m.RecordedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert metrics: %w", err)
	}
	return nil
}

func (r *Repository) GetMetricsSummary(ctx context.Context, siteID uuid.UUID) (*PublicationMetricsSummary, error) {
	if err := r.checkDB(); err != nil {
		return nil, err
	}

	var s PublicationMetricsSummary

	err := r.db.QueryRow(ctx, `
		SELECT
			COALESCE(COUNT(*),0),
			COALESCE(SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN status = 'scheduled' THEN 1 ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN status = 'draft' THEN 1 ELSE 0 END),0),
			COALESCE(SUM(CASE WHEN status = 'archived' THEN 1 ELSE 0 END),0),
			COALESCE((SELECT SUM(view_count) FROM publication_metrics WHERE site_id = $1),0),
			COALESCE((SELECT AVG(view_count) FROM publication_metrics WHERE site_id = $1),0),
			COALESCE((SELECT SUM(share_count) FROM publication_metrics WHERE site_id = $1),0),
			COALESCE((SELECT SUM(comment_count) FROM publication_metrics WHERE site_id = $1),0),
			COALESCE((SELECT COUNT(*) FROM publication_queue WHERE site_id = $1 AND status = 'pending'),0),
			COALESCE((SELECT COUNT(*) FROM publication_schedule WHERE site_id = $1 AND status = 'scheduled'),0)
		FROM publications WHERE site_id = $1 AND status != 'deleted'`,
		siteID,
	).Scan(&s.TotalPublications, &s.PublishedCount, &s.ScheduledCount, &s.DraftCount,
		&s.ArchivedCount, &s.TotalViews, &s.AvgViews, &s.TotalShares, &s.TotalComments,
		&s.QueueSize, &s.PendingSchedules)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics summary: %w", err)
	}

	return &s, nil
}


