package http_test

import (
	"bytes"
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
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

func setupServicesTestRouter(t *testing.T) http.Handler {
	tmpDB, err := db.OpenSQLite("file::memory:?cache=shared&_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatalf("failed opening in-memory db: %v", err)
	}

	tmpDir := t.TempDir()
	servicesRepo := db.NewServicesRepo(tmpDB, tmpDir+"/d1")
	appRepo := db.NewAppRepo(tmpDB)
	depRepo := db.NewDeploymentRepo(tmpDB)
	nodeRepo := db.NewNodeRepo(tmpDB)
	domRepo := db.NewDomainRepo(tmpDB)

	proxyAdapter := traefik.NewFileProvider(tmpDir+"/traefik.yaml", "letsencrypt")
	storageAdapter, _ := storage.NewLocalStorageAdapter(tmpDir+"/storage", "garage_local")
	supervisor := docker.NewCelldSupervisor("ghcr.io/denoland/celld")

	nodeUsecase := usecase.NewNodeUsecase(nodeRepo, supervisor, "s3://fleet-bucket")
	appUsecase := usecase.NewAppUsecase(appRepo, depRepo, nodeRepo, domRepo, storageAdapter, proxyAdapter, "fleet-bucket")
	domainUsecase := usecase.NewDomainUsecase(domRepo, appRepo, appUsecase)
	runtimeUsecase := usecase.NewRuntimeUsecase(nodeRepo, supervisor, storageAdapter, "s3://fleet-bucket", domain.DefaultCelldVersion)

	servicesUsecase := usecase.NewServicesUsecase(servicesRepo, appRepo, appUsecase, storageAdapter, "fleet-bucket")
	servicesHandler := http_adapter.NewServicesHandler(servicesUsecase)
	apiHandler := http_adapter.NewAPIHandler(nodeUsecase, appUsecase, domainUsecase, runtimeUsecase, depRepo)

	r := chi.NewRouter()
	r.Route("/api/v1", func(sub chi.Router) {
		http_adapter.HandlerFromMux(apiHandler, sub)
		servicesHandler.RegisterRoutes(sub)
	})
	return r
}

func TestServicesHandler(t *testing.T) {
	router := setupServicesTestRouter(t)

	t.Run("Given KV REST endpoints", func(t *testing.T) {
		var nsID string

		t.Run("When creating a KV namespace", func(t *testing.T) {
			body := []byte(`{"title": "user-sessions"}`)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/kv/namespaces", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then responds with 201 Created and valid namespace ID", func(t *testing.T) {
				if rec.Code != http.StatusCreated {
					t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
				}
				var ns domain.KVNamespace
				_ = json.NewDecoder(rec.Body).Decode(&ns)
				if ns.ID == "" || ns.Name != "user-sessions" {
					t.Errorf("unexpected namespace: %+v", ns)
				}
				nsID = ns.ID
			})
		})

		t.Run("When writing and reading a KV pair", func(t *testing.T) {
			putBody := []byte(`{"value": "active-session-token", "expiration_ttl": 3600}`)
			putReq := httptest.NewRequest(http.MethodPut, "/api/v1/kv/namespaces/"+nsID+"/values/session_123", bytes.NewReader(putBody))
			putReq.Header.Set("Content-Type", "application/json")
			putRec := httptest.NewRecorder()
			router.ServeHTTP(putRec, putReq)

			t.Run("Then PUT responds with 200 OK", func(t *testing.T) {
				if putRec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", putRec.Code, putRec.Body.String())
				}
			})

			getReq := httptest.NewRequest(http.MethodGet, "/api/v1/kv/namespaces/"+nsID+"/values/session_123", nil)
			getRec := httptest.NewRecorder()
			router.ServeHTTP(getRec, getReq)

			t.Run("Then GET retrieves the written value", func(t *testing.T) {
				if getRec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", getRec.Code, getRec.Body.String())
				}
				var pair domain.KVPair
				_ = json.NewDecoder(getRec.Body).Decode(&pair)
				if pair.Value != "active-session-token" {
					t.Errorf("expected active-session-token, got %s", pair.Value)
				}
			})
		})
	})

	t.Run("Given D1 SQL REST endpoints", func(t *testing.T) {
		var dbID string

		t.Run("When creating a D1 database", func(t *testing.T) {
			body := []byte(`{"name": "analytics-db"}`)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/d1/databases", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then responds with 201 and valid database ID", func(t *testing.T) {
				if rec.Code != http.StatusCreated {
					t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
				}
				var d1 domain.D1Database
				_ = json.NewDecoder(rec.Body).Decode(&d1)
				dbID = d1.ID
			})
		})

		t.Run("When executing DDL and SELECT queries on the D1 database", func(t *testing.T) {
			ddlBody := []byte(`{"sql": "CREATE TABLE events (id TEXT PRIMARY KEY, type TEXT);"}`)
			ddlReq := httptest.NewRequest(http.MethodPost, "/api/v1/d1/databases/"+dbID+"/query", bytes.NewReader(ddlBody))
			ddlReq.Header.Set("Content-Type", "application/json")
			ddlRec := httptest.NewRecorder()
			router.ServeHTTP(ddlRec, ddlReq)

			insertBody := []byte(`{"sql": "INSERT INTO events (id, type) VALUES ('evt_1', 'click');"}`)
			insReq := httptest.NewRequest(http.MethodPost, "/api/v1/d1/databases/"+dbID+"/query", bytes.NewReader(insertBody))
			insReq.Header.Set("Content-Type", "application/json")
			insRec := httptest.NewRecorder()
			router.ServeHTTP(insRec, insReq)

			queryBody := []byte(`{"sql": "SELECT id, type FROM events;"}`)
			qReq := httptest.NewRequest(http.MethodPost, "/api/v1/d1/databases/"+dbID+"/query", bytes.NewReader(queryBody))
			qReq.Header.Set("Content-Type", "application/json")
			qRec := httptest.NewRecorder()
			router.ServeHTTP(qRec, qReq)

			t.Run("Then query returns the inserted row with success", func(t *testing.T) {
				if qRec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", qRec.Code, qRec.Body.String())
				}
				var res domain.D1QueryResult
				_ = json.NewDecoder(qRec.Body).Decode(&res)
				if len(res.Rows) != 1 {
					t.Fatalf("expected 1 result row, got: %+v", res)
				}
				if res.Rows[0]["type"] != "click" {
					t.Errorf("expected type click, got %v", res.Rows[0]["type"])
				}
			})
		})
	})

	t.Run("Given Dynamic Workers evaluation endpoint", func(t *testing.T) {
		t.Run("When evaluating JavaScript code on the fly", func(t *testing.T) {
			evalBody := []byte(`{
				"code": "export default { fetch(req) { return new Response('Dynamic Celld 0.5.1 Active'); } };",
				"method": "GET",
				"path": "/"
			}`)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/dynamic-workers/eval", bytes.NewReader(evalBody))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then returns 200 with evaluated response body", func(t *testing.T) {
				if rec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
				}
				var resp http_adapter.EvalDynamicWorkerResp
				_ = json.NewDecoder(rec.Body).Decode(&resp)
				if resp.Status != 200 || resp.Body != "Dynamic Celld 0.5.1 Active" {
					t.Errorf("unexpected dynamic worker response: %+v", resp)
				}
			})
		})
	})

	t.Run("Given Queues endpoints", func(t *testing.T) {
		var qID string
		t.Run("When creating a queue and sending a message", func(t *testing.T) {
			qBody := []byte(`{"name": "email-tasks", "max_retries": 3}`)
			qReq := httptest.NewRequest(http.MethodPost, "/api/v1/queues", bytes.NewReader(qBody))
			qReq.Header.Set("Content-Type", "application/json")
			qRec := httptest.NewRecorder()
			router.ServeHTTP(qRec, qReq)

			var queue domain.Queue
			_ = json.NewDecoder(qRec.Body).Decode(&queue)
			qID = queue.ID

			msgBody := []byte(`{"body": "send-welcome-email"}`)
			msgReq := httptest.NewRequest(http.MethodPost, "/api/v1/queues/"+qID+"/messages", bytes.NewReader(msgBody))
			msgReq.Header.Set("Content-Type", "application/json")
			msgRec := httptest.NewRecorder()
			router.ServeHTTP(msgRec, msgReq)

			t.Run("Then message is stored and retrievable", func(t *testing.T) {
				if msgRec.Code != http.StatusCreated {
					t.Fatalf("expected 201, got %d: %s", msgRec.Code, msgRec.Body.String())
				}
				listReq := httptest.NewRequest(http.MethodGet, "/api/v1/queues/"+qID+"/messages", nil)
				listRec := httptest.NewRecorder()
				router.ServeHTTP(listRec, listReq)

				var msgs []*domain.QueueMessage
				_ = json.NewDecoder(listRec.Body).Decode(&msgs)
				if len(msgs) != 1 || msgs[0].Body != "send-welcome-email" {
					t.Errorf("unexpected queue messages: %+v", msgs)
				}
			})
		})
	})
}
