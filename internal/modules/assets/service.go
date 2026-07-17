package assets

import (
	"context"
	"encoding/json"
	"fmt"
	goimage "image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"nexora/internal/kernel"
	"nexora/internal/pkg/audit"
	"nexora/internal/pkg/cache"
	"nexora/internal/pkg/config"
	"nexora/internal/pkg/database"
	imgpkg "nexora/internal/pkg/image"
	"nexora/internal/pkg/logger"
	"nexora/internal/pkg/storage"
)

type Service struct {
	log         *logger.Logger
	db          *database.Database
	cache       *cache.Cache
	eventBus    *kernel.EventBus
	auditLog    *audit.Logger
	storage     storage.Driver
	validateCfg FileValidationConfig
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache, st storage.Driver) *Service {
	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}
	if ch == nil {
		ch = cache.New(true)
	}
	return &Service{
		log:         log,
		db:          db,
		cache:       ch,
		auditLog:    audit.New(pool, log),
		storage:     st,
		validateCfg: DefaultFileValidationConfig(),
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

func (s *Service) Upload(ctx context.Context, siteID, userID uuid.UUID, file multipart.File, header *multipart.FileHeader, req UploadRequest) (*Asset, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	ext := strings.ToLower(strings.TrimPrefix(path.Ext(header.Filename), "."))
	if ext == "" {
		return nil, ErrInvalidFileType
	}

	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		return nil, ErrInvalidFile
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, fmt.Errorf("failed to seek file: %w", err)
	}

	mimeType := http.DetectContentType(buf)
	if !s.isAllowedMimeType(mimeType, ext) {
		return nil, ErrInvalidFileType
	}

	assetType := s.classifyMIMEType(mimeType)
	sizeLimit := s.getSizeLimit(assetType)
	if header.Size > sizeLimit {
		return nil, ErrFileTooLarge
	}

	now := time.Now()
	assetID := uuid.New()
	storageFilename := fmt.Sprintf("%s.%s", assetID.String(), ext)
	storagePath := filepath.Join(siteID.String(), s.classifyMIMEType(mimeType), storageFilename)

	if err := s.storage.Upload(ctx, storagePath, file); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	var asset Asset
	asset.ID = assetID
	asset.SiteID = siteID
	asset.UserID = userID
	asset.Filename = storageFilename
	asset.OriginalName = header.Filename
	asset.MimeType = mimeType
	asset.Extension = ext
	asset.Size = header.Size
	asset.StorageProvider = "local"
	asset.StoragePath = storagePath
	asset.URL = s.storage.URL(storagePath)
	asset.AltText = req.AltText
	asset.Title = req.Title
	asset.Caption = req.Caption
	asset.Description = req.Description
	asset.Metadata = make(map[string]interface{})

	if strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml" {
		localDriver, ok := s.storage.(*storage.LocalDriver)
		if !ok {
			return nil, fmt.Errorf("storage driver is not local")
		}
		fullPath := filepath.Join(localDriver.BasePath, storagePath)
		imgInfo, err := s.processImage(ctx, fullPath, siteID, assetID, ext)
		if err != nil {
			s.log.Warn("failed to process image", "error", err, "asset_id", assetID)
		} else {
			asset.Width = &imgInfo.Width
			asset.Height = &imgInfo.Height
			asset.ThumbnailPath = fmt.Sprintf("%s/thumbnails/%s.%s", siteID.String(), assetID.String(), ext)
			asset.OptimizedPath = fmt.Sprintf("%s/optimized/%s.%s", siteID.String(), assetID.String(), ext)
		}
	}

	metadataJSON := "{}"
	if asset.Metadata != nil {
		b, _ := json.Marshal(asset.Metadata)
		metadataJSON = string(b)
	}

	_, err = p.Exec(ctx,
		`INSERT INTO assets (id, site_id, user_id, filename, original_name, mime_type, extension, size,
		 width, height, alt_text, title, caption, description, thumbnail_path, optimized_path,
		 storage_provider, storage_path, url, metadata, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20::jsonb,$21,$22)`,
		asset.ID, asset.SiteID, asset.UserID, asset.Filename, asset.OriginalName, asset.MimeType, asset.Extension,
		asset.Size, asset.Width, asset.Height, asset.AltText, asset.Title, asset.Caption, asset.Description,
		asset.ThumbnailPath, asset.OptimizedPath, asset.StorageProvider, asset.StoragePath, asset.URL,
		metadataJSON, now, now,
	)
	if err != nil {
		if delErr := s.storage.Delete(ctx, storagePath); delErr != nil {
			s.log.Warn("failed to clean up uploaded file", "error", delErr, "path", storagePath)
		}
		return nil, fmt.Errorf("failed to create asset record: %w", err)
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("asset.created"),
		EntityType: "asset",
		EntityID:   &assetID,
		Payload:    map[string]interface{}{"original_name": header.Filename, "mime_type": mimeType, "size": header.Size},
	})

	s.fireEvent(ctx, EventAssetCreated, map[string]interface{}{
		"asset_id": assetID.String(),
		"site_id":  siteID.String(),
		"filename": header.Filename,
		"mime":     mimeType,
	}, siteID)

	return &asset, nil
}

func (s *Service) processImage(ctx context.Context, fullPath string, siteID, assetID uuid.UUID, ext string) (*imgpkg.ImageInfo, error) {
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer func() { _ = f.Close() }()

	src, _, err := goimage.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	localDriver, ok := s.storage.(*storage.LocalDriver)
	if !ok {
		return nil, fmt.Errorf("storage driver is not local")
	}
	storageDir := localDriver.BasePath

	thumbPath := filepath.Join(storageDir, siteID.String(), "thumbnails", fmt.Sprintf("%s.%s", assetID.String(), ext))
	if err := imgpkg.GenerateVariant(fullPath, thumbPath, imgpkg.VariantThumbnail); err != nil {
		s.log.Warn("failed to generate thumbnail", "error", err, "asset_id", assetID)
	}

	optPath := filepath.Join(storageDir, siteID.String(), "optimized", fmt.Sprintf("%s.%s", assetID.String(), ext))
	if err := imgpkg.GenerateVariant(fullPath, optPath, imgpkg.VariantLarge); err != nil {
		s.log.Warn("failed to generate optimized", "error", err, "asset_id", assetID)
	}

	bounds := src.Bounds()
	return &imgpkg.ImageInfo{Width: bounds.Dx(), Height: bounds.Dy()}, nil
}

func (s *Service) GetByID(ctx context.Context, siteID, assetID uuid.UUID) (*Asset, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	var a Asset
	var altText, title, caption, description, thumbnailPath, optimizedPath, url string
	var width, height *int
	var deletedAt *time.Time
	var metadataJSON []byte

	err = p.QueryRow(ctx,
		`SELECT id, site_id, user_id, filename, original_name, mime_type, extension, size,
		 width, height, COALESCE(alt_text,''), COALESCE(title,''), COALESCE(caption,''), COALESCE(description,''),
		 COALESCE(thumbnail_path,''), COALESCE(optimized_path,''), storage_provider, storage_path, COALESCE(url,''),
		 COALESCE(metadata::text,'{}'), created_at, updated_at, deleted_at
		 FROM assets WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		assetID, siteID,
	).Scan(
		&a.ID, &a.SiteID, &a.UserID, &a.Filename, &a.OriginalName, &a.MimeType, &a.Extension, &a.Size,
		&width, &height, &altText, &title, &caption, &description,
		&thumbnailPath, &optimizedPath, &a.StorageProvider, &a.StoragePath, &url,
		&metadataJSON, &a.CreatedAt, &a.UpdatedAt, &deletedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrAssetNotFound
		}
		return nil, fmt.Errorf("failed to get asset: %w", err)
	}

	a.Width = width
	a.Height = height
	a.AltText = altText
	a.Title = title
	a.Caption = caption
	a.Description = description
	a.ThumbnailPath = thumbnailPath
	a.OptimizedPath = optimizedPath
	a.URL = url
	a.DeletedAt = deletedAt

	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &a.Metadata); err != nil {
			s.log.Warn("failed to unmarshal asset metadata", "error", err, "asset_id", a.ID)
		}
	}
	if a.Metadata == nil {
		a.Metadata = make(map[string]interface{})
	}

	return &a, nil
}

func (s *Service) List(ctx context.Context, req AssetListRequest) (*AssetListResponse, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PerPage < 1 || req.PerPage > 100 {
		req.PerPage = 20
	}

	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	allowedSorts := map[string]bool{"created_at": true, "updated_at": true, "original_name": true, "size": true, "mime_type": true}
	sortColumn := "created_at"
	if req.Sort != "" && allowedSorts[req.Sort] {
		sortColumn = req.Sort
	}

	orderDir := "DESC"
	if strings.EqualFold(req.Order, "asc") {
		orderDir = "ASC"
	}

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	whereClauses = append(whereClauses, "a.deleted_at IS NULL")
	whereClauses = append(whereClauses, fmt.Sprintf("a.site_id = $%d", argIdx))
	args = append(args, req.SiteID)
	argIdx++

	if req.Extension != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("a.extension = $%d", argIdx))
		args = append(args, req.Extension)
		argIdx++
	}

	if req.Type != "" {
		typePrefix := string(req.Type)
		if typePrefix == "image" || typePrefix == "video" || typePrefix == "audio" {
			whereClauses = append(whereClauses, fmt.Sprintf("a.mime_type LIKE $%d", argIdx))
			args = append(args, typePrefix+"/%")
			argIdx++
		} else if typePrefix == "document" {
			whereClauses = append(whereClauses, fmt.Sprintf("(a.mime_type LIKE 'application/%%' OR a.mime_type LIKE 'text/%%')"))
		}
	}

	if req.Search != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(a.original_name ILIKE $%d OR a.alt_text ILIKE $%d OR a.title ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+req.Search+"%")
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	var total int
	countQuery := fmt.Sprintf(`SELECT COUNT(*) FROM assets a WHERE %s`, whereSQL)
	err = p.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to count assets: %w", err)
	}

	offset := (req.Page - 1) * req.PerPage

	query := fmt.Sprintf(
		`SELECT a.id, a.site_id, a.user_id, a.filename, a.original_name, a.mime_type, a.extension, a.size,
		 a.width, a.height, COALESCE(a.alt_text,''), COALESCE(a.title,''), COALESCE(a.caption,''), COALESCE(a.description,''),
		 COALESCE(a.thumbnail_path,''), COALESCE(a.optimized_path,''), a.storage_provider, a.storage_path, COALESCE(a.url,''),
		 COALESCE(a.metadata::text,'{}'), a.created_at, a.updated_at, a.deleted_at
		 FROM assets a
		 WHERE %s
		 ORDER BY a.%s %s
		 LIMIT $%d OFFSET $%d`,
		whereSQL, sortColumn, orderDir, argIdx, argIdx+1,
	)
	args = append(args, req.PerPage, offset)

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list assets: %w", err)
	}
	defer rows.Close()

	var assets []Asset
	for rows.Next() {
		var a Asset
		var altText, title, caption, description, thumbnailPath, optimizedPath, url string
		var width, height *int
		var deletedAt *time.Time
		var metadataJSON []byte

		err := rows.Scan(
			&a.ID, &a.SiteID, &a.UserID, &a.Filename, &a.OriginalName, &a.MimeType, &a.Extension, &a.Size,
			&width, &height, &altText, &title, &caption, &description,
			&thumbnailPath, &optimizedPath, &a.StorageProvider, &a.StoragePath, &url,
			&metadataJSON, &a.CreatedAt, &a.UpdatedAt, &deletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan asset: %w", err)
		}

		a.Width = width
		a.Height = height
		a.AltText = altText
		a.Title = title
		a.Caption = caption
		a.Description = description
		a.ThumbnailPath = thumbnailPath
		a.OptimizedPath = optimizedPath
		a.URL = url
		a.DeletedAt = deletedAt

		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &a.Metadata); err != nil {
				s.log.Warn("failed to unmarshal asset metadata", "error", err, "asset_id", a.ID)
			}
		}
		if a.Metadata == nil {
			a.Metadata = make(map[string]interface{})
		}

		assets = append(assets, a)
	}

	if assets == nil {
		assets = []Asset{}
	}

	return &AssetListResponse{
		Assets:  assets,
		Total:   total,
		Page:    req.Page,
		PerPage: req.PerPage,
	}, nil
}

func (s *Service) Update(ctx context.Context, siteID, assetID uuid.UUID, req UpdateAssetRequest) (*Asset, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	existing, err := s.GetByID(ctx, siteID, assetID)
	if err != nil {
		return nil, err
	}

	var setClauses []string
	var args []interface{}
	argIdx := 1

	if req.AltText != nil {
		setClauses = append(setClauses, fmt.Sprintf("alt_text = $%d", argIdx))
		args = append(args, *req.AltText)
		argIdx++
	}
	if req.Title != nil {
		setClauses = append(setClauses, fmt.Sprintf("title = $%d", argIdx))
		args = append(args, *req.Title)
		argIdx++
	}
	if req.Caption != nil {
		setClauses = append(setClauses, fmt.Sprintf("caption = $%d", argIdx))
		args = append(args, *req.Caption)
		argIdx++
	}
	if req.Description != nil {
		setClauses = append(setClauses, fmt.Sprintf("description = $%d", argIdx))
		args = append(args, *req.Description)
		argIdx++
	}

	if len(setClauses) > 0 {
		setClauses = append(setClauses, "updated_at = NOW()")
		updateQuery := fmt.Sprintf(
			`UPDATE assets SET %s WHERE id = $%d AND site_id = $%d AND deleted_at IS NULL`,
			strings.Join(setClauses, ", "), argIdx, argIdx+1,
		)
		args = append(args, assetID, siteID)

		_, err = p.Exec(ctx, updateQuery, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to update asset: %w", err)
		}

		s.fireEvent(ctx, EventAssetUpdated, map[string]interface{}{
			"asset_id": assetID.String(),
			"site_id":  siteID.String(),
		}, siteID)
	}

	if req.AltText != nil {
		existing.AltText = *req.AltText
	}
	if req.Title != nil {
		existing.Title = *req.Title
	}
	if req.Caption != nil {
		existing.Caption = *req.Caption
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}

	return existing, nil
}

func (s *Service) Delete(ctx context.Context, siteID, assetID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	asset, err := s.GetByID(ctx, siteID, assetID)
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`UPDATE assets SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1 AND site_id = $2 AND deleted_at IS NULL`,
		assetID, siteID,
	)
	if err != nil {
		return fmt.Errorf("failed to soft delete asset: %w", err)
	}

	if asset.ThumbnailPath != "" {
		if err := s.storage.Delete(ctx, asset.ThumbnailPath); err != nil {
			s.log.Warn("failed to delete thumbnail", "error", err, "path", asset.ThumbnailPath)
		}
	}
	if asset.OptimizedPath != "" {
		if err := s.storage.Delete(ctx, asset.OptimizedPath); err != nil {
			s.log.Warn("failed to delete optimized", "error", err, "path", asset.OptimizedPath)
		}
	}
	if err := s.storage.Delete(ctx, asset.StoragePath); err != nil {
		s.log.Warn("failed to delete storage file", "error", err, "path", asset.StoragePath)
	}

	s.auditLog.Log(ctx, audit.Entry{
		SiteID:     &siteID,
		EntityType: "asset",
		EntityID:   &assetID,
		Action:     audit.Action("asset.deleted"),
		Payload:    map[string]interface{}{"original_name": asset.OriginalName},
	})

	s.fireEvent(ctx, EventAssetDeleted, map[string]interface{}{
		"asset_id": assetID.String(),
		"site_id":  siteID.String(),
	}, siteID)

	return nil
}

func (s *Service) LinkToPost(ctx context.Context, siteID uuid.UUID, req LinkAssetRequest) (*PostAsset, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	asset, err := s.GetByID(ctx, siteID, req.AssetID)
	if err != nil {
		return nil, err
	}
	_ = asset

	postAssetID := uuid.New()
	_, err = p.Exec(ctx,
		`INSERT INTO post_assets (id, post_id, asset_id, sort_order, type) VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (post_id, asset_id) DO UPDATE SET sort_order = $4, type = $5`,
		postAssetID, req.PostID, req.AssetID, req.SortOrder, string(req.Type),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to link asset to post: %w", err)
	}

	pa := &PostAsset{
		ID:        postAssetID,
		PostID:    req.PostID,
		AssetID:   req.AssetID,
		SortOrder: req.SortOrder,
		Type:      req.Type,
		CreatedAt: time.Now(),
	}

	return pa, nil
}

func (s *Service) UnlinkFromPost(ctx context.Context, siteID, postID, assetID uuid.UUID) error {
	p, err := s.pool()
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx,
		`DELETE FROM post_assets WHERE post_id = $1 AND asset_id = $2`,
		postID, assetID,
	)
	if err != nil {
		return fmt.Errorf("failed to unlink asset from post: %w", err)
	}

	return nil
}

func (s *Service) GetPostAssets(ctx context.Context, siteID, postID uuid.UUID) ([]PostAsset, error) {
	p, err := s.pool()
	if err != nil {
		return nil, err
	}

	rows, err := p.Query(ctx,
		`SELECT pa.id, pa.post_id, pa.asset_id, pa.sort_order, pa.type, pa.created_at
		 FROM post_assets pa
		 INNER JOIN assets a ON a.id = pa.asset_id
		 WHERE pa.post_id = $1 AND a.site_id = $2 AND a.deleted_at IS NULL
		 ORDER BY pa.sort_order ASC`,
		postID, siteID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get post assets: %w", err)
	}
	defer rows.Close()

	var postAssets []PostAsset
	for rows.Next() {
		var pa PostAsset
		err := rows.Scan(&pa.ID, &pa.PostID, &pa.AssetID, &pa.SortOrder, &pa.Type, &pa.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan post asset: %w", err)
		}
		postAssets = append(postAssets, pa)
	}

	if postAssets == nil {
		postAssets = []PostAsset{}
	}

	return postAssets, nil
}

func (s *Service) isAllowedMimeType(mimeType, ext string) bool {
	expectedMime, ok := ExtensionMIMEMap[ext]
	if !ok {
		return false
	}

	if mimeType != expectedMime {
		return false
	}

	assetType := s.classifyMIMEType(mimeType)
	allowed, ok := s.validateCfg.AllowedMIMETypes[assetType]
	if !ok {
		return false
	}

	for _, a := range allowed {
		if a == mimeType {
			return true
		}
	}

	return false
}

func (s *Service) classifyMIMEType(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "image"
	}
	if strings.HasPrefix(mimeType, "video/") {
		return "video"
	}
	if strings.HasPrefix(mimeType, "audio/") {
		return "audio"
	}
	return "document"
}

func (s *Service) getSizeLimit(assetType string) int64 {
	switch assetType {
	case "image":
		return s.validateCfg.MaxImageSize
	case "video":
		return s.validateCfg.MaxVideoSize
	case "audio":
		return s.validateCfg.MaxAudioSize
	default:
		return s.validateCfg.MaxDocumentSize
	}
}


