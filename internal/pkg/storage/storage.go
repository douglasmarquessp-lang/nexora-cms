package storage

import (
	"context"
	"fmt"
	"io"
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
}

func NewLocalDriver(basePath string) *LocalDriver {
	return &LocalDriver{BasePath: basePath}
}

var errNotImplemented = fmt.Errorf("storage: driver not fully implemented")

func (d *LocalDriver) Upload(ctx context.Context, path string, reader io.Reader) error {
	return fmt.Errorf("%w: upload not available in this version", errNotImplemented)
}

func (d *LocalDriver) Download(ctx context.Context, path string) (io.ReadCloser, error) {
	return nil, fmt.Errorf("%w: download not available in this version", errNotImplemented)
}

func (d *LocalDriver) Delete(ctx context.Context, path string) error {
	return fmt.Errorf("%w: delete not available in this version", errNotImplemented)
}

func (d *LocalDriver) Exists(ctx context.Context, path string) (bool, error) {
	return false, fmt.Errorf("%w: exists check not available in this version", errNotImplemented)
}

func (d *LocalDriver) URL(path string) string {
	return "/" + path
}
