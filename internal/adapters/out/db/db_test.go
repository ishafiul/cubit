package db_test

import (
	"context"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/db"
	"github.com/ishaf/cubit/internal/domain"
)

func TestSQLiteRepositories(t *testing.T) {
	ctx := context.Background()

	// In-memory SQLite for test
	database, err := db.OpenSQLite("file::memory:?cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed to open in-memory test db: %v", err)
	}
	defer database.Close()

	nodeRepo := db.NewNodeRepo(database)
	appRepo := db.NewAppRepo(database)
	depRepo := db.NewDeploymentRepo(database)
	domRepo := db.NewDomainRepo(database)

	t.Run("Given an empty SQLite database", func(t *testing.T) {
		t.Run("When saving and fetching a node then data is accurately persisted", func(t *testing.T) {
			node, _ := domain.NewNode("n-100", "node-paris", "10.0.0.5", 8081, 8080, "0.2.0")
			if err := nodeRepo.Save(ctx, node); err != nil {
				t.Fatalf("expected save to succeed: %v", err)
			}

			fetched, err := nodeRepo.GetByID(ctx, "n-100")
			if err != nil {
				t.Fatalf("expected get to succeed: %v", err)
			}
			if fetched.Name != "node-paris" {
				t.Errorf("expected name node-paris, got %s", fetched.Name)
			}
		})

		t.Run("When saving an application and deployment", func(t *testing.T) {
			app, _ := domain.NewApplication("a-100", "my-app", "https://github.com/my/app", "main", nil, nil)
			if err := appRepo.Save(ctx, app); err != nil {
				t.Fatalf("failed saving app: %v", err)
			}

			dep, _ := domain.NewDeployment("d-100", "a-100", "commit-abc", "Initial commit")
			if err := depRepo.Save(ctx, dep); err != nil {
				t.Fatalf("failed saving dep: %v", err)
			}

			t.Run("Then deployment logs can be appended and retrieved", func(t *testing.T) {
				_ = depRepo.AppendLog(ctx, dep.ID, domain.DeploymentLog{
					Step:    domain.LogStepEsbuild,
					Message: "Compiled bundle successfully",
					Level:   domain.LogLevelInfo,
				})

				logs, err := depRepo.GetLogs(ctx, dep.ID)
				if err != nil {
					t.Fatalf("failed fetching logs: %v", err)
				}
				if len(logs) != 1 {
					t.Fatalf("expected 1 log, got %d", len(logs))
				}
				if logs[0].Step != domain.LogStepEsbuild {
					t.Errorf("expected esbuild step, got %s", logs[0].Step)
				}
			})
		})

		t.Run("When creating and binding a domain", func(t *testing.T) {
			dom, _ := domain.NewDomain("dom-100", "a-100", "app.example.com", "/api")
			if err := domRepo.Save(ctx, dom); err != nil {
				t.Fatalf("failed saving domain: %v", err)
			}

			t.Run("Then it is listed by application ID", func(t *testing.T) {
				domains, err := domRepo.ListByAppID(ctx, "a-100")
				if err != nil {
					t.Fatalf("failed listing domains: %v", err)
				}
				if len(domains) != 1 {
					t.Fatalf("expected 1 domain, got %d", len(domains))
				}
				if domains[0].Hostname != "app.example.com" {
					t.Errorf("expected hostname app.example.com, got %s", domains[0].Hostname)
				}
			})
		})

		t.Run("When saving an inline application", func(t *testing.T) {
			customCode := `export default { fetch: () => new Response("hello world") };`
			inlineApp, err := domain.NewApplicationWithSource("a-inline", "inline-app", domain.SourceTypeInline, "", "", customCode, nil, nil)
			if err != nil {
				t.Fatalf("failed creating inline app: %v", err)
			}

			if err := appRepo.Save(ctx, inlineApp); err != nil {
				t.Fatalf("failed saving inline app: %v", err)
			}

			t.Run("Then it is retrieved with SourceTypeInline and inline code intact", func(t *testing.T) {
				fetched, err := appRepo.GetByID(ctx, "a-inline")
				if err != nil {
					t.Fatalf("failed retrieving inline app: %v", err)
				}
				if fetched.SourceType != domain.SourceTypeInline {
					t.Errorf("expected SourceTypeInline, got %s", fetched.SourceType)
				}
				if fetched.InlineCode != customCode {
					t.Errorf("expected custom code, got %s", fetched.InlineCode)
				}
				if fetched.GitRepo != "" {
					t.Errorf("expected empty git repo, got %s", fetched.GitRepo)
				}
				if fetched.Subdomain != "inline-app" {
					t.Errorf("expected subdomain 'inline-app', got %s", fetched.Subdomain)
				}
			})

			t.Run("Then it is retrieved via GetBySubdomain", func(t *testing.T) {
				bySub, err := appRepo.GetBySubdomain(ctx, "inline-app")
				if err != nil {
					t.Fatalf("failed retrieving app by subdomain: %v", err)
				}
				if bySub.ID != "a-inline" {
					t.Errorf("expected app id a-inline, got %s", bySub.ID)
				}
			})
		})

		t.Run("When saving multiple deployments with sequential build versions", func(t *testing.T) {
			dep1, _ := domain.NewDeploymentWithVersion("d-ver-1", "a-100", "hash-1", "Build 1", 1)
			dep2, _ := domain.NewDeploymentWithVersion("d-ver-2", "a-100", "hash-2", "Build 2", 2)
			_ = depRepo.Save(ctx, dep1)
			_ = depRepo.Save(ctx, dep2)

			t.Run("Then GetLatestBuildVersion returns 2", func(t *testing.T) {
				latest, err := depRepo.GetLatestBuildVersion(ctx, "a-100")
				if err != nil {
					t.Fatalf("failed getting latest build version: %v", err)
				}
				if latest != 2 {
					t.Errorf("expected latest version 2, got %d", latest)
				}
			})

			t.Run("Then ListByAppID preserves build versions ordered descending", func(t *testing.T) {
				deps, err := depRepo.ListByAppID(ctx, "a-100")
				if err != nil {
					t.Fatalf("failed listing deployments: %v", err)
				}
				if len(deps) < 2 {
					t.Fatalf("expected at least 2 deployments, got %d", len(deps))
				}
				if deps[0].BuildVersion != 2 {
					t.Errorf("expected first deployment to be build version 2, got %d", deps[0].BuildVersion)
				}
			})
		})
	})
}
