package domain_test

import (
	"context"
	"database/sql"
	"testing"

	coreDomain "github.com/ishaf/cubit/internal/domain"
	domModule "github.com/ishaf/cubit/internal/modules/domain"
	_ "modernc.org/sqlite"
)

func TestSQLiteRepository(t *testing.T) {
	ctx := context.Background()
	db := setupTestDB(t)
	defer db.Close()

	repo := domModule.NewRepository(db)

	t.Run("Given a domain entity", func(t *testing.T) {
		dom, err := coreDomain.NewDomain("dom-1", "app-1", "api.example.com", "/v1")
		if err != nil {
			t.Fatalf("unexpected error creating domain: %v", err)
		}

		t.Run("When saving the domain then it can be retrieved by ID and by hostname", func(t *testing.T) {
			if err := repo.Save(ctx, dom); err != nil {
				t.Fatalf("expected no error saving domain, got: %v", err)
			}

			byID, err := repo.GetByID(ctx, "dom-1")
			if err != nil {
				t.Fatalf("expected domain by ID, got: %v", err)
			}
			if byID.Hostname != "api.example.com" || byID.PathPrefix != "/v1" {
				t.Errorf("unexpected domain retrieved by ID: %+v", byID)
			}

			byHost, err := repo.GetByHostname(ctx, "api.example.com")
			if err != nil {
				t.Fatalf("expected domain by hostname, got: %v", err)
			}
			if byHost.ID != "dom-1" {
				t.Errorf("expected ID dom-1, got %s", byHost.ID)
			}
		})

		t.Run("When querying a non-existent hostname", func(t *testing.T) {
			_, err := repo.GetByHostname(ctx, "missing.domain.com")
			if err == nil {
				t.Fatal("expected error for non-existent hostname, got nil")
			}
		})

		t.Run("When saving a wildcard hostname", func(t *testing.T) {
			wildDom, err := coreDomain.NewDomain("dom-wild", "app-1", "*.example.com", "/")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if err := repo.Save(ctx, wildDom); err != nil {
				t.Fatalf("expected no error saving wildcard domain, got: %v", err)
			}

			retrieved, err := repo.GetByHostname(ctx, "*.example.com")
			if err != nil {
				t.Fatalf("expected retrieval of wildcard hostname, got: %v", err)
			}
			if retrieved.Hostname != "*.example.com" {
				t.Errorf("expected *.example.com, got %s", retrieved.Hostname)
			}
		})

		t.Run("When saving multiple paths under the same hostname", func(t *testing.T) {
			pathDom1, _ := coreDomain.NewDomain("dom-p1", "app-1", "service.example.com", "/v1")
			pathDom2, _ := coreDomain.NewDomain("dom-p2", "app-1", "service.example.com", "/v2")

			if err := repo.Save(ctx, pathDom1); err != nil {
				t.Fatalf("expected no error saving path 1: %v", err)
			}
			if err := repo.Save(ctx, pathDom2); err != nil {
				t.Fatalf("expected no error saving path 2: %v", err)
			}

			// Test GetByHostAndPath
			got1, err := repo.GetByHostAndPath(ctx, "service.example.com", "/v1")
			if err != nil || got1.ID != "dom-p1" {
				t.Errorf("expected dom-p1, got %+v (err: %v)", got1, err)
			}
			got2, err := repo.GetByHostAndPath(ctx, "service.example.com", "/v2")
			if err != nil || got2.ID != "dom-p2" {
				t.Errorf("expected dom-p2, got %+v (err: %v)", got2, err)
			}

			// Test ListByHostname
			list, err := repo.ListByHostname(ctx, "service.example.com")
			if err != nil {
				t.Fatalf("expected no error listing by hostname: %v", err)
			}
			if len(list) != 2 {
				t.Errorf("expected 2 routes for service.example.com, got %d", len(list))
			}
		})
	})
}

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open test sqlite db: %v", err)
	}

	schema := `
	CREATE TABLE applications (
		id TEXT PRIMARY KEY,
		name TEXT UNIQUE NOT NULL
	);
	CREATE TABLE domains (
		id TEXT PRIMARY KEY,
		application_id TEXT NOT NULL,
		hostname TEXT NOT NULL,
		path_prefix TEXT NOT NULL DEFAULT '/',
		ssl_active INTEGER NOT NULL DEFAULT 0,
		created_at TIMESTAMP NOT NULL,
		updated_at TIMESTAMP NOT NULL,
		UNIQUE(hostname, path_prefix),
		FOREIGN KEY(application_id) REFERENCES applications(id) ON DELETE CASCADE
	);
	`
	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("failed creating schema: %v", err)
	}

	return db
}
