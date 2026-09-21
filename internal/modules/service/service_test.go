package service_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishaf/cubit/internal/infrastructure/db"
	srvModule "github.com/ishaf/cubit/internal/modules/service"
)

func TestServicesService(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "cubit-srv-test-*")
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

	repo := srvModule.NewRepository(database, tempDir)
	svc := srvModule.NewService(repo, nil, nil, nil, "cubit-fleet")

	t.Run("Given KV namespace operations", func(t *testing.T) {
		t.Run("When creating a namespace and putting a value", func(t *testing.T) {
			ns, err := svc.CreateKVNamespace(ctx, "session-tokens")
			if err != nil {
				t.Fatalf("failed creating namespace: %v", err)
			}

			pair, err := svc.PutKVPair(ctx, ns.ID, "sess_123", "user_admin", 3600, "")
			if err != nil {
				t.Fatalf("failed putting KV pair: %v", err)
			}

			t.Run("Then the stored key-value pair is retrievable", func(t *testing.T) {
				saved, err := svc.GetKVPair(ctx, ns.ID, pair.Key)
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
			d1, err := svc.CreateD1Database(ctx, "billing-db")
			if err != nil {
				t.Fatalf("failed creating D1 db: %v", err)
			}

			_, err = svc.ExecuteD1Query(ctx, d1.ID, "CREATE TABLE invoices (id TEXT PRIMARY KEY, amount REAL)")
			if err != nil {
				t.Fatalf("failed executing DDL: %v", err)
			}

			_, err = svc.ExecuteD1Query(ctx, d1.ID, "INSERT INTO invoices (id, amount) VALUES ('inv_1', 99.50)")
			if err != nil {
				t.Fatalf("failed inserting record: %v", err)
			}

			res, err := svc.ExecuteD1Query(ctx, d1.ID, "SELECT id, amount FROM invoices")
			if err != nil {
				t.Fatalf("failed query: %v", err)
			}

			t.Run("Then rows and columns are correctly returned", func(t *testing.T) {
				if len(res.Rows) != 1 {
					t.Fatalf("expected 1 row, got %d", len(res.Rows))
				}
				if res.Rows[0]["id"] != "inv_1" {
					t.Errorf("expected inv_1, got %v", res.Rows[0]["id"])
				}
			})
		})
	})

	t.Run("Given message queue operations", func(t *testing.T) {
		t.Run("When sending messages to a queue", func(t *testing.T) {
			q, err := svc.CreateQueue(ctx, "email-notifications", "", 3)
			if err != nil {
				t.Fatalf("failed creating queue: %v", err)
			}

			msg, err := svc.SendQueueMessage(ctx, q.ID, "{\"to\":\"user@example.com\"}")
			if err != nil {
				t.Fatalf("failed sending message: %v", err)
			}

			t.Run("Then message is listed in the queue backlog", func(t *testing.T) {
				messages, err := svc.ListQueueMessages(ctx, q.ID, 10)
				if err != nil {
					t.Fatalf("failed listing queue messages: %v", err)
				}
				if len(messages) != 1 {
					t.Fatalf("expected 1 message, got %d", len(messages))
				}
				if messages[0].ID != msg.ID {
					t.Errorf("expected id %s, got %s", msg.ID, messages[0].ID)
				}
			})
		})
	})

	t.Run("Given R2 bucket operations", func(t *testing.T) {
		t.Run("When creating a custom R2 bucket", func(t *testing.T) {
			b, err := svc.CreateR2Bucket(ctx, "user-media-assets")
			if err != nil {
				t.Fatalf("failed creating R2 bucket: %v", err)
			}
			if b.Name != "user-media-assets" {
				t.Errorf("expected bucket name user-media-assets, got %s", b.Name)
			}

			t.Run("Then it is listed alongside the protected fleet bucket", func(t *testing.T) {
				buckets, err := svc.ListR2Buckets(ctx)
				if err != nil {
					t.Fatalf("failed listing R2 buckets: %v", err)
				}
				if len(buckets) < 2 {
					t.Fatalf("expected at least 2 buckets, got %d", len(buckets))
				}

				var foundFleet, foundUser bool
				for _, bk := range buckets {
					if bk.Name == "cubit-fleet" && bk.IsSystem {
						foundFleet = true
					}
					if bk.Name == "user-media-assets" && !bk.IsSystem {
						foundUser = true
					}
				}
				if !foundFleet {
					t.Errorf("expected protected system bucket cubit-fleet")
				}
				if !foundUser {
					t.Errorf("expected user bucket user-media-assets")
				}
			})

			t.Run("Then uploading to system fleet bucket is forbidden", func(t *testing.T) {
				uploadErr := svc.UploadR2Object(ctx, "cubit-fleet", "test.txt", []byte("hello"))
				if uploadErr == nil {
					t.Fatalf("expected upload error on system bucket, got nil")
				}
			})

			t.Run("Then deleting system fleet bucket is forbidden", func(t *testing.T) {
				delErr := svc.DeleteR2Bucket(ctx, "cubit-fleet")
				if delErr == nil {
					t.Fatalf("expected delete error on system bucket, got nil")
				}
			})

			t.Run("Then deleting custom user bucket succeeds", func(t *testing.T) {
				if err := svc.DeleteR2Bucket(ctx, "user-media-assets"); err != nil {
					t.Fatalf("failed deleting user bucket: %v", err)
				}
			})
		})
	})
}
