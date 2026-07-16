package media

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

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
	log        *logger.Logger
	repo       *Repository
	cache      *cache.Cache
	eventBus   *kernel.EventBus
	auditLog   *audit.Logger
	storage    storage.Driver
	validator  *Validator
	cfg        FileValidationConfig
	basePath   string
	baseURL    string
	driverName string
}

func NewService(cfg *config.Config, log *logger.Logger, db *database.Database, ch *cache.Cache, st storage.Driver) *Service {
	var pool database.Pool
	if db != nil {
		pool = db.Pool
	}

	validCfg := DefaultFileValidationConfig()
	driverName := storage.DriverLocal
	if cfg != nil {
		driverName = cfg.Storage.Driver
	}

	if ch == nil {
		ch = cache.New(true)
	}

	return &Service{
		log:        log,
		repo:       NewRepository(pool),
		cache:      ch,
		auditLog:   audit.New(pool, log),
		storage:    st,
		validator:  NewValidator(validCfg),
		cfg:        validCfg,
		basePath:   "/tmp/nexora-media",
		baseURL:    "/uploads",
		driverName: driverName,
	}
}

func (s *Service) SetEventBus(bus *kernel.EventBus) {
	s.eventBus = bus
}

func (s *Service) fireEvent(ctx context.Context, eventType kernel.EventType, payload interface{}) {
	if s.eventBus != nil {
		s.eventBus.EmitAsync(ctx, eventType, payload, "")
	}
}

func (s *Service) checkDB() error {
	if s.repo == nil {
		return ErrDatabaseNotAvail
	}
	return s.repo.checkDB()
}

func (s *Service) Upload(ctx context.Context, siteID, userID uuid.UUID, file multipart.File, header *multipart.FileHeader, req UploadRequest) (*Media, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	mimeType, ext, err := s.validator.ValidateFile(header, file)
	if err != nil {
		return nil, err
	}

	hash, err := ComputeHash(file)
	if err != nil {
		return nil, fmt.Errorf("failed to compute hash: %w", err)
	}

	existing, err := s.repo.FindByHash(ctx, siteID, hash)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	storageKey := fmt.Sprintf("%s/%s/%s.%s", siteID.String(), ClassifyMIMEType(mimeType), uuid.New().String(), ext)

	if err := s.storage.Upload(ctx, storageKey, file); err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	now := time.Now()
	mediaID := uuid.New()

	m := &Media{
		ID:              mediaID,
		SiteID:          siteID,
		FolderID:        req.FolderID,
		Filename:        path.Base(storageKey),
		OriginalName:    header.Filename,
		MimeType:        mimeType,
		Extension:       ext,
		Size:            header.Size,
		Hash:            hash,
		AltText:         req.AltText,
		Caption:         req.Caption,
		StorageProvider: s.driverName,
		StorageKey:      storageKey,
		Metadata:        make(map[string]interface{}),
		CreatedBy:       userID,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml" {
		mediaPath := filepath.Join(s.basePath, storageKey)
		imgInfo, err := s.processImage(ctx, mediaPath, siteID, mediaID, ext)
		if err != nil {
			s.log.Warn("failed to process image", "error", err, "media_id", mediaID)
		} else {
			m.Width = &imgInfo.Width
			m.Height = &imgInfo.Height
		}
	}

	if err := s.repo.Create(ctx, m); err != nil {
		if delErr := s.storage.Delete(ctx, storageKey); delErr != nil {
			s.log.Warn("failed to clean up uploaded file", "error", delErr, "key", storageKey)
		}
		return nil, err
	}

	s.auditLog.Log(ctx, audit.Entry{
		UserID:     &userID,
		SiteID:     &siteID,
		Action:     audit.Action("media.uploaded"),
		EntityType: "media",
		EntityID:   &mediaID,
		Payload:    map[string]interface{}{"original_name": header.Filename, "mime_type": mimeType, "size": header.Size},
	})

	s.fireEvent(ctx, EventMediaUploaded, map[string]interface{}{
		"media_id": mediaID.String(),
		"site_id":  siteID.String(),
		"filename": header.Filename,
		"mime":     mimeType,
	})

	s.invalidateCache(ctx, siteID)

	variants, _ := s.repo.GetVariants(ctx, mediaID)
	m.Variants = variants

	return m, nil
}

func (s *Service) processImage(ctx context.Context, fullPath string, siteID, mediaID uuid.UUID, ext string) (*imgpkg.ImageInfo, error) {
	f, err := os.Open(fullPath)
	if err != nil {
		localPath := filepath.Join(s.basePath, fmt.Sprintf("%s/%s.%s", siteID.String(), mediaID.String(), ext))
		f, err = os.Open(localPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open image: %w", err)
		}
	}
	defer func() { _ = f.Close() }()

	src, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	bounds := src.Bounds()
	info := &imgpkg.ImageInfo{Width: bounds.Dx(), Height: bounds.Dy()}

	storageDir := s.basePath

	sizes := map[VariantType]imgpkg.VariantDimensions{
		VariantThumbnail: {MaxWidth: 150, MaxHeight: 150},
		VariantSmall:     {MaxWidth: 320, MaxHeight: 240},
		VariantMedium:    {MaxWidth: 800, MaxHeight: 600},
		VariantLarge:     {MaxWidth: 1920, MaxHeight: 1080},
	}

	outputMimeTypes := []string{mimeTypeFromExt(ext)}
	if ext == "jpg" || ext == "jpeg" || ext == "png" {
		outputMimeTypes = append(outputMimeTypes, "image/webp")
	}

	for variantType, dims := range sizes {
		for _, outputMime := range outputMimeTypes {
			outputExt := extFromMimeType(outputMime)
			variantKey := fmt.Sprintf("%s/variants/%s/%s/%s.%s",
				siteID.String(), string(variantType), mediaID.String(), mediaID.String(), outputExt)
			variantFullPath := filepath.Join(storageDir, variantKey)

			if err := os.MkdirAll(filepath.Dir(variantFullPath), 0755); err != nil {
				s.log.Warn("failed to create variant directory", "error", err)
				continue
			}

			outputPath := variantFullPath
			if outputMime == "image/webp" {
				outputPath = variantFullPath
				srcFile := fullPath
				if err := imgpkg.ConvertToWebP(srcFile, outputPath, 80); err != nil {
					s.log.Warn("failed to convert to webp", "error", err, "variant", variantType)
					continue
				}
			} else {
				if err := imgpkg.GenerateVariant(fullPath, outputPath, imgpkg.VariantSize(variantType)); err != nil {
					s.log.Warn("failed to generate variant", "error", err, "variant", variantType)
					continue
				}
			}

			var variantWidth, variantHeight int
			if dims.MaxWidth > 0 && dims.MaxHeight > 0 {
				if info.Width <= dims.MaxWidth && info.Height <= dims.MaxHeight {
					variantWidth = info.Width
					variantHeight = info.Height
				} else {
					ratio := float64(info.Width) / float64(info.Height)
					targetRatio := float64(dims.MaxWidth) / float64(dims.MaxHeight)
					if ratio > targetRatio {
						variantWidth = dims.MaxWidth
						variantHeight = int(float64(dims.MaxWidth) / ratio)
					} else {
						variantHeight = dims.MaxHeight
						variantWidth = int(float64(dims.MaxHeight) * ratio)
					}
				}
			}

			vInfo, _ := os.Stat(outputPath)
			var fileSize int64
			if vInfo != nil {
				fileSize = vInfo.Size()
			}

			variant := &MediaVariant{
				ID:         uuid.New(),
				MediaID:    mediaID,
				Variant:    variantType,
				Width:      variantWidth,
				Height:     variantHeight,
				FileSize:   fileSize,
				MimeType:   outputMime,
				StorageKey: variantKey,
				CreatedAt:  time.Now(),
			}

			if err := s.repo.CreateVariant(ctx, variant); err != nil {
				s.log.Warn("failed to save variant", "error", err, "variant", variantType)
			}
		}
	}

	return info, nil
}

func (s *Service) GetByID(ctx context.Context, siteID, mediaID uuid.UUID) (*Media, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("media:%s:%s", siteID.String(), mediaID.String())
	var cached Media
	if ok, _ := s.cache.GetJSON(ctx, cacheKey, &cached); ok {
		return &cached, nil
	}

	m, err := s.repo.GetByID(ctx, siteID, mediaID)
	if err != nil {
		return nil, err
	}

	variants, err := s.repo.GetVariants(ctx, mediaID)
	if err == nil {
		for i := range variants {
			variants[i].URL = s.storage.URL(variants[i].StorageKey)
		}
		m.Variants = variants
	}

	if err := s.cache.SetJSON(ctx, cacheKey, m, 5*time.Minute); err != nil {
		s.log.Warn("failed to cache media", "error", err, "key", cacheKey)
	}

	return m, nil
}

func (s *Service) List(ctx context.Context, req MediaListRequest) (*MediaListResponse, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	return s.repo.List(ctx, req)
}

func (s *Service) Update(ctx context.Context, siteID, mediaID uuid.UUID, req UpdateMediaRequest) (*Media, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	m, err := s.repo.Update(ctx, siteID, mediaID, req)
	if err != nil {
		return nil, err
	}

	s.auditLog.Log(ctx, audit.Entry{
		EntityType: "media",
		EntityID:   &mediaID,
		Action:     audit.Action("media.updated"),
	})

	s.fireEvent(ctx, EventMediaUpdated, map[string]interface{}{
		"media_id": mediaID.String(),
		"site_id":  siteID.String(),
	})

	s.invalidateCache(ctx, siteID)

	return m, nil
}

func (s *Service) Delete(ctx context.Context, siteID, mediaID uuid.UUID) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	m, err := s.repo.GetByID(ctx, siteID, mediaID)
	if err != nil {
		return err
	}

	if err := s.repo.SoftDelete(ctx, siteID, mediaID); err != nil {
		return err
	}

	variants, _ := s.repo.GetVariants(ctx, mediaID)
	for _, v := range variants {
		if err := s.storage.Delete(ctx, v.StorageKey); err != nil {
			s.log.Warn("failed to delete variant", "error", err, "key", v.StorageKey)
		}
	}
	if err := s.storage.Delete(ctx, m.StorageKey); err != nil {
		s.log.Warn("failed to delete storage file", "error", err, "key", m.StorageKey)
	}

	s.auditLog.Log(ctx, audit.Entry{
		EntityType: "media",
		EntityID:   &mediaID,
		Action:     audit.Action("media.deleted"),
		Payload:    map[string]interface{}{"original_name": m.OriginalName},
	})

	s.fireEvent(ctx, EventMediaDeleted, map[string]interface{}{
		"media_id": mediaID.String(),
		"site_id":  siteID.String(),
	})

	s.invalidateCache(ctx, siteID)

	return nil
}

func (s *Service) Restore(ctx context.Context, siteID, mediaID uuid.UUID) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	if err := s.repo.Restore(ctx, siteID, mediaID); err != nil {
		return err
	}

	s.fireEvent(ctx, EventMediaRestored, map[string]interface{}{
		"media_id": mediaID.String(),
		"site_id":  siteID.String(),
	})

	s.invalidateCache(ctx, siteID)

	return nil
}

func (s *Service) Move(ctx context.Context, siteID uuid.UUID, mediaIDs []uuid.UUID, folderID *uuid.UUID) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	if folderID != nil {
		if _, err := s.repo.GetFolderByID(ctx, siteID, *folderID); err != nil {
			return err
		}
	}

	return s.repo.Move(ctx, siteID, mediaIDs, folderID)
}

func (s *Service) Copy(ctx context.Context, siteID, userID uuid.UUID, mediaIDs []uuid.UUID, destFolderID *uuid.UUID) ([]*Media, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	var copied []*Media

	for _, mediaID := range mediaIDs {
		src, err := s.repo.GetByID(ctx, siteID, mediaID)
		if err != nil {
			continue
		}

		data, err := s.storage.Download(ctx, src.StorageKey)
		if err != nil {
			continue
		}
		defer func() { _ = data.Close() }()

		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, data); err != nil {
			continue
		}

		mimeType := src.MimeType
		ext := src.Extension

		now := time.Now()
		newID := uuid.New()
		storageKey := fmt.Sprintf("%s/copies/%s.%s", siteID.String(), newID.String(), ext)

		if err := s.storage.Upload(ctx, storageKey, bytes.NewReader(buf.Bytes())); err != nil {
			continue
		}

		hashBytes := sha256.Sum256(buf.Bytes())
		hash := hex.EncodeToString(hashBytes[:])

		newMedia := &Media{
			ID:              newID,
			SiteID:          siteID,
			FolderID:        destFolderID,
			Filename:        path.Base(storageKey),
			OriginalName:    "copy-of-" + src.OriginalName,
			MimeType:        mimeType,
			Extension:       ext,
			Size:            src.Size,
			Width:           src.Width,
			Height:          src.Height,
			Hash:            hash,
			AltText:         src.AltText,
			Caption:         src.Caption,
			StorageProvider: s.driverName,
			StorageKey:      storageKey,
			Metadata:        src.Metadata,
			CreatedBy:       userID,
			CreatedAt:       now,
			UpdatedAt:       now,
		}

		if err := s.repo.Create(ctx, newMedia); err != nil {
			if delErr := s.storage.Delete(ctx, storageKey); delErr != nil {
				s.log.Warn("failed to clean up copied file", "error", delErr, "key", storageKey)
			}
			continue
		}

		copied = append(copied, newMedia)
	}

	if copied == nil {
		copied = []*Media{}
	}

	return copied, nil
}

func (s *Service) Search(ctx context.Context, siteID uuid.UUID, query string, page, perPage int) (*MediaListResponse, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	req := MediaListRequest{
		SiteID:  siteID,
		Search:  query,
		Page:    page,
		PerPage: perPage,
		Sort:    "created_at",
		Order:   "DESC",
	}

	return s.repo.List(ctx, req)
}

func (s *Service) CreateFolder(ctx context.Context, siteID, userID uuid.UUID, req CreateFolderRequest) (*Folder, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	if err := s.validator.ValidateFolderName(req.Name); err != nil {
		return nil, err
	}

	now := time.Now()
	f := &Folder{
		ID:          uuid.New(),
		SiteID:      siteID,
		ParentID:    req.ParentID,
		Name:        req.Name,
		Slug:        Slugify(req.Name),
		Description: req.Description,
		SortOrder:   0,
		CreatedBy:   userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.CreateFolder(ctx, f); err != nil {
		if strings.Contains(err.Error(), "duplicate") || strings.Contains(err.Error(), "unique") {
			f.Slug = Slugify(req.Name) + "-" + uuid.New().String()[:8]
			if err := s.repo.CreateFolder(ctx, f); err != nil {
				return nil, fmt.Errorf("failed to create folder: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to create folder: %w", err)
		}
	}

	s.fireEvent(ctx, EventFolderCreated, map[string]interface{}{
		"folder_id": f.ID.String(),
		"site_id":   siteID.String(),
		"name":      req.Name,
	})

	return f, nil
}

func (s *Service) GetFolderByID(ctx context.Context, siteID, folderID uuid.UUID) (*Folder, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	return s.repo.GetFolderByID(ctx, siteID, folderID)
}

func (s *Service) ListFolders(ctx context.Context, siteID uuid.UUID) ([]Folder, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	cacheKey := fmt.Sprintf("folders:%s", siteID.String())
	var cached []Folder
	if ok, _ := s.cache.GetJSON(ctx, cacheKey, &cached); ok {
		return cached, nil
	}

	folders, err := s.repo.ListFolders(ctx, siteID)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetJSON(ctx, cacheKey, folders, 5*time.Minute); err != nil {
		s.log.Warn("failed to cache folders", "error", err, "key", cacheKey)
	}

	return folders, nil
}

func (s *Service) UpdateFolder(ctx context.Context, siteID, folderID uuid.UUID, req UpdateFolderRequest) (*Folder, error) {
	if err := s.checkDB(); err != nil {
		return nil, err
	}
	if req.Name != nil {
		if err := s.validator.ValidateFolderName(*req.Name); err != nil {
			return nil, err
		}
	}

	f, err := s.repo.UpdateFolder(ctx, siteID, folderID, req)
	if err != nil {
		return nil, err
	}

	s.invalidateCache(ctx, siteID)

	return f, nil
}

func (s *Service) DeleteFolder(ctx context.Context, siteID, folderID uuid.UUID) error {
	if err := s.checkDB(); err != nil {
		return err
	}
	count, err := s.repo.GetFolderChildCount(ctx, folderID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrFolderNotEmpty
	}

	count, err = s.repo.GetFolderSubfolderCount(ctx, folderID)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrFolderNotEmpty
	}

	if err := s.repo.DeleteFolder(ctx, siteID, folderID); err != nil {
		return err
	}

	s.fireEvent(ctx, EventFolderDeleted, map[string]interface{}{
		"folder_id": folderID.String(),
		"site_id":   siteID.String(),
	})

	s.invalidateCache(ctx, siteID)

	return nil
}

func (s *Service) invalidateCache(ctx context.Context, siteID uuid.UUID) {
	if err := s.cache.Delete(ctx, fmt.Sprintf("folders:%s", siteID.String())); err != nil {
		s.log.Warn("failed to invalidate folders cache", "error", err, "site_id", siteID)
	}
	pattern := fmt.Sprintf("media:%s:*", siteID.String())
	if err := s.cache.Delete(ctx, pattern); err != nil {
		s.log.Warn("failed to invalidate media cache", "error", err, "pattern", pattern)
	}
}

func mimeTypeFromExt(ext string) string {
	switch ext {
	case "jpg", "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	case "gif":
		return "image/gif"
	case "webp":
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

func extFromMimeType(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return "jpg"
	case "image/png":
		return "png"
	case "image/gif":
		return "gif"
	case "image/webp":
		return "webp"
	default:
		return "jpg"
	}
}
