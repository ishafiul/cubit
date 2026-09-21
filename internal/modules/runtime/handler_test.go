package runtime_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
	rtModule "github.com/ishaf/cubit/internal/modules/runtime"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupRuntimeRouter(svc rtModule.Service) *gin.Engine {
	r := gin.New()
	rg := r.Group("/api/v1")
	h := rtModule.NewHandler(svc)
	h.RegisterRoutes(rg)
	return r
}

func TestRuntimeHandler(t *testing.T) {
	t.Run("Given a runtime HTTP handler with running fleet", func(t *testing.T) {
		nodes := []*domain.Node{
			{ID: "node-1", Name: "n1", Status: domain.NodeStatusActive, CelldVersion: "0.5.1"},
		}
		repo := &mockNodeLister{nodes: nodes}
		svc := rtModule.NewService(repo, nil, nil, "s3://cubit-fleet", "0.5.1")
		router := setupRuntimeRouter(svc)

		t.Run("When getting runtime status via GET /api/v1/runtime/status", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/runtime/status", nil)
			router.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with activeNodesCount 1", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d", w.Code)
				}
				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["activeNodesCount"] != float64(1) {
					t.Fatalf("expected activeNodesCount 1, got %v", resp["activeNodesCount"])
				}
			})
		})

		t.Run("When triggering upgrade via POST /api/v1/runtime/upgrade", func(t *testing.T) {
			body := map[string]string{"targetVersion": "0.5.2"}
			jsonBody, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/runtime/upgrade", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with upgrade confirmation", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
				}
				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["targetVersion"] != "0.5.2" {
					t.Fatalf("expected targetVersion 0.5.2, got %v", resp["targetVersion"])
				}
			})
		})
	})
}
