package domain_test

import (
	"testing"

	"github.com/ishaf/cubit/internal/domain"
)

func TestCelldServicesDomain(t *testing.T) {
	t.Run("Given KV namespace parameters", func(t *testing.T) {
		t.Run("When name is empty then returns validation error", func(t *testing.T) {
			_, err := domain.NewKVNamespace("kv-1", "   ")
			if err == nil {
				t.Error("expected error for empty KV namespace name")
			}
		})

		t.Run("When valid parameters provided then initializes KV namespace", func(t *testing.T) {
			ns, err := domain.NewKVNamespace("kv-1", "cache-ns")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ns.Name != "cache-ns" {
				t.Errorf("expected name cache-ns, got %s", ns.Name)
			}
		})
	})

	t.Run("Given D1 database parameters", func(t *testing.T) {
		t.Run("When name is empty then returns validation error", func(t *testing.T) {
			_, err := domain.NewD1Database("d1-1", "")
			if err == nil {
				t.Error("expected error for empty D1 database name")
			}
		})

		t.Run("When valid parameters provided then initializes D1 database", func(t *testing.T) {
			db, err := domain.NewD1Database("d1-1", "prod-users")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if db.Name != "prod-users" {
				t.Errorf("expected name prod-users, got %s", db.Name)
			}
		})
	})

	t.Run("Given Queue parameters", func(t *testing.T) {
		t.Run("When name is empty then returns validation error", func(t *testing.T) {
			_, err := domain.NewQueue("q-1", "", "app-1", 3)
			if err == nil {
				t.Error("expected error for empty queue name")
			}
		})

		t.Run("When valid parameters provided then initializes Queue", func(t *testing.T) {
			q, err := domain.NewQueue("q-1", "notifications-queue", "app-1", 5)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if q.MaxRetries != 5 {
				t.Errorf("expected maxRetries 5, got %d", q.MaxRetries)
			}
		})
	})

	t.Run("Given Cron Trigger parameters", func(t *testing.T) {
		t.Run("When expression or target is empty then returns validation error", func(t *testing.T) {
			_, err := domain.NewCronTrigger("c-1", "nightly", "", "app-1")
			if err == nil {
				t.Error("expected error for empty cron expression")
			}
		})

		t.Run("When valid parameters provided then initializes CronTrigger", func(t *testing.T) {
			trigger, err := domain.NewCronTrigger("c-1", "nightly-cleanup", "0 0 * * *", "app-1")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if trigger.Status != "active" {
				t.Errorf("expected active status, got %s", trigger.Status)
			}
		})
	})

	t.Run("Given Durable Object class parameters", func(t *testing.T) {
		t.Run("When valid parameters provided then initializes DurableObjectClass with facets", func(t *testing.T) {
			doc, err := domain.NewDurableObjectClass("do-1", "UserSession", "app-1", nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(doc.Facets) == 0 {
				t.Error("expected default facet to be initialized")
			}
			if doc.StorageBackend != "sqlite" {
				t.Errorf("expected storage backend sqlite, got %s", doc.StorageBackend)
			}
		})
	})

	t.Run("Given Container workload parameters", func(t *testing.T) {
		t.Run("When valid parameters provided then initializes ContainerWorkload", func(t *testing.T) {
			cw, err := domain.NewContainerWorkload("ct-1", "redis-sidecar", "redis:7-alpine", 6379, nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cw.Port != 6379 {
				t.Errorf("expected port 6379, got %d", cw.Port)
			}
			if cw.Status != "running" {
				t.Errorf("expected running status, got %s", cw.Status)
			}
		})
	})

	t.Run("Given Static site parameters", func(t *testing.T) {
		t.Run("When valid parameters provided then initializes StaticSite with default SPA routing", func(t *testing.T) {
			site, err := domain.NewStaticSite("st-1", "landing-page", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !site.SPARouting {
				t.Error("expected SPARouting to default to true")
			}
			if site.IndexDocument != "index.html" {
				t.Errorf("expected index.html, got %s", site.IndexDocument)
			}
		})
	})
}
