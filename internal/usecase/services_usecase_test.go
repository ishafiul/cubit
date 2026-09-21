package usecase_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/db"
	"github.com/ishaf/cubit/internal/usecase"
)

func TestServicesUsecase(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "cubit-uc-services-test-*")
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

	repo := db.NewServicesRepo(database, tempDir)
	appRepo := db.NewAppRepo(database)
	uc := usecase.NewServicesUsecase(repo, appRepo, nil, nil, "cubit-fleet")

	t.Run("Given KV namespace operations", func(t *testing.T) {
		t.Run("When creating a namespace and putting a value", func(t *testing.T) {
			ns, err := uc.CreateKVNamespace(ctx, "session-tokens")
			if err != nil {
				t.Fatalf("failed creating namespace: %v", err)
			}

			pair, err := uc.PutKVPair(ctx, ns.ID, "sess_123", "user_admin", 3600, "")
			if err != nil {
				t.Fatalf("failed putting KV pair: %v", err)
			}

			t.Run("Then the stored key-value pair is retrievable", func(t *testing.T) {
				saved, err := uc.GetKVPair(ctx, ns.ID, pair.Key)
				if err != nil {
					t.Fatalf("failed getting KV pair: %v", err)
				}
				if saved.Value != "user_admin" {
					t.Errorf("expected value user_admin, got %s", saved.Value)
				}
			})
		})
	})

	t.Run("Given D1 database operations", func(t *testing.T) {
		t.Run("When creating a D1 database and executing SQL statements", func(t *testing.T) {
			d1, err := uc.CreateD1Database(ctx, "billing-db")
			if err != nil {
				t.Fatalf("failed creating D1 db: %v", err)
			}

			// Create table & insert
			_, err = uc.ExecuteD1Query(ctx, d1.ID, "CREATE TABLE invoices (id INT, amount INT);")
			if err != nil {
				t.Fatalf("table creation failed: %v", err)
			}
			_, err = uc.ExecuteD1Query(ctx, d1.ID, "INSERT INTO invoices VALUES (1, 100);")
			if err != nil {
				t.Fatalf("insert failed: %v", err)
			}

			t.Run("Then SELECT query returns the inserted data", func(t *testing.T) {
				res, err := uc.ExecuteD1Query(ctx, d1.ID, "SELECT * FROM invoices;")
				if err != nil {
					t.Fatalf("query failed: %v", err)
				}
				if len(res.Rows) != 1 {
					t.Fatalf("expected 1 row, got %d", len(res.Rows))
				}
			})
		})
	})

	t.Run("Given Dynamic Worker evaluation", func(t *testing.T) {
		t.Run("When executing a JavaScript worker snippet dynamically", func(t *testing.T) {
			code := `export default { fetch: () => new Response("dynamic worker ok") };`
			status, _, body, err := uc.EvalDynamicWorker(ctx, code, "GET", "/hello", nil, nil)

			t.Run("Then it executes successfully and returns 200 with response body", func(t *testing.T) {
				if err != nil {
					t.Fatalf("dynamic worker execution failed: %v", err)
				}
				if status != 200 {
					t.Errorf("expected status 200, got %d", status)
				}
				if string(body) != "dynamic worker ok" {
					t.Errorf("expected response body 'dynamic worker ok', got %q", string(body))
				}
			})
		})
	})
}
