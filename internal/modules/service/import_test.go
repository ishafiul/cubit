package service_test

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/storage"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	srvModule "github.com/ishaf/cubit/internal/modules/service"
)

func TestStateDataImportTooling(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "import-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	defer database.Close()

	storageAdapter, err := storage.NewLocalStorageAdapter(filepath.Join(tempDir, "storage"), string(storage.DriverGarageLocal))
	if err != nil {
		t.Fatalf("failed creating storage adapter: %v", err)
	}

	repo := srvModule.NewRepository(database, tempDir)
	svc := srvModule.NewService(repo, nil, nil, storageAdapter, "cubit-fleet")

	t.Run("Given wrangler kv bulk get export JSON", func(t *testing.T) {
		ns, err := svc.CreateKVNamespace(ctx, "imported-cache")
		if err != nil {
			t.Fatalf("failed creating kv namespace: %v", err)
		}

		b64Secret := base64.StdEncoding.EncodeToString([]byte("binary-secret-data"))
		kvData := `[
			{"key": "user:1", "value": "Alice", "expiration_ttl": 3600, "metadata": "{\"role\":\"admin\"}"},
			{"key": "user:2", "value": "Bob", "metadata": "{\"role\":\"member\"}"},
			{"key": "secret:token", "value": "` + b64Secret + `", "base64": true}
		]`

		t.Run("When importing bulk KV entries", func(t *testing.T) {
			res, err := svc.ImportKVBulk(ctx, ns.ID, []byte(kvData))

			t.Run("Then all entries are inserted successfully", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error importing KV, got: %v", err)
				}
				if res.Imported != 3 {
					t.Errorf("expected 3 imported entries, got %d", res.Imported)
				}
				if res.NamespaceID != ns.ID {
					t.Errorf("expected namespace id %s, got %s", ns.ID, res.NamespaceID)
				}

				// Verify Alice
				alice, err := svc.GetKVPair(ctx, ns.ID, "user:1")
				if err != nil {
					t.Fatalf("failed getting user:1: %v", err)
				}
				if alice.Value != "Alice" {
					t.Errorf("expected Alice, got %s", alice.Value)
				}

				// Verify base64 decoded secret
				secret, err := svc.GetKVPair(ctx, ns.ID, "secret:token")
				if err != nil {
					t.Fatalf("failed getting secret:token: %v", err)
				}
				if secret.Value != "binary-secret-data" {
					t.Errorf("expected decoded binary-secret-data, got %s", secret.Value)
				}
			})
		})
	})

	t.Run("Given wrangler d1 export SQL dump", func(t *testing.T) {
		d1, err := svc.CreateD1Database(ctx, "prod-customers")
		if err != nil {
			t.Fatalf("failed creating d1 database: %v", err)
		}

		sqlDump := `
-- Cloudflare D1 Export Dump
CREATE TABLE IF NOT EXISTS customers (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT UNIQUE NOT NULL
);

INSERT INTO customers (id, name, email) VALUES (1, 'Alice Smith', 'alice@example.com');
INSERT INTO customers (id, name, email) VALUES (2, 'Bob Jones', 'bob@example.com');
INSERT INTO customers (id, name, email) VALUES (3, 'Charlie Brown', 'charlie@example.com');
`

		t.Run("When executing D1 import", func(t *testing.T) {
			res, err := svc.ImportD1SQL(ctx, d1.ID, sqlDump)

			t.Run("Then schema is created and records are queryable", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error importing D1 SQL, got: %v", err)
				}
				if res.Executed < 3 {
					t.Errorf("expected at least 3 statements executed, got %d", res.Executed)
				}

				// Query imported data
				queryRes, err := svc.ExecuteD1Query(ctx, d1.ID, "SELECT COUNT(*) as count FROM customers;")
				if err != nil {
					t.Fatalf("failed querying customers: %v", err)
				}
				if len(queryRes.Rows) == 0 {
					t.Fatalf("expected query results")
				}
				countVal := queryRes.Rows[0]["count"]
				if countVal != int64(3) && countVal != float64(3) && countVal != 3 {
					t.Errorf("expected count 3, got %v (%T)", countVal, countVal)
				}
			})
		})
	})

	t.Run("Given batch R2 objects for bucket import", func(t *testing.T) {
		b64Img := base64.StdEncoding.EncodeToString([]byte("fake-png-bytes"))
		importReq := &srvModule.R2ImportRequest{
			Bucket: "assets-bucket",
			Objects: []srvModule.R2ImportObject{
				{
					Key:         "images/banner.png",
					Content:     b64Img,
					Base64:      true,
					ContentType: "image/png",
				},
				{
					Key:         "manifest.json",
					Content:     `{"version":"1.0.0"}`,
					ContentType: "application/json",
				},
			},
		}

		t.Run("When importing R2 objects", func(t *testing.T) {
			res, err := svc.ImportR2Objects(ctx, "assets-bucket", importReq.Objects)

			t.Run("Then bucket is created and objects are stored in storage", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error importing R2 objects: %v", err)
				}
				if res.Imported != 2 {
					t.Errorf("expected 2 imported objects, got %d", res.Imported)
				}

				// Verify manifest.json
				data, err := svc.GetR2Object(ctx, "assets-bucket", "manifest.json")
				if err != nil {
					t.Fatalf("failed getting R2 object: %v", err)
				}
				if string(data) != `{"version":"1.0.0"}` {
					t.Errorf("expected json content, got %s", string(data))
				}

				// Verify base64 decoded banner.png
				imgData, err := svc.GetR2Object(ctx, "assets-bucket", "images/banner.png")
				if err != nil {
					t.Fatalf("failed getting banner.png: %v", err)
				}
				if string(imgData) != "fake-png-bytes" {
					t.Errorf("expected decoded image bytes, got %s", string(imgData))
				}
			})
		})
	})

	t.Run("Given file-based CLI migration helpers", func(t *testing.T) {
		t.Run("When importing KV from file", func(t *testing.T) {
			kvFilePath := filepath.Join(tempDir, "kv_export.json")
			_ = os.WriteFile(kvFilePath, []byte(`[{"key":"file-key","value":"file-val"}]`), 0644)

			res, err := svc.ImportKVFromFile(ctx, "file-ns", kvFilePath)
			t.Run("Then imports file contents successfully", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res.Imported != 1 {
					t.Errorf("expected 1 imported, got %d", res.Imported)
				}
			})
		})

		t.Run("When importing D1 from SQL file", func(t *testing.T) {
			sqlFilePath := filepath.Join(tempDir, "d1_export.sql")
			_ = os.WriteFile(sqlFilePath, []byte(`CREATE TABLE file_t (id INT); INSERT INTO file_t VALUES (42);`), 0644)

			res, err := svc.ImportD1FromFile(ctx, "file-db", sqlFilePath)
			t.Run("Then executes SQL file successfully", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res.Executed < 1 {
					t.Errorf("expected executed > 0, got %d", res.Executed)
				}
			})
		})

		t.Run("When importing R2 from directory", func(t *testing.T) {
			uploadDir := filepath.Join(tempDir, "r2_upload_dir")
			_ = os.MkdirAll(filepath.Join(uploadDir, "nested"), 0755)
			_ = os.WriteFile(filepath.Join(uploadDir, "file1.txt"), []byte("content1"), 0644)
			_ = os.WriteFile(filepath.Join(uploadDir, "nested", "file2.txt"), []byte("content2"), 0644)

			res, err := svc.ImportR2FromDirectory(ctx, "dir-bucket", uploadDir)
			t.Run("Then imports all files from directory recursively", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res.Imported != 2 {
					t.Errorf("expected 2 imported, got %d", res.Imported)
				}
			})
		})
	})

	t.Run("Given error conditions during import", func(t *testing.T) {
		t.Run("When importing invalid base64 in KV", func(t *testing.T) {
			badKV := `[{"key":"bad","value":"not-valid-base64!","base64":true}]`
			_, err := svc.ImportKVBulk(ctx, "err-ns", []byte(badKV))
			t.Run("Then returns validation error", func(t *testing.T) {
				if err == nil {
					t.Errorf("expected error for invalid base64, got nil")
				}
			})
		})

		t.Run("When importing invalid base64 in R2", func(t *testing.T) {
			badR2 := []srvModule.R2ImportObject{
				{Key: "bad.txt", Content: "not-valid-base64!", Base64: true},
			}
			_, err := svc.ImportR2Objects(ctx, "err-bucket", badR2)
			t.Run("Then returns validation error", func(t *testing.T) {
				if err == nil {
					t.Errorf("expected error for invalid base64, got nil")
				}
			})
		})

		t.Run("When importing R2 with empty bucket name", func(t *testing.T) {
			_, err := svc.ImportR2Objects(ctx, "  ", []srvModule.R2ImportObject{})
			t.Run("Then returns validation error", func(t *testing.T) {
				if err == nil {
					t.Errorf("expected error for empty bucket name, got nil")
				}
			})
		})
	})
}
