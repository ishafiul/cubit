package deployment_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/deployment"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupDeploymentRouter(svc deployment.Service) *gin.Engine {
	r := gin.New()
	rg := r.Group("/api/v1")
	h := deployment.NewHandler(svc)
	h.RegisterRoutes(rg)
	return r
}

func TestDeploymentHandler(t *testing.T) {
	t.Run("Given an application and deployment handler", func(t *testing.T) {
		app := &domain.Application{
			ID:         "app-test",
			Name:       "test-app",
			SourceType: domain.SourceTypeInline,
			InlineCode: "export default { fetch() { return new Response('hello'); } }",
			CreatedAt:  time.Now().UTC(),
		}
		appMgr := &mockAppManager{app: app}
		repo := newMockDeploymentRepo()
		svc := deployment.NewService(repo, appMgr, nil, nil, "cubit-fleet")
		router := setupDeploymentRouter(svc)

		t.Run("When deploying an application via POST /api/v1/applications/:id/deploy", func(t *testing.T) {
			body := map[string]interface{}{
				"commitHash": "c0ffee1234",
			}
			jsonBody, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/applications/app-test/deploy", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			t.Run("Then it returns 201 Created with buildVersion 1", func(t *testing.T) {
				if w.Code != http.StatusCreated {
					t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
				}

				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["buildVersion"] != float64(1) {
					t.Fatalf("expected buildVersion 1, got %v", resp["buildVersion"])
				}
				if resp["status"] != "active" {
					t.Fatalf("expected status active, got %v", resp["status"])
				}
			})
		})

		t.Run("When listing deployments via GET /api/v1/applications/:id/deployments", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/applications/app-test/deployments", nil)
			router.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with the deployments list", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d", w.Code)
				}
				var list []map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &list)
				if len(list) != 1 {
					t.Fatalf("expected 1 deployment, got %d", len(list))
				}
			})
		})

		t.Run("When deploying directly via POST /api/v1/applications/:id/deploy/direct", func(t *testing.T) {
			body := []byte(`{"bundle":"export default { fetch: () => new Response('direct') };","commitMessage":"direct push"}`)
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/applications/app-test/deploy/direct", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			t.Run("Then it returns 201 Created with active status and new build version", func(t *testing.T) {
				if w.Code != http.StatusCreated {
					t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
				}

				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["status"] != "active" {
					t.Fatalf("expected status active, got %v", resp["status"])
				}
				if resp["buildVersion"] != float64(2) {
					t.Fatalf("expected buildVersion 2, got %v", resp["buildVersion"])
				}
			})
		})
	})
}
