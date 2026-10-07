package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/ishaf/cubit/internal/infrastructure/db"
)

func TestOpenSQLite_AuthSchema(t *testing.T) {
	t.Run("Given a temporary SQLite database", func(t *testing.T) {
		tempDir := t.TempDir()
		dbPath := filepath.Join(tempDir, "test_auth.db")
		dsn := "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

		t.Run("When OpenSQLite is initialized then all auth tables and indexes exist", func(t *testing.T) {
			database, err := db.OpenSQLite(dsn)
			if err != nil {
				t.Fatalf("OpenSQLite failed: %v", err)
			}
			defer database.Close()

			requiredTables := []string{
				"roles",
				"users",
				"refresh_tokens",
				"api_tokens",
			}

			for _, table := range requiredTables {
				var count int
				err := database.QueryRowContext(context.Background(),
					"SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&count)
				if err != nil {
					t.Fatalf("failed to query sqlite_master for table %s: %v", table, err)
				}
				if count != 1 {
					t.Errorf("expected table %s to exist in database, count = %d", table, count)
				}
			}

			requiredIndexes := []string{
				"idx_users_email",
				"idx_users_role_id",
				"idx_refresh_tokens_user_id",
				"idx_refresh_tokens_hash",
				"idx_api_tokens_user_id",
				"idx_api_tokens_hash",
			}

			for _, idx := range requiredIndexes {
				var count int
				err := database.QueryRowContext(context.Background(),
					"SELECT count(*) FROM sqlite_master WHERE type='index' AND name=?", idx).Scan(&count)
				if err != nil {
					t.Fatalf("failed to query sqlite_master for index %s: %v", idx, err)
				}
				if count != 1 {
					t.Errorf("expected index %s to exist in database, count = %d", idx, count)
				}
			}
		})
	})
}
