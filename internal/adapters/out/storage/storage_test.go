package storage_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/storage"
)

func TestStorageAdapter(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	adapter, err := storage.NewLocalStorageAdapter(tmpDir, string(storage.DriverGarageLocal))
	if err != nil {
		t.Fatalf("failed initializing storage adapter: %v", err)
	}

	t.Run("Given a configured storage adapter", func(t *testing.T) {
		t.Run("When checking health then it succeeds", func(t *testing.T) {
			if err := adapter.CheckHealth(ctx); err != nil {
				t.Errorf("expected health check to pass: %v", err)
			}
		})

		t.Run("When ensuring bucket and uploading a bundle", func(t *testing.T) {
			bucket := "my-fleet-bucket"
			key := "deployments/worker-1/bundle.js"
			data := []byte("console.log('worker code');")

			if err := adapter.EnsureBucket(ctx, bucket); err != nil {
				t.Fatalf("expected bucket creation to succeed: %v", err)
			}

			if err := adapter.UploadBundle(ctx, bucket, key, data); err != nil {
				t.Fatalf("expected upload to succeed: %v", err)
			}

			t.Run("Then object exists on disk with identical content", func(t *testing.T) {
				expectedPath := filepath.Join(tmpDir, bucket, key)
				content, err := os.ReadFile(expectedPath)
				if err != nil {
					t.Fatalf("expected file to exist at %s: %v", expectedPath, err)
				}
				if string(content) != string(data) {
					t.Errorf("expected content %s, got %s", data, content)
				}
			})
		})
	})
}
