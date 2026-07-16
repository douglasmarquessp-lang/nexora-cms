package media

import (
	"crypto/sha256"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path"
	"regexp"
	"strings"
)

var folderNameRegex = regexp.MustCompile(`^[a-zA-Z0-9\s\-_\.]+$`)

type Validator struct {
	cfg FileValidationConfig
}

func NewValidator(cfg FileValidationConfig) *Validator {
	return &Validator{cfg: cfg}
}

func (v *Validator) ValidateFile(header *multipart.FileHeader, file multipart.File) (string, string, error) {
	ext := strings.ToLower(strings.TrimPrefix(path.Ext(header.Filename), "."))
	if ext == "" {
		return "", "", ErrInvalidFileType
	}

	mimeType, err := v.detectMimeType(file, ext)
	if err != nil {
		return "", "", err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", "", ErrInvalidFile
	}

	if !v.isAllowedMimeType(mimeType) {
		return "", "", ErrInvalidFileType
	}

	if header.Size > v.cfg.MaxFileSize {
		return "", "", ErrFileTooLarge
	}

	return mimeType, ext, nil
}

func (v *Validator) detectMimeType(file multipart.File, ext string) (string, error) {
	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		return "", ErrInvalidFile
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", ErrInvalidFile
	}

	mimeType := http.DetectContentType(buf)
	expectedMime, ok := ExtensionMIMEMap[ext]
	if ok && mimeType != expectedMime {
		mimeType = expectedMime
	}

	return mimeType, nil
}

func (v *Validator) isAllowedMimeType(mimeType string) bool {
	for _, mimes := range v.cfg.AllowedMIMETypes {
		for _, allowed := range mimes {
			if allowed == mimeType {
				return true
			}
		}
	}
	return false
}

func (v *Validator) ValidateFolderName(name string) error {
	if len(strings.TrimSpace(name)) == 0 {
		return ErrInvalidFolderName
	}
	if len(name) > 255 {
		return ErrInvalidFolderName
	}
	if !folderNameRegex.MatchString(name) {
		return ErrInvalidFolderName
	}
	return nil
}

func ComputeHash(file multipart.File) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("failed to seek file after hash: %w", err)
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func IsImageType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml"
}

func Slugify(name string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = regexp.MustCompile(`[^a-z0-9\-]`).ReplaceAllString(slug, "")
	slug = regexp.MustCompile(`-+`).ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "folder"
	}
	return slug
}
