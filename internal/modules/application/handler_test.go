package application_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/application"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestRouter(svc application.Service) *gin.Engine {
	r := gin.New()
	rg := r.Group("/api/v1")
	h := application.NewHandler(svc)
	h.RegisterRoutes(rg)
	return r
}

func TestApplicationHandler(t *testing.T) {
	t.Run("Given an application HTTP handler with an empty repository", func(t *testing.T) {
		repo := newMockAppRepo()
		svc := application.NewService(repo, nil, nil, "cubit-fleet")
		router := setupTestRouter(svc)

		t.Run("When creating an application via POST /api/v1/applications", func(t *testing.T) {
			body := map[string]interface{}{
				"name":       "api-worker",
				"sourceType": "git",
				"gitRepo":    "https://github.com/example/api",
				"branch":     "main",
				"autoDeploy": true,
			}
			jsonBody, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			t.Run("Then it returns 201 Created with application payload", func(t *testing.T) {
				if w.Code != http.StatusCreated {
					t.Fatalf("expected status 201, got %d: %s", w.Code, w.Body.String())
				}

				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)
				if resp["name"] != "api-worker" {
					t.Fatalf("expected name api-worker, got %v", resp["name"])
				}
				if resp["autoDeploy"] != true {
					t.Fatalf("expected autoDeploy true, got %v", resp["autoDeploy"])
				}
			})
		})

		t.Run("When listing applications via GET /api/v1/applications", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/applications", nil)
			router.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with the array of applications", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d", w.Code)
				}
				var list []map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &list)
				if len(list) != 1 {
					t.Fatalf("expected 1 application, got %d", len(list))
				}
			})
		})

		t.Run("When retrieving a non-existent application via GET /api/v1/applications/:id", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/applications/non-existent-id", nil)
			router.ServeHTTP(w, req)

			t.Run("Then it returns 404 Not Found", func(t *testing.T) {
				if w.Code != http.StatusNotFound {
					t.Fatalf("expected status 404, got %d", w.Code)
				}
			})
		})

		t.Run("When importing wrangler.json via POST /api/v1/applications/:id/wrangler/import", func(t *testing.T) {
			app, _ := svc.Create(
				context.Background(),
				"import-worker",
				domain.SourceTypeInline,
				"",
				"main",
				"",
				"export default {}",
				false,
				[]domain.EnvironmentVariable{
					{Key: "EXISTING_SECRET", Value: "topsecret", IsSecret: true},
				},
				nil,
			)

			body := map[string]interface{}{
				"rawConfig": `{
					"compatibility_date": "2024-09-23",
					"vars": {
						"EXISTING_SECRET": "placeholder",
						"API_URL": "https://api.example.com"
					},
					"kv_namespaces": [
						{ "binding": "CACHE", "id": "kv-id-99" }
					]
				}`,
				"format": "auto",
			}
			jsonBody, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/applications/"+app.ID+"/wrangler/import", bytes.NewReader(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			t.Run("Then it returns 200 OK with import summary and updated application", func(t *testing.T) {
				if w.Code != http.StatusOK {
					t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
				}

				var resp map[string]interface{}
				_ = json.Unmarshal(w.Body.Bytes(), &resp)

				if resp["success"] != true {
					t.Fatalf("expected success true, got %v", resp["success"])
				}
				if resp["importedVarsCount"] != float64(2) {
					t.Errorf("expected 2 imported vars, got %v", resp["importedVarsCount"])
				}
				if resp["preservedSecretsCount"] != float64(1) {
					t.Errorf("expected 1 preserved secret, got %v", resp["preservedSecretsCount"])
				}
				if resp["compatibilityDate"] != "2024-09-23" {
					t.Errorf("expected compat date 2024-09-23, got %v", resp["compatibilityDate"])
				}

				appData := resp["application"].(map[string]interface{})
				envVars := appData["envVars"].([]interface{})
				foundSecret := false
				foundAPI := false
				for _, evRaw := range envVars {
					ev := evRaw.(map[string]interface{})
					k := ev["key"]
					if k == nil {
						k = ev["Key"]
					}
					v := ev["value"]
					if v == nil {
						v = ev["Value"]
					}
					isSec := ev["isSecret"]
					if isSec == nil {
						isSec = ev["IsSecret"]
					}
					if k == "EXISTING_SECRET" && isSec == true {
						foundSecret = true
					}
					if k == "API_URL" && v == "https://api.example.com" {
						foundAPI = true
					}
				}
				if !foundSecret {
					t.Errorf("expected EXISTING_SECRET to remain secret in response")
				}
				if !foundAPI {
					t.Errorf("expected API_URL to be imported in response")
				}
			})
		})
	})
}
