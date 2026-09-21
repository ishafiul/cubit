package node_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/modules/node"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupNodeRouter(svc node.Service) *gin.Engine {
	r := gin.New()
	rg := r.Group("/api/v1")
	h := node.NewHandler(svc)
	h.RegisterRoutes(rg)
	return r
}

func TestNodeHandler(t *testing.T) {
	t.Run("Given a node HTTP handler with an empty repository", func(t *testing.T) {
		repo := newMockNodeRepo()
		syncer := &mockRouteSyncer{}
		svc := node.NewService(repo, nil, syncer, "s3://cubit-fleet")
		router := setupNodeRouter(svc)

		var createdID string

		t.Run("When registering a node via POST /api/v1/nodes", func(t *testing.T) {
			body := map[string]interface{}{
				"name":      "edge-node-1",
				"ipAddress": "10.0.0.1",
			}
			jsonBody, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/nodes", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			t.Run("Then it returns 201 Created with node data", func(t *testing.T) {
				if w.Code != http.StatusCreated {
					t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
				}

				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["name"] != "edge-node-1" {
					t.Fatalf("expected name edge-node-1, got %v", resp["name"])
				}
				if id, ok := resp["id"].(string); ok {
					createdID = id
				} else {
					t.Fatalf("expected id in response, got %v", resp["id"])
				}
			})
		})

		t.Run("When listing nodes via GET /api/v1/nodes", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/nodes", nil)
			router.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with registered nodes", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d", w.Code)
				}
				var list []map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &list)
				if len(list) != 1 {
					t.Fatalf("expected 1 node, got %d", len(list))
				}
			})
		})

		t.Run("When draining a node via POST /api/v1/nodes/:id/drain", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/nodes/"+createdID+"/drain", nil)
			router.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with status draining", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
				}
				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["status"] != "draining" {
					t.Fatalf("expected status draining, got %v", resp["status"])
				}
			})
		})

		t.Run("When activating a node via POST /api/v1/nodes/:id/activate", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/nodes/"+createdID+"/activate", nil)
			router.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with status active", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
				}
				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["status"] != "active" {
					t.Fatalf("expected status active, got %v", resp["status"])
				}
			})
		})

		t.Run("When deleting a node via DELETE /api/v1/nodes/:id", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/nodes/"+createdID, nil)
			router.ServeHTTP(w, req)

			t.Run("Then it returns 204 No Content and route syncer is triggered", func(t *testing.T) {
				if w.Code != http.StatusNoContent {
					t.Fatalf("expected status 204, got %d: %s", w.Code, w.Body.String())
				}
				if syncer.syncCount != 4 {
					t.Fatalf("expected 4 syncs (create, drain, activate, delete), got %d", syncer.syncCount)
				}
			})
		})
	})
}
