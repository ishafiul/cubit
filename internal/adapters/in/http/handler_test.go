package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	http_adapter "github.com/ishaf/cubit/internal/adapters/in/http"
	"github.com/ishaf/cubit/internal/adapters/out/db"
	"github.com/ishaf/cubit/internal/adapters/out/docker"
	"github.com/ishaf/cubit/internal/adapters/out/storage"
	"github.com/ishaf/cubit/internal/adapters/out/traefik"
	"github.com/ishaf/cubit/internal/usecase"
)

func setupTestServer(t *testing.T) http.Handler {
	ctx := context.Background()
	tmpDB, err := db.OpenSQLite("file::memory:?cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed opening in-memory db: %v", err)
	}

	nodeRepo := db.NewNodeRepo(tmpDB)
	appRepo := db.NewAppRepo(tmpDB)
	depRepo := db.NewDeploymentRepo(tmpDB)
	domRepo := db.NewDomainRepo(tmpDB)

	tmpDir := t.TempDir()
	proxyAdapter := traefik.NewFileProvider(tmpDir+"/traefik.yaml", "letsencrypt")
	storageAdapter, _ := storage.NewLocalStorageAdapter(tmpDir+"/storage", "garage_local")
	supervisor := docker.NewCelldSupervisor("ghcr.io/denoland/celld")

	nodeUsecase := usecase.NewNodeUsecase(nodeRepo, supervisor, "s3://fleet-bucket")
	appUsecase := usecase.NewAppUsecase(appRepo, depRepo, nodeRepo, domRepo, storageAdapter, proxyAdapter, "fleet-bucket")
	domainUsecase := usecase.NewDomainUsecase(domRepo, appRepo, appUsecase)
	runtimeUsecase := usecase.NewRuntimeUsecase(nodeRepo, supervisor, storageAdapter, "s3://fleet-bucket", "0.2.0")

	// Register an initial node
	_, _ = nodeUsecase.RegisterNode(ctx, "test-node", "10.0.0.1", 8081, 8080, "0.2.0")

	apiHandler := http_adapter.NewAPIHandler(nodeUsecase, appUsecase, domainUsecase, runtimeUsecase, depRepo)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) {
		http_adapter.HandlerFromMux(apiHandler, sub)
	})
	return r
}

func TestHTTPAPI(t *testing.T) {
	router := setupTestServer(t)

	t.Run("Given the Cubit REST API", func(t *testing.T) {
		t.Run("When querying runtime status then returns 200 with current celld version", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/runtime/status", nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d", rec.Code)
			}

			var status http_adapter.RuntimeStatus
			_ = json.NewDecoder(rec.Body).Decode(&status)
			if status.CurrentCelldVersion != "0.2.0" {
				t.Errorf("expected celld version 0.2.0, got %s", status.CurrentCelldVersion)
			}
			if status.ActiveNodesCount != 1 {
				t.Errorf("expected 1 active node, got %d", status.ActiveNodesCount)
			}
		})

		t.Run("When creating and deploying an application", func(t *testing.T) {
			createBody := []byte(`{
				"name": "my-api",
				"gitRepo": "https://github.com/test/repo",
				"branch": "main"
			}`)
			createReq := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
			createReq.Header.Set("Content-Type", "application/json")
			createRec := httptest.NewRecorder()

			router.ServeHTTP(createRec, createReq)

			t.Run("Then application is created with 201 status", func(t *testing.T) {
				if createRec.Code != http.StatusCreated {
					t.Fatalf("expected 201 created, got %d: %s", createRec.Code, createRec.Body.String())
				}

				var app http_adapter.Application
				_ = json.NewDecoder(createRec.Body).Decode(&app)

				// Now deploy the app
				deployReq := httptest.NewRequest(http.MethodPost, "/api/v1/applications/"+app.Id.String()+"/deploy", nil)
				deployRec := httptest.NewRecorder()
				router.ServeHTTP(deployRec, deployReq)

				if deployRec.Code != http.StatusAccepted {
					t.Fatalf("expected 202 accepted, got %d", deployRec.Code)
				}

				var dep http_adapter.Deployment
				_ = json.NewDecoder(deployRec.Body).Decode(&dep)
				if dep.Status != http_adapter.DeploymentStatusActive {
					t.Errorf("expected active deployment, got %s", dep.Status)
				}
			})
		})

		t.Run("When creating an inline Hello World application via API", func(t *testing.T) {
			createBody := []byte(`{
				"name": "hello-world-api",
				"sourceType": "inline"
			}`)
			createReq := httptest.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(createBody))
			createReq.Header.Set("Content-Type", "application/json")
			createRec := httptest.NewRecorder()

			router.ServeHTTP(createRec, createReq)

			t.Run("Then application is created with 201 status and default Hello World template", func(t *testing.T) {
				if createRec.Code != http.StatusCreated {
					t.Fatalf("expected 201 created, got %d: %s", createRec.Code, createRec.Body.String())
				}

				var app http_adapter.Application
				_ = json.NewDecoder(createRec.Body).Decode(&app)

				if app.SourceType != http_adapter.Inline {
					t.Errorf("expected source type inline, got %s", app.SourceType)
				}
				if app.InlineCode == nil || *app.InlineCode == "" {
					t.Fatal("expected inlineCode to be populated, got nil or empty")
				}
				if app.GitRepo != nil && *app.GitRepo != "" {
					t.Errorf("expected empty git repo, got %s", *app.GitRepo)
				}

				// Deploy inline application
				deployReq := httptest.NewRequest(http.MethodPost, "/api/v1/applications/"+app.Id.String()+"/deploy", nil)
				deployRec := httptest.NewRecorder()
				router.ServeHTTP(deployRec, deployReq)

				if deployRec.Code != http.StatusAccepted {
					t.Fatalf("expected 202 accepted for inline deploy, got %d", deployRec.Code)
				}
			})
		})
	})
}
