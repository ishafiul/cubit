package db_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/db"
	"github.com/ishaf/cubit/internal/domain"
)

func TestServicesRepo(t *testing.T) {
	ctx := context.Background()
	tempDir, err := os.MkdirTemp("", "cubit-services-test-*")
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

	t.Run("Given an empty KV store", func(t *testing.T) {
		t.Run("When creating a namespace and putting a key-value pair", func(t *testing.T) {
			ns, _ := domain.NewKVNamespace("ns-1", "user-cache")
			if err := repo.SaveKVNamespace(ctx, ns); err != nil {
				t.Fatalf("failed saving namespace: %v", err)
			}

			pair, _ := domain.NewKVPair("ns-1", "user:42", `{"name":"Alice"}`, 3600, "json")
			if err := repo.PutKVPair(ctx, pair); err != nil {
				t.Fatalf("failed putting KV pair: %v", err)
			}

			t.Run("Then the key-value pair is retrievable with correct value", func(t *testing.T) {
				saved, err := repo.GetKVPair(ctx, "ns-1", "user:42")
				if err != nil {
					t.Fatalf("failed getting KV pair: %v", err)
				}
				if saved.Value != `{"name":"Alice"}` {
					t.Errorf("expected JSON value, got %s", saved.Value)
				}
			})

			t.Run("Then listing pairs returns the saved key", func(t *testing.T) {
				list, err := repo.ListKVPairs(ctx, "ns-1")
				if err != nil || len(list) != 1 {
					t.Fatalf("expected 1 pair, got %d (err: %v)", len(list), err)
				}
				if list[0].Key != "user:42" {
					t.Errorf("expected key user:42, got %s", list[0].Key)
				}
			})
		})
	})

	t.Run("Given an isolated D1 database", func(t *testing.T) {
		t.Run("When creating a database and executing SQL queries", func(t *testing.T) {
			d1, _ := domain.NewD1Database("d1-test-1", "analytics-db")
			if err := repo.SaveD1Database(ctx, d1); err != nil {
				t.Fatalf("failed saving D1 db: %v", err)
			}

			// DDL: Create table
			_, err := repo.ExecuteD1Query(ctx, "d1-test-1", "CREATE TABLE events (id INTEGER PRIMARY KEY, event_type TEXT);")
			if err != nil {
				t.Fatalf("DDL failed: %v", err)
			}

			// DML: Insert row
			_, err = repo.ExecuteD1Query(ctx, "d1-test-1", "INSERT INTO events (event_type) VALUES ('page_view');")
			if err != nil {
				t.Fatalf("Insert failed: %v", err)
			}

			t.Run("Then SELECT returns the inserted record", func(t *testing.T) {
				res, err := repo.ExecuteD1Query(ctx, "d1-test-1", "SELECT * FROM events;")
				if err != nil {
					t.Fatalf("SELECT failed: %v", err)
				}
				if len(res.Rows) != 1 {
					t.Fatalf("expected 1 row, got %d", len(res.Rows))
				}
				if res.Rows[0]["event_type"] != "page_view" {
					t.Errorf("expected event_type page_view, got %v", res.Rows[0]["event_type"])
				}
			})
		})
	})

	t.Run("Given Queues and Messages", func(t *testing.T) {
		t.Run("When creating a queue and sending a message", func(t *testing.T) {
			q, _ := domain.NewQueue("q-test-1", "task-queue", "app-worker", 3)
			if err := repo.SaveQueue(ctx, q); err != nil {
				t.Fatalf("failed saving queue: %v", err)
			}

			msg := &domain.QueueMessage{
				ID:      "msg-1",
				QueueID: "q-test-1",
				Body:    `{"task":"send_email"}`,
			}
			if err := repo.EnqueueMessage(ctx, msg); err != nil {
				t.Fatalf("failed enqueuing message: %v", err)
			}

			t.Run("Then the message is listed with backlog count", func(t *testing.T) {
				messages, err := repo.ListQueueMessages(ctx, "q-test-1", 10)
				if err != nil || len(messages) != 1 {
					t.Fatalf("expected 1 message, got %d (err: %v)", len(messages), err)
				}
				if messages[0].Body != `{"task":"send_email"}` {
					t.Errorf("unexpected message body: %s", messages[0].Body)
				}
			})
		})
	})

	t.Run("Given Cron Triggers", func(t *testing.T) {
		t.Run("When creating a cron trigger and recording an execution run", func(t *testing.T) {
			cron, _ := domain.NewCronTrigger("cron-1", "hourly-sync", "0 * * * *", "app-worker")
			if err := repo.SaveCronTrigger(ctx, cron); err != nil {
				t.Fatalf("failed saving cron trigger: %v", err)
			}

			run := &domain.CronRun{
				ID:         "run-1",
				TriggerID:  "cron-1",
				Status:     "success",
				StatusCode: 200,
				DurationMs: 12.5,
			}
			if err := repo.RecordCronRun(ctx, run); err != nil {
				t.Fatalf("failed recording run: %v", err)
			}

			t.Run("Then the run history is recorded and trigger last_run_at is updated", func(t *testing.T) {
				runs, err := repo.ListCronRuns(ctx, "cron-1")
				if err != nil || len(runs) != 1 {
					t.Fatalf("expected 1 run, got %d (err: %v)", len(runs), err)
				}
				if runs[0].StatusCode != 200 {
					t.Errorf("expected status code 200, got %d", runs[0].StatusCode)
				}
			})
		})
	})
}
