package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path"
	"strings"
)

type S3Driver struct {
	Bucket   string
	Region   string
	Endpoint string
	BaseURL  string
	// In a real implementation, this would use the AWS SDK
	// For now, we use a local-filesystem-based implementation that
	// demonstrates the interface pattern. The actual S3 integration
	// would replace the upload/download methods with S3 API calls.
	LocalPath string
}

func NewS3Driver(bucket, region, endpoint, accessKey, secretKey, baseURL, localPath string) *S3Driver {
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com", bucket, region)
	}
	return &S3Driver{
		Bucket:    bucket,
		Region:    region,
		Endpoint:  endpoint,
		BaseURL:   baseURL,
		LocalPath: localPath,
	}
}

func (d *S3Driver) Upload(ctx context.Context, key string, reader io.Reader) error {
	fullPath := path.Join(d.LocalPath, key)
	dir := path.Dir(fullPath)
	if err := mkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("s3: failed to create directory %s: %w", dir, err)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("s3: failed to read data: %w", err)
	}

	if err := writeFile(fullPath, data, 0o644); err != nil {
		return fmt.Errorf("s3: failed to write file %s: %w", fullPath, err)
	}

	return nil
}

func (d *S3Driver) Download(ctx context.Context, key string) (io.ReadCloser, error) {
	fullPath := path.Join(d.LocalPath, key)
	data, err := readFile(fullPath)
	if err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return nil, fmt.Errorf("s3: file not found: %s", key)
		}
		return nil, fmt.Errorf("s3: failed to read file %s: %w", fullPath, err)
	}
	return io.NopCloser(bytes.NewReader(data)), nil
}

func (d *S3Driver) Delete(ctx context.Context, key string) error {
	fullPath := path.Join(d.LocalPath, key)
	if err := removeFile(fullPath); err != nil {
		if strings.Contains(err.Error(), "no such file") {
			return nil
		}
		return fmt.Errorf("s3: failed to delete file %s: %w", fullPath, err)
	}
	return nil
}

func (d *S3Driver) Exists(ctx context.Context, key string) (bool, error) {
	fullPath := path.Join(d.LocalPath, key)
	_, err := statFile(fullPath)
	if err == nil {
		return true, nil
	}
	if strings.Contains(err.Error(), "no such file") {
		return false, nil
	}
	return false, fmt.Errorf("s3: failed to check file %s: %w", fullPath, err)
}

func (d *S3Driver) URL(key string) string {
	base := strings.TrimRight(d.BaseURL, "/")
	return fmt.Sprintf("%s/%s", base, strings.TrimLeft(key, "/"))
}

func (d *S3Driver) GetBucket() string {
	return d.Bucket
}

func (d *S3Driver) GetRegion() string {
	return d.Region
}

func (d *S3Driver) GetEndpoint() string {
	return d.Endpoint
}
