package media

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

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

func (r *Repository) Create(ctx context.Context, m *Media) error {
	metadataJSON := "{}"
	if m.Metadata != nil {
		b, _ := json.Marshal(m.Metadata)
		metadataJSON = string(b)
	}

	var folderID interface{}
	if m.FolderID != nil {
		folderID = *m.FolderID
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO media (id, site_id, folder_id, filename, original_name, mime_type, extension, size,
		 width, height, duration, hash, alt_text, caption, storage_provider, storage_key, metadata, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17::jsonb,$18,$19,$20)`,
		m.ID, m.SiteID, folderID, m.Filename, m.OriginalName, m.MimeType, m.Extension,
		m.Size, m.Width, m.Height, m.Duration, m.Hash, m.AltText, m.Caption,
		m.StorageProvider, m.StorageKey, metadataJSON, m.CreatedBy, m.CreatedAt, m.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create media: %w", err)
	}

	return nil
}

func (r *Repository) CreateVariant(ctx context.Context, v *MediaVariant) error {
	metadataJSON := "{}"
	if v.Metadata != nil {
		b, _ := json.Marshal(v.Metadata)
		metadataJSON = string(b)
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO media_variants (id, media_id, variant, width, height, file_size, mime_type, storage_key, metadata, created_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10)
		 ON CONFLICT (media_id, variant, mime_type) DO UPDATE SET
		 width = $4, height = $5, file_size = $6, storage_key = $8, metadata = $9::jsonb`,
		v.ID, v.MediaID, string(v.Variant), v.Width, v.Height, v.FileSize, v.MimeType,
		v.StorageKey, metadataJSON, v.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create media variant: %w", err)
	}

	return nil
}

func (r *Repository) GetByID(ctx context.Context, siteID, mediaID uuid.UUID) (*Media, error) {
	var m Media
	var folderID *uuid.UUID
	var width, height *int
	var deletedAt *time.Time
	var metadataJSON []byte
	var altText, caption string

	err := r.db.QueryRow(ctx,
		`SELECT id, site_id, folder_id, filename, original_name, mime_type, extension, size,
		 width, height, duration, hash, COALESCE(alt_text,''), COALESCE(caption,''),
		 storage_provider, storage_key, COALESCE(metadata::text,'{}'), created_by, created_at, updated_at, deleted_at
		 FROM media WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		mediaID, siteID,
	).Scan(
		&m.ID, &m.SiteID, &folderID, &m.Filename, &m.OriginalName, &m.MimeType, &m.Extension, &m.Size,
		&width, &height, &m.Duration, &m.Hash, &altText, &caption,
		&m.StorageProvider, &m.StorageKey, &metadataJSON, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrMediaNotFound
		}
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	m.FolderID = folderID
	m.Width = width
	m.Height = height
	m.AltText = altText
	m.Caption = caption
	m.DeletedAt = deletedAt

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &m.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal media metadata: %w", err)
		}
	}
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}

	return &m, nil
}

func (r *Repository) GetVariants(ctx context.Context, mediaID uuid.UUID) ([]MediaVariant, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, media_id, variant, width, height, file_size, mime_type, storage_key,
		 COALESCE(metadata::text,'{}'), created_at
		 FROM media_variants WHERE media_id = $1 ORDER BY created_at ASC`,
		mediaID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get variants: %w", err)
	}
	defer rows.Close()

	var variants []MediaVariant
	for rows.Next() {
		var v MediaVariant
		var metadataJSON []byte
		var variant string

		err := rows.Scan(
			&v.ID, &v.MediaID, &variant, &v.Width, &v.Height, &v.FileSize, &v.MimeType,
			&v.StorageKey, &metadataJSON, &v.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan variant: %w", err)
		}

		v.Variant = VariantType(variant)

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &v.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal variant metadata: %w", err)
			}
		}
		if v.Metadata == nil {
			v.Metadata = make(map[string]interface{})
		}

		variants = append(variants, v)
	}

	if variants == nil {
		variants = []MediaVariant{}
	}

	return variants, nil
}

func (r *Repository) List(ctx context.Context, req MediaListRequest) (*MediaListResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	allowedSorts := map[string]bool{"created_at": true, "updated_at": true, "original_name": true, "size": true, "mime_type": true, "extension": true}
	sortColumn := "m.created_at"
	if req.Sort != "" {
		if req.Sort == "name" {
			req.Sort = "original_name"
		}
		if allowedSorts[req.Sort] {
			sortColumn = "m." + req.Sort
		}
	}

	orderDir := "DESC"
	if strings.EqualFold(req.Order, "asc") {
		orderDir = "ASC"
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, "m.deleted_at IS NULL")
	whereClauses = append(whereClauses, fmt.Sprintf("m.site_id = $%d", argIdx))
	args = append(args, req.SiteID)
	argIdx++

	if req.FolderID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("m.folder_id = $%d", argIdx))
		args = append(args, *req.FolderID)
	} else {
		whereClauses = append(whereClauses, "m.folder_id IS NULL")
	}
	argIdx++

	if req.Extension != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("m.extension = $%d", argIdx))
		args = append(args, req.Extension)
		argIdx++
	}

	if req.MimeType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("m.mime_type LIKE $%d", argIdx))
		args = append(args, req.MimeType+"/%")
		argIdx++
	}

	if req.Type != "" {
		typePrefix := string(req.Type)
		whereClauses = append(whereClauses, fmt.Sprintf("m.mime_type LIKE $%d", argIdx))
		args = append(args, typePrefix+"/%")
		argIdx++
	}

	if req.Search != "" {
		whereClauses = append(whereClauses,
			fmt.Sprintf("(m.original_name ILIKE $%d OR m.alt_text ILIKE $%d OR m.caption ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+req.Search+"%")
		argIdx++
	}

	if req.MinSize != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("m.size >= $%d", argIdx))
		args = append(args, *req.MinSize)
		argIdx++
	}

	if req.MaxSize != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("m.size <= $%d", argIdx))
		args = append(args, *req.MaxSize)
		argIdx++
	}

	if req.FromDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("m.created_at >= $%d", argIdx))
		args = append(args, *req.FromDate)
		argIdx++
	}

	if req.ToDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("m.created_at <= $%d", argIdx))
		args = append(args, *req.ToDate)
		argIdx++
	}

	if req.CreatedBy != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("m.created_by = $%d", argIdx))
		args = append(args, *req.CreatedBy)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM media m WHERE %s`, whereSQL)
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count media: %w", err)
	}

	offset := (req.Page - 1) * req.PerPage

	query := fmt.Sprintf(
		`SELECT m.id, m.site_id, m.folder_id, m.filename, m.original_name, m.mime_type, m.extension, m.size,
		 m.width, m.height, m.duration, m.hash, COALESCE(m.alt_text,''), COALESCE(m.caption,''),
		 m.storage_provider, m.storage_key, COALESCE(m.metadata::text,'{}'), m.created_by, m.created_at, m.updated_at, m.deleted_at
		 FROM media m
		 WHERE %s
		 ORDER BY %s %s
		 LIMIT $%d OFFSET $%d`,
		whereSQL, sortColumn, orderDir, argIdx, argIdx+1,
	)
	args = append(args, req.PerPage, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list media: %w", err)
	}
	defer rows.Close()

	var mediaList []Media
	for rows.Next() {
		var m Media
		var folderID *uuid.UUID
		var width, height *int
		var deletedAt *time.Time
		var metadataJSON []byte
		var altText, caption string

		err := rows.Scan(
			&m.ID, &m.SiteID, &folderID, &m.Filename, &m.OriginalName, &m.MimeType, &m.Extension, &m.Size,
			&width, &height, &m.Duration, &m.Hash, &altText, &caption,
			&m.StorageProvider, &m.StorageKey, &metadataJSON, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media: %w", err)
		}

		m.FolderID = folderID
		m.Width = width
		m.Height = height
		m.AltText = altText
		m.Caption = caption
		m.DeletedAt = deletedAt

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &m.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal media metadata: %w", err)
			}
		}
		if m.Metadata == nil {
			m.Metadata = make(map[string]interface{})
		}

		mediaList = append(mediaList, m)
	}

	if mediaList == nil {
		mediaList = []Media{}
	}

	return &MediaListResponse{
		Media:   mediaList,
		Total:   total,
		Page:    req.Page,
		PerPage: req.PerPage,
	}, nil
}

func (r *Repository) Update(ctx context.Context, siteID, mediaID uuid.UUID, req UpdateMediaRequest) (*Media, error) {
	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.FolderID != nil {
		setClauses = append(setClauses, fmt.Sprintf("folder_id = $%d", argIdx))
		args = append(args, *req.FolderID)
		argIdx++
	}
	if req.AltText != nil {
		setClauses = append(setClauses, fmt.Sprintf("alt_text = $%d", argIdx))
		args = append(args, *req.AltText)
		argIdx++
	}
	if req.Caption != nil {
		setClauses = append(setClauses, fmt.Sprintf("caption = $%d", argIdx))
		args = append(args, *req.Caption)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, siteID, mediaID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	updateQuery := fmt.Sprintf(
		`UPDATE media SET %s WHERE id = $%d AND site_id = $%d AND deleted_at IS NULL`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, mediaID, siteID)

	_, err := r.db.Exec(ctx, updateQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update media: %w", err)
	}

	return r.GetByID(ctx, siteID, mediaID)
}

func (r *Repository) SoftDelete(ctx context.Context, siteID, mediaID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE media SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		mediaID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete media: %w", err)
	}
	return nil
}

func (r *Repository) Restore(ctx context.Context, siteID, mediaID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE media SET deleted_at = NULL, updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NOT NULL`,
		mediaID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to restore media: %w", err)
	}
	return nil
}

func (r *Repository) PermanentlyDelete(ctx context.Context, mediaID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`DELETE FROM media WHERE id = $1`,
		mediaID,
	)
	if err != nil {
		return fmt.Errorf("failed to permanently delete media: %w", err)
	}
	return nil
}

func (r *Repository) Move(ctx context.Context, siteID uuid.UUID, mediaIDs []uuid.UUID, folderID *uuid.UUID) error {
	argIdx := 1
	args := []interface{}{siteID}
	placeholders := make([]string, len(mediaIDs))
	for i, id := range mediaIDs {
		argIdx++
		placeholders[i] = fmt.Sprintf("$%d", argIdx)
		args = append(args, id)
	}

	var folderClause string
	if folderID != nil {
		argIdx++
		folderClause = fmt.Sprintf("folder_id = $%d", argIdx)
		args = append(args, *folderID)
	} else {
		folderClause = "folder_id = NULL"
	}

	query := fmt.Sprintf(
		`UPDATE media SET %s, updated_at = NOW() WHERE site_id = $1 AND id IN (%s) AND deleted_at IS NULL`,
		folderClause, strings.Join(placeholders, ","),
	)

	_, err := r.db.Exec(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("failed to move media: %w", err)
	}
	return nil
}

func (r *Repository) GetTotalSize(ctx context.Context, siteID uuid.UUID) (int64, error) {
	var total int64
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(size), 0) FROM media WHERE site_id = $1 AND deleted_at IS NULL`,
		siteID,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("failed to get total size: %w", err)
	}
	return total, nil
}

func (r *Repository) FindByHash(ctx context.Context, siteID uuid.UUID, hash string) (*Media, error) {
	var m Media
	var folderID *uuid.UUID
	var width, height *int
	var deletedAt *time.Time
	var metadataJSON []byte
	var altText, caption string

	err := r.db.QueryRow(ctx,
		`SELECT id, site_id, folder_id, filename, original_name, mime_type, extension, size,
		 width, height, duration, hash, COALESCE(alt_text,''), COALESCE(caption,''),
		 storage_provider, storage_key, COALESCE(metadata::text,'{}'), created_by, created_at, updated_at, deleted_at
		 FROM media WHERE site_id = $1 AND hash = $2 AND deleted_at IS NULL LIMIT 1`,
		siteID, hash,
	).Scan(
		&m.ID, &m.SiteID, &folderID, &m.Filename, &m.OriginalName, &m.MimeType, &m.Extension, &m.Size,
		&width, &height, &m.Duration, &m.Hash, &altText, &caption,
		&m.StorageProvider, &m.StorageKey, &metadataJSON, &m.CreatedBy, &m.CreatedAt, &m.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find media by hash: %w", err)
	}

	m.FolderID = folderID
	m.Width = width
	m.Height = height
	m.AltText = altText
	m.Caption = caption
	m.DeletedAt = deletedAt

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &m.Metadata); err != nil {
			return nil, fmt.Errorf("failed to unmarshal media metadata: %w", err)
		}
	}
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}

	return &m, nil
}

func (r *Repository) CreateFolder(ctx context.Context, f *Folder) error {
	var parentID interface{}
	if f.ParentID != nil {
		parentID = *f.ParentID
	}

	_, err := r.db.Exec(ctx,
		`INSERT INTO folders (id, site_id, parent_id, name, slug, description, sort_order, created_by, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		f.ID, f.SiteID, parentID, f.Name, f.Slug, f.Description, f.SortOrder, f.CreatedBy, f.CreatedAt, f.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create folder: %w", err)
	}
	return nil
}

func (r *Repository) GetFolderByID(ctx context.Context, siteID, folderID uuid.UUID) (*Folder, error) {
	var f Folder
	var parentID *uuid.UUID
	var deletedAt *time.Time

	err := r.db.QueryRow(ctx,
		`SELECT id, site_id, parent_id, name, slug, COALESCE(description,''), sort_order, created_by, created_at, updated_at, deleted_at
		 FROM folders WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		folderID, siteID,
	).Scan(
		&f.ID, &f.SiteID, &parentID, &f.Name, &f.Slug, &f.Description, &f.SortOrder,
		&f.CreatedBy, &f.CreatedAt, &f.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrFolderNotFound
		}
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	f.ParentID = parentID
	f.DeletedAt = deletedAt

	return &f, nil
}

func (r *Repository) ListFolders(ctx context.Context, siteID uuid.UUID) ([]Folder, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, site_id, parent_id, name, slug, COALESCE(description,''), sort_order, created_by, created_at, updated_at, deleted_at
		 FROM folders WHERE site_id = $1 AND deleted_at IS NULL
		 ORDER BY sort_order ASC, name ASC`,
		siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}
	defer rows.Close()

	var folders []Folder
	for rows.Next() {
		var f Folder
		var parentID *uuid.UUID
		var deletedAt *time.Time

		err := rows.Scan(
			&f.ID, &f.SiteID, &parentID, &f.Name, &f.Slug, &f.Description, &f.SortOrder,
			&f.CreatedBy, &f.CreatedAt, &f.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan folder: %w", err)
		}

		f.ParentID = parentID
		f.DeletedAt = deletedAt

		folders = append(folders, f)
	}

	if folders == nil {
		folders = []Folder{}
	}

	return folders, nil
}

func (r *Repository) UpdateFolder(ctx context.Context, siteID, folderID uuid.UUID, req UpdateFolderRequest) (*Folder, error) {
	var setClauses []string
	var args []interface{}
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
	if req.ParentID != nil {
		setClauses = append(setClauses, fmt.Sprintf("parent_id = $%d", argIdx))
		args = append(args, *req.ParentID)
		argIdx++
	}

	if len(setClauses) == 0 {
		return r.GetFolderByID(ctx, siteID, folderID)
	}

	setClauses = append(setClauses, "updated_at = NOW()")
	updateQuery := fmt.Sprintf(
		`UPDATE folders SET %s WHERE id = $%d AND site_id = $%d AND deleted_at IS NULL`,
		strings.Join(setClauses, ", "), argIdx, argIdx+1,
	)
	args = append(args, folderID, siteID)

	_, err := r.db.Exec(ctx, updateQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to update folder: %w", err)
	}

	return r.GetFolderByID(ctx, siteID, folderID)
}

func (r *Repository) DeleteFolder(ctx context.Context, siteID, folderID uuid.UUID) error {
	_, err := r.db.Exec(ctx,
		`UPDATE folders SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		folderID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete folder: %w", err)
	}

	_, _ = r.db.Exec(ctx,
		`UPDATE media SET folder_id = NULL WHERE folder_id = $1 AND site_id = $2`,
		folderID, siteID,
	)

	return nil
}

func (r *Repository) GetFolderChildCount(ctx context.Context, folderID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM media WHERE folder_id = $1 AND deleted_at IS NULL`,
		folderID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count folder children: %w", err)
	}
	return count, nil
}

func (r *Repository) GetFolderSubfolderCount(ctx context.Context, folderID uuid.UUID) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM folders WHERE parent_id = $1 AND deleted_at IS NULL`,
		folderID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count subfolders: %w", err)
	}
	return count, nil
}
