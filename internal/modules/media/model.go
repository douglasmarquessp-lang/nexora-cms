package media

import (
	"errors"
	"time"

	"github.com/google/uuid"

	"nexora/internal/kernel"
)

type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeDocument MediaType = "document"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeOther    MediaType = "other"
)

type VariantType string

const (
	VariantThumbnail VariantType = "thumbnail"
	VariantSmall     VariantType = "small"
	VariantMedium    VariantType = "medium"
	VariantLarge     VariantType = "large"
	VariantOriginal  VariantType = "original"
)

var AllVariants = []VariantType{VariantThumbnail, VariantSmall, VariantMedium, VariantLarge, VariantOriginal}

const (
	EventMediaUploaded kernel.EventType = "media.uploaded"
	EventMediaUpdated  kernel.EventType = "media.updated"
	EventMediaDeleted  kernel.EventType = "media.deleted"
	EventMediaRestored kernel.EventType = "media.restored"
	EventFolderCreated kernel.EventType = "folder.created"
	EventFolderDeleted kernel.EventType = "folder.deleted"
)

var (
	ErrMediaNotFound       = errors.New("media not found")
	ErrMediaNotInSite      = errors.New("media does not belong to this site")
	ErrFolderNotFound      = errors.New("folder not found")
	ErrInvalidFileType     = errors.New("invalid file type")
	ErrFileTooLarge        = errors.New("file too large")
	ErrInvalidFile         = errors.New("invalid file content")
	ErrDatabaseNotAvail    = errors.New("database not available")
	ErrInvalidPagination   = errors.New("invalid pagination parameters")
	ErrDuplicateFile       = errors.New("duplicate file (hash collision)")
	ErrStorageLimitReached = errors.New("storage limit reached for this site")
	ErrInvalidFolderName   = errors.New("invalid folder name")
	ErrFolderNotEmpty      = errors.New("folder is not empty")
	ErrFolderDepthExceeded = errors.New("folder nesting depth exceeded")
)

type Media struct {
	ID              uuid.UUID              `json:"id"`
	SiteID          uuid.UUID              `json:"site_id"`
	FolderID        *uuid.UUID             `json:"folder_id,omitempty"`
	Filename        string                 `json:"filename"`
	OriginalName    string                 `json:"original_name"`
	MimeType        string                 `json:"mime_type"`
	Extension       string                 `json:"extension"`
	Size            int64                  `json:"size"`
	Width           *int                   `json:"width,omitempty"`
	Height          *int                   `json:"height,omitempty"`
	Duration        int                    `json:"duration,omitempty"`
	Hash            string                 `json:"hash"`
	AltText         string                 `json:"alt_text,omitempty"`
	Caption         string                 `json:"caption,omitempty"`
	StorageProvider string                 `json:"storage_provider"`
	StorageKey      string                 `json:"storage_key"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	CreatedBy       uuid.UUID              `json:"created_by"`
	DeletedAt       *time.Time             `json:"deleted_at,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`

	Variants []MediaVariant `json:"variants,omitempty"`
	Folder   *Folder        `json:"folder,omitempty"`
}

type MediaVariant struct {
	ID         uuid.UUID              `json:"id"`
	MediaID    uuid.UUID              `json:"media_id"`
	Variant    VariantType            `json:"variant"`
	Width      int                    `json:"width"`
	Height     int                    `json:"height"`
	FileSize   int64                  `json:"file_size"`
	MimeType   string                 `json:"mime_type"`
	StorageKey string                 `json:"storage_key"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`

	URL string `json:"url,omitempty"`
}

type Folder struct {
	ID          uuid.UUID  `json:"id"`
	SiteID      uuid.UUID  `json:"site_id"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	Name        string     `json:"name"`
	Slug        string     `json:"slug"`
	Description string     `json:"description,omitempty"`
	SortOrder   int        `json:"sort_order"`
	CreatedBy   uuid.UUID  `json:"created_by"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type UploadRequest struct {
	FolderID *uuid.UUID `json:"folder_id,omitempty"`
	AltText  string     `json:"alt_text,omitempty"`
	Caption  string     `json:"caption,omitempty"`
}

type UpdateMediaRequest struct {
	FolderID *uuid.UUID `json:"folder_id,omitempty"`
	AltText  *string    `json:"alt_text,omitempty"`
	Caption  *string    `json:"caption,omitempty"`
}

type MediaListRequest struct {
	SiteID    uuid.UUID
	FolderID  *uuid.UUID
	Type      MediaType
	Extension string
	Search    string
	Sort      string
	Order     string
	Page      int
	PerPage   int
	MimeType  string
	MinSize   *int64
	MaxSize   *int64
	FromDate  *time.Time
	ToDate    *time.Time
	CreatedBy *uuid.UUID
}

type MediaListResponse struct {
	Media   []Media `json:"media"`
	Total   int     `json:"total"`
	Page    int     `json:"page"`
	PerPage int     `json:"per_page"`
}

type MoveMediaRequest struct {
	MediaIDs []uuid.UUID `json:"media_ids"`
	FolderID *uuid.UUID  `json:"folder_id,omitempty"`
}

type CopyMediaRequest struct {
	MediaIDs []uuid.UUID `json:"media_ids"`
	FolderID *uuid.UUID  `json:"folder_id,omitempty"`
}

type CreateFolderRequest struct {
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
}

type UpdateFolderRequest struct {
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
}

type FolderListResponse struct {
	Folders []Folder `json:"folders"`
}

type MediaUploadProgress struct {
	MediaID  string  `json:"media_id"`
	Filename string  `json:"filename"`
	Progress float64 `json:"progress"`
	Status   string  `json:"status"`
	Error    string  `json:"error,omitempty"`
}

type FileValidationConfig struct {
	MaxFileSize       int64
	AllowedMIMETypes  map[string][]string
	MaxStoragePerSite int64
}

func DefaultFileValidationConfig() FileValidationConfig {
	return FileValidationConfig{
		MaxFileSize: 50 * 1024 * 1024,
		AllowedMIMETypes: map[string][]string{
			"image":    {"image/jpeg", "image/png", "image/gif", "image/webp", "image/avif", "image/svg+xml"},
			"video":    {"video/mp4", "video/webm", "video/ogg", "video/quicktime"},
			"document": {"application/pdf", "application/msword", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", "text/csv", "text/plain", "application/vnd.ms-excel", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
			"audio":    {"audio/mpeg", "audio/wav", "audio/ogg", "audio/aac", "audio/flac"},
		},
		MaxStoragePerSite: 1024 * 1024 * 1024,
	}
}

var ExtensionMIMEMap = map[string]string{
	"jpg":  "image/jpeg",
	"jpeg": "image/jpeg",
	"png":  "image/png",
	"gif":  "image/gif",
	"webp": "image/webp",
	"avif": "image/avif",
	"svg":  "image/svg+xml",
	"mp4":  "video/mp4",
	"webm": "video/webm",
	"ogg":  "video/ogg",
	"mov":  "video/quicktime",
	"pdf":  "application/pdf",
	"doc":  "application/msword",
	"docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	"csv":  "text/csv",
	"txt":  "text/plain",
	"xls":  "application/vnd.ms-excel",
	"xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	"mp3":  "audio/mpeg",
	"wav":  "audio/wav",
	"aac":  "audio/aac",
	"flac": "audio/flac",
}

func DetectMimeTypeFromExtension(ext string) string {
	if mime, ok := ExtensionMIMEMap[ext]; ok {
		return mime
	}
	return "application/octet-stream"
}

func ClassifyMIMEType(mimeType string) MediaType {
	if len(mimeType) >= 5 && mimeType[:5] == "image" {
		return MediaTypeImage
	}
	if len(mimeType) >= 5 && mimeType[:5] == "video" {
		return MediaTypeVideo
	}
	if len(mimeType) >= 5 && mimeType[:5] == "audio" {
		return MediaTypeAudio
	}
	return MediaTypeDocument
}
