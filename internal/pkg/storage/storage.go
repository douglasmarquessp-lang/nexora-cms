package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Driver interface {
	Upload(ctx context.Context, path string, reader io.Reader) error
	Download(ctx context.Context, path string) (io.ReadCloser, error)
	Delete(ctx context.Context, path string) error
	Exists(ctx context.Context, path string) (bool, error)
	URL(path string) string
}

type LocalDriver struct {
	BasePath string
	BaseURL  string
}

func NewLocalDriver(basePath, baseURL string) *LocalDriver {
	return &LocalDriver{BasePath: basePath, BaseURL: baseURL}
}

func (d *LocalDriver) Upload(ctx context.Context, path string, reader io.Reader) error {
	fullPath := filepath.Join(d.BasePath, path)
	dir := filepath.Dir(fullPath)
	if err := mkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("storage: failed to create directory %s: %w", dir, err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("storage: failed to read data: %w", err)
	}

	if err := writeFile(fullPath, data, 0644); err != nil {
		return fmt.Errorf("storage: failed to write file %s: %w", fullPath, err)
	}

	return nil
}

func (d *LocalDriver) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := filepath.Join(d.BasePath, path)
	data, err := readFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("storage: file not found: %s", path)
		}
		return nil, fmt.Errorf("storage: failed to read file %s: %w", fullPath, err)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (d *LocalDriver) Delete(ctx context.Context, path string) error {
	fullPath := filepath.Join(d.BasePath, path)
	if err := removeFile(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("storage: failed to delete file %s: %w", fullPath, err)
	}
	return nil
}

func (d *LocalDriver) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := filepath.Join(d.BasePath, path)
	_, err := statFile(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("storage: failed to check file %s: %w", fullPath, err)
}

func (d *LocalDriver) URL(path string) string {
	return fmt.Sprintf("%s/%s", d.BaseURL, path)
}

const (
	DriverLocal = "local"
	DriverS3    = "s3"
	DriverR2    = "r2"
	DriverMinIO = "minio"
)

// NewDriver creates a storage driver based on the driver name.
// For S3-compatible drivers (s3, r2, minio), bucket, region, endpoint, accessKey, secretKey
// are required. For local driver, only localPath and baseURL are used.
func NewDriver(driverName, localPath, baseURL, bucket, region, endpoint, accessKey, secretKey string) Driver {
	switch driverName {
	case DriverS3, DriverR2, DriverMinIO:
		return NewS3Driver(bucket, region, endpoint, accessKey, secretKey, baseURL, localPath)
	default:
		return NewLocalDriver(localPath, baseURL)
	}
}
