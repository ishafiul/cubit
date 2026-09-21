package domain_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	domModule "github.com/ishaf/cubit/internal/modules/domain"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupDomainRouter(svc domModule.Service) *gin.Engine {
	r := gin.New()
	rg := r.Group("/api/v1")
	h := domModule.NewHandler(svc)
	h.RegisterRoutes(rg)
	return r
}

func TestDomainHandler(t *testing.T) {
	t.Run("Given a domain HTTP handler with an empty repository", func(t *testing.T) {
		repo := newMockDomainRepo()
		svc := domModule.NewService(repo, &mockAppVerifier{}, nil)
		router := setupDomainRouter(svc)

		t.Run("When binding a domain via POST /api/v1/domains", func(t *testing.T) {
			body := map[string]interface{}{
				"applicationId": "app-123",
				"hostname":      "my-site.com",
				"pathPrefix":    "/",
			}
			jsonBody, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/domains", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			t.Run("Then it returns 201 Created with domain data", func(t *testing.T) {
				if w.Code != http.StatusCreated {
					t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
				}

				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["hostname"] != "my-site.com" {
					t.Fatalf("expected hostname my-site.com, got %v", resp["hostname"])
				}
			})
		})

		t.Run("When listing domains via GET /api/v1/domains", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/domains", nil)
			router.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with registered domains", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d", w.Code)
				}
				var list []map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &list)
				if len(list) != 1 {
					t.Fatalf("expected 1 domain, got %d", len(list))
				}
			})
		})
	})
}
