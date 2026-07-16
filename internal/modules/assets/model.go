package assets

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

type AssetType string

const (
	AssetTypeImage    AssetType = "image"
	AssetTypeVideo    AssetType = "video"
	AssetTypeDocument AssetType = "document"
	AssetTypeAudio    AssetType = "audio"
	AssetTypeOther    AssetType = "other"
)

type PostAssetType string

const (
	PostAssetFeaturedImage PostAssetType = "featured_image"
	PostAssetGallery       PostAssetType = "gallery"
	PostAssetAttachment    PostAssetType = "attachment"
)

const (
	EventAssetCreated kernel.EventType = "asset.created"
	EventAssetDeleted kernel.EventType = "asset.deleted"
	EventAssetUpdated  kernel.EventType = "asset.updated"
)

var (
	ErrAssetNotFound     = errors.New("asset not found")
	ErrAssetNotInSite    = errors.New("asset does not belong to this site")
	ErrInvalidFileType   = errors.New("invalid file type")
	ErrFileTooLarge      = errors.New("file too large")
	ErrInvalidFile       = errors.New("invalid file content")
	ErrDatabaseNotAvail  = errors.New("database not available")
	ErrInvalidPagination = errors.New("invalid pagination parameters")
	ErrPostNotFound      = errors.New("post not found")
)

type Asset struct {
	ID              uuid.UUID              `json:"id"`
	SiteID          uuid.UUID              `json:"site_id"`
	UserID          uuid.UUID              `json:"user_id"`
	Filename        string                 `json:"filename"`
	OriginalName    string                 `json:"original_name"`
	MimeType        string                 `json:"mime_type"`
	Extension       string                 `json:"extension"`
	Size            int64                  `json:"size"`
	Width           *int                   `json:"width,omitempty"`
	Height          *int                   `json:"height,omitempty"`
	AltText         string                 `json:"alt_text,omitempty"`
	Title           string                 `json:"title,omitempty"`
	Caption         string                 `json:"caption,omitempty"`
	Description     string                 `json:"description,omitempty"`
	ThumbnailPath   string                 `json:"thumbnail_path,omitempty"`
	OptimizedPath   string                 `json:"optimized_path,omitempty"`
	StorageProvider string                 `json:"storage_provider"`
	StoragePath     string                 `json:"storage_path"`
	URL             string                 `json:"url"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	DeletedAt       *time.Time             `json:"deleted_at,omitempty"`
}

type PostAsset struct {
	ID        uuid.UUID    `json:"id"`
	PostID    uuid.UUID    `json:"post_id"`
	AssetID   uuid.UUID    `json:"asset_id"`
	SortOrder int          `json:"sort_order"`
	Type      PostAssetType `json:"type"`
	CreatedAt time.Time    `json:"created_at"`
	Asset     *Asset       `json:"asset,omitempty"`
}

type UploadRequest struct {
	AltText     string `json:"alt_text,omitempty"`
	Title       string `json:"title,omitempty"`
	Caption     string `json:"caption,omitempty"`
	Description string `json:"description,omitempty"`
}

type UpdateAssetRequest struct {
	AltText     *string `json:"alt_text,omitempty"`
	Title       *string `json:"title,omitempty"`
	Caption     *string `json:"caption,omitempty"`
	Description *string `json:"description,omitempty"`
}

type AssetListRequest struct {
	SiteID    uuid.UUID
	Type      AssetType
	Extension string
	Search    string
	Page      int
	PerPage   int
	Sort      string
	Order     string
}

type AssetListResponse struct {
	Assets  []Asset `json:"assets"`
	Total   int     `json:"total"`
	Page    int     `json:"page"`
	PerPage int     `json:"per_page"`
}

type LinkAssetRequest struct {
	PostID    uuid.UUID      `json:"post_id"`
	AssetID   uuid.UUID      `json:"asset_id"`
	SortOrder int            `json:"sort_order"`
	Type      PostAssetType  `json:"type"`
}

type FileValidationConfig struct {
	MaxImageSize    int64
	MaxVideoSize    int64
	MaxDocumentSize int64
	MaxAudioSize    int64
	AllowedMIMETypes map[string][]string
}

func DefaultFileValidationConfig() FileValidationConfig {
	return FileValidationConfig{
		MaxImageSize:    10 * 1024 * 1024,
		MaxVideoSize:    100 * 1024 * 1024,
		MaxDocumentSize: 10 * 1024 * 1024,
		MaxAudioSize:    50 * 1024 * 1024,
		AllowedMIMETypes: map[string][]string{
			"image":    {"image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml"},
			"video":    {"video/mp4", "video/webm", "video/ogg"},
			"document": {"application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "text/csv", "text/plain", "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
			"audio":    {"audio/mpeg", "audio/wav", "audio/ogg"},
		},
	}
}

var ExtensionMIMEMap = map[string]string{
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"webp": "image/webp",
	"svg":  "image/svg+xml",
	"mp4":  "video/mp4",
	"webm": "video/webm",
	"ogg":  "video/ogg",
	"pdf":  "application/pdf",
	"doc":  "application/msword",
	"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"csv":  "text/csv",
	"txt":  "text/plain",
	"xls":  "application/vnd.ms-excel",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"mp3":  "audio/mpeg",
	"wav":  "audio/wav",
}

var MagicBytesMap = map[string][]byte{
	"image/jpeg":                       {0xFF, 0xD8, 0xFF},
	"image/png":                        {0x89, 0x50, 0x4E, 0x47},
	"image/gif":                        {0x47, 0x49, 0x46},
	"image/webp":                       {0x52, 0x49, 0x46, 0x46},
	"image/svg+xml":                    {0x3C, 0x3F, 0x78, 0x6D, 0x6C},
	"application/pdf":                  {0x25, 0x50, 0x44, 0x46},
	"application/msword":               {0xD0, 0xCF, 0x11, 0xE0},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {0x50, 0x4B, 0x03, 0x04},
	"video/mp4":                        {0x00, 0x00, 0x00},
	"video/webm":                       {0x1A, 0x45, 0xDF, 0xA3},
}

func DetectMimeTypeFromExtension(ext string) string {
	if mime, ok := ExtensionMIMEMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}
