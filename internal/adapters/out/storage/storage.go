package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// StorageDriver specifies the storage driver type.
type StorageDriver string

const (
	DriverGarageLocal StorageDriver = "garage_local"
	DriverExternalS3  StorageDriver = "external_s3"
)

// LocalStorageAdapter implements usecase.StoragePort for local disk / Garage S3 storage on bare metal.
type LocalStorageAdapter struct {
	baseDir    string
	driverName string
	mu         sync.RWMutex
}

// NewLocalStorageAdapter creates a new LocalStorageAdapter.
func NewLocalStorageAdapter(baseDir string, driverName string) (*LocalStorageAdapter, error) {
	if driverName == "" {
		driverName = string(DriverGarageLocal)
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to initialize storage base directory: %w", err)
	}
	return &LocalStorageAdapter{
		baseDir:    baseDir,
		driverName: driverName,
	}, nil
}

// EnsureBucket ensures the storage bucket exists.
func (a *LocalStorageAdapter) EnsureBucket(ctx context.Context, bucketName string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	bucketPath := filepath.Join(a.baseDir, bucketName)
	return os.MkdirAll(bucketPath, 0755)
}

// UploadBundle stores a worker deployment bundle.
func (a *LocalStorageAdapter) UploadBundle(ctx context.Context, bucketName, objectKey string, data []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	targetPath := filepath.Join(a.baseDir, bucketName, objectKey)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	return os.WriteFile(targetPath, data, 0644)
}

// DownloadBundle reads a stored worker bundle.
func (a *LocalStorageAdapter) DownloadBundle(ctx context.Context, bucketName, objectKey string) ([]byte, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	targetPath := filepath.Join(a.baseDir, bucketName, objectKey)
	return os.ReadFile(targetPath)
}

// CheckHealth verifies that the storage backend is reachable and writable.
func (a *LocalStorageAdapter) CheckHealth(ctx context.Context) error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	testFile := filepath.Join(a.baseDir, ".healthcheck")
	if err := os.WriteFile(testFile, []byte("ok"), 0644); err != nil {
		return err
	}
	_ = os.Remove(testFile)
	return nil
}

// DriverName returns the active storage driver identifier.
func (a *LocalStorageAdapter) DriverName() string {
	return a.driverName
}
