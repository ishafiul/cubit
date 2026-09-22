package service_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/adapters/out/storage"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	srvModule "github.com/ishaf/cubit/internal/modules/service"
)

func setupTestGinRouter(t *testing.T) (*gin.Engine, *srvModule.ServicesService) {
	gin.SetMode(gin.TestMode)
	tempDir, err := os.MkdirTemp("", "cubit-srv-handler-test-*")
	if err != nil {
		t.Fatalf("failed creating temp dir: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	dbPath := filepath.Join(tempDir, "test.db")
	database, err := db.OpenSQLite(dbPath)
	if err != nil {
		t.Fatalf("failed opening sqlite: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	repo := srvModule.NewRepository(database, tempDir)
	storageAdapter, err := storage.NewLocalStorageAdapter(tempDir, string(storage.DriverGarageLocal))
	if err != nil {
		t.Fatalf("failed creating storage adapter: %v", err)
	}
	svc := srvModule.NewService(repo, nil, nil, storageAdapter, "cubit-fleet")
	handler := srvModule.NewHandler(svc)

	r := gin.New()
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)

	return r, svc
}

func TestServicesHandler(t *testing.T) {
	router, _ := setupTestGinRouter(t)

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
					t.Errorf("expected 'active-session-token', got '%s'", pair.Value)
				}
			})
		})
	})

	t.Run("Given D1 REST endpoints", func(t *testing.T) {
		var dbID string

		t.Run("When creating a D1 database", func(t *testing.T) {
			body := []byte(`{"name": "billing-db"}`)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/d1/databases", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then responds with 201 Created and database ID", func(t *testing.T) {
				if rec.Code != http.StatusCreated {
					t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
				}
				var db domain.D1Database
				_ = json.NewDecoder(rec.Body).Decode(&db)
				if db.ID == "" || db.Name != "billing-db" {
					t.Errorf("unexpected db: %+v", db)
				}
				dbID = db.ID
			})
		})

		t.Run("When executing D1 queries", func(t *testing.T) {
			ddlBody := []byte(`{"sql": "CREATE TABLE customers (id TEXT PRIMARY KEY, name TEXT);"}`)
			ddlReq := httptest.NewRequest(http.MethodPost, "/api/v1/d1/databases/"+dbID+"/query", bytes.NewReader(ddlBody))
			ddlReq.Header.Set("Content-Type", "application/json")
			ddlRec := httptest.NewRecorder()
			router.ServeHTTP(ddlRec, ddlReq)

			t.Run("Then DDL executes successfully", func(t *testing.T) {
				if ddlRec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", ddlRec.Code, ddlRec.Body.String())
				}
			})

			insBody := []byte(`{"sql": "INSERT INTO customers (id, name) VALUES ('cust_1', 'Alice');"}`)
			insReq := httptest.NewRequest(http.MethodPost, "/api/v1/d1/databases/"+dbID+"/query", bytes.NewReader(insBody))
			insReq.Header.Set("Content-Type", "application/json")
			insRec := httptest.NewRecorder()
			router.ServeHTTP(insRec, insReq)

			t.Run("Then INSERT executes successfully", func(t *testing.T) {
				if insRec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", insRec.Code, insRec.Body.String())
				}
			})

			selBody := []byte(`{"sql": "SELECT * FROM customers;"}`)
			selReq := httptest.NewRequest(http.MethodPost, "/api/v1/d1/databases/"+dbID+"/query", bytes.NewReader(selBody))
			selReq.Header.Set("Content-Type", "application/json")
			selRec := httptest.NewRecorder()
			router.ServeHTTP(selRec, selReq)

			t.Run("Then SELECT returns matching rows", func(t *testing.T) {
				if selRec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", selRec.Code, selRec.Body.String())
				}
				var qResult domain.D1QueryResult
				_ = json.NewDecoder(selRec.Body).Decode(&qResult)
				if len(qResult.Rows) != 1 || qResult.Rows[0]["name"] != "Alice" {
					t.Errorf("unexpected query result: %+v", qResult)
				}
			})
		})
	})

	t.Run("Given Queues REST endpoints", func(t *testing.T) {
		var qID string

		t.Run("When creating a queue and sending a message", func(t *testing.T) {
			body := []byte(`{"name": "order-processing"}`)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/queues", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			t.Run("Then queue is created with 201 Created", func(t *testing.T) {
				if rec.Code != http.StatusCreated {
					t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
				}
				var q domain.Queue
				_ = json.NewDecoder(rec.Body).Decode(&q)
				qID = q.ID
			})

			msgBody := []byte(`{"body": "order_456"}`)
			msgReq := httptest.NewRequest(http.MethodPost, "/api/v1/queues/"+qID+"/messages", bytes.NewReader(msgBody))
			msgReq.Header.Set("Content-Type", "application/json")
			msgRec := httptest.NewRecorder()
			router.ServeHTTP(msgRec, msgReq)

			t.Run("Then message is sent with 201 Created", func(t *testing.T) {
				if msgRec.Code != http.StatusCreated {
					t.Fatalf("expected 201, got %d: %s", msgRec.Code, msgRec.Body.String())
				}
			})
		})
	})

	t.Run("Given R2 REST endpoints", func(t *testing.T) {
		bucketName := "assets-bucket"

		t.Run("When creating a bucket and managing objects", func(t *testing.T) {
			bktBody := []byte(`{"name": "assets-bucket"}`)
			bktReq := httptest.NewRequest(http.MethodPost, "/api/v1/r2/buckets", bytes.NewReader(bktBody))
			bktReq.Header.Set("Content-Type", "application/json")
			bktRec := httptest.NewRecorder()
			router.ServeHTTP(bktRec, bktReq)

			t.Run("Then bucket is created with 201 Created", func(t *testing.T) {
				if bktRec.Code != http.StatusCreated {
					t.Fatalf("expected 201, got %d: %s", bktRec.Code, bktRec.Body.String())
				}
			})

			upBody := []byte(`{"key": "docs/readme.txt", "content": "hello R2 object"}`)
			upReq := httptest.NewRequest(http.MethodPost, "/api/v1/r2/buckets/"+bucketName+"/upload", bytes.NewReader(upBody))
			upReq.Header.Set("Content-Type", "application/json")
			upRec := httptest.NewRecorder()
			router.ServeHTTP(upRec, upReq)

			t.Run("Then object upload succeeds with 200 OK", func(t *testing.T) {
				if upRec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", upRec.Code, upRec.Body.String())
				}
			})

			getReq := httptest.NewRequest(http.MethodGet, "/api/v1/r2/buckets/"+bucketName+"/objects/docs/readme.txt", nil)
			getRec := httptest.NewRecorder()
			router.ServeHTTP(getRec, getReq)

			t.Run("Then object is retrieved with 200 OK and expected content", func(t *testing.T) {
				if getRec.Code != http.StatusOK {
					t.Fatalf("expected 200, got %d: %s", getRec.Code, getRec.Body.String())
				}
				if getRec.Body.String() != "hello R2 object" {
					t.Fatalf("expected 'hello R2 object', got '%s'", getRec.Body.String())
				}
			})

			delReq := httptest.NewRequest(http.MethodDelete, "/api/v1/r2/buckets/"+bucketName+"/objects/docs/readme.txt", nil)
			delRec := httptest.NewRecorder()
			router.ServeHTTP(delRec, delReq)

			t.Run("Then object deletion responds with 204 No Content", func(t *testing.T) {
				if delRec.Code != http.StatusNoContent {
					t.Fatalf("expected 204, got %d: %s", delRec.Code, delRec.Body.String())
				}
			})
		})
	})
}
