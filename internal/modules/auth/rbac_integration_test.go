package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/core/middleware"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	appModule "github.com/ishaf/cubit/internal/modules/application"
	authModule "github.com/ishaf/cubit/internal/modules/auth"
	nodeModule "github.com/ishaf/cubit/internal/modules/node"
)

type rbacIntegrationHarness struct {
	router      *gin.Engine
	authService authModule.Service
	jwtSecret   string
}

func setupRBACIntegrationHarness(t *testing.T) *rbacIntegrationHarness {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "rbac_integration.db")
	dsn := "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

	database, err := db.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("failed to initialize sqlite: %v", err)
	}
	t.Cleanup(func() { database.Close() })

	jwtSecret := "integration-test-secret-32-bytes"
	authRepo := authModule.NewRepository(database)
	authService := authModule.NewService(authRepo, jwtSecret)
	authHandler := authModule.NewHandler(authService)

	appRepo := appModule.NewRepository(database)
	appService := appModule.NewService(appRepo, nil, nil, "cubit-fleet")
	appHandler := appModule.NewHandler(appService)

	nodeRepo := nodeModule.NewRepository(database)
	nodeService := nodeModule.NewService(nodeRepo, nil, nil, "s3://cubit-fleet")
	nodeHandler := nodeModule.NewHandler(nodeService)

	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Public routes
	publicAPI := r.Group("/api/v1")
	{
		authHandler.RegisterPublicRoutes(publicAPI)
	}

	// Protected routes
	protectedAPI := r.Group("/api/v1")
	protectedAPI.Use(middleware.Authenticate(authService))
	{
		authHandler.RegisterRoleRoutes(protectedAPI, middleware.RequirePermission)
		appHandler.RegisterRoutes(protectedAPI, middleware.RequirePermission)
		nodeHandler.RegisterRoutes(protectedAPI, middleware.RequirePermission)
	}

	return &rbacIntegrationHarness{
		router:      r,
		authService: authService,
		jwtSecret:   jwtSecret,
	}
}

func (h *rbacIntegrationHarness) createBearerToken(userID, email, roleID string, perms []string) string {
	claims := domain.JWTClaims{
		UserID:      userID,
		Email:       email,
		RoleID:      roleID,
		Permissions: perms,
	}
	token, _ := domain.GenerateJWT(claims, h.jwtSecret, 15*time.Minute)
	return "Bearer " + token
}

func TestRBAC_IntegrationEnforcement(t *testing.T) {
	harness := setupRBACIntegrationHarness(t)
	ctx := context.Background()

	// Setup cluster admin
	_, adminUser, err := harness.authService.Setup(ctx, "Cluster Admin", "admin@cubit.local", "AdminPass123!")
	if err != nil {
		t.Fatalf("failed to setup cluster admin: %v", err)
	}

	adminBearer := harness.createBearerToken(adminUser.ID, adminUser.Email, domain.SystemRoleAdminID, []string{"*"})
	devBearer := harness.createBearerToken("usr-dev", "dev@cubit.local", domain.SystemRoleDeveloperID, []string{
		"apps:*", "services:*", "deployments:*", "domains:*", "nodes:read",
	})
	viewerBearer := harness.createBearerToken("usr-viewer", "viewer@cubit.local", domain.SystemRoleViewerID, []string{
		"*:read", "apps:read", "services:read", "nodes:read", "domains:read",
	})

	t.Run("Given an unauthenticated caller", func(t *testing.T) {
		t.Run("When invoking protected GET /api/v1/applications then it returns 401 Unauthorized", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/applications", nil)
			w := httptest.NewRecorder()
			harness.router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized, got %d", w.Code)
			}
		})

		t.Run("When invoking public GET /api/v1/auth/status then it returns 200 OK without token", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
			w := httptest.NewRecorder()
			harness.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK for public status probe, got %d", w.Code)
			}
		})
	})

	t.Run("Given a caller with Viewer permissions", func(t *testing.T) {
		t.Run("When querying GET /api/v1/applications then it succeeds with 200 OK", func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/applications", nil)
			req.Header.Set("Authorization", viewerBearer)
			w := httptest.NewRecorder()
			harness.router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK for viewer reading apps, got %d", w.Code)
			}
		})

		t.Run("When attempting mutating POST /api/v1/applications then it is rejected with 403 Forbidden", func(t *testing.T) {
			body := map[string]any{
				"name":       "viewer-forbidden-app",
				"sourceType": "inline",
				"inlineCode": "export default {}",
				"branch":     "main",
			}
			bodyBytes, _ := json.Marshal(body)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(bodyBytes))
			req.Header.Set("Authorization", viewerBearer)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			harness.router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403 Forbidden for viewer mutating app, got %d", w.Code)
			}

			var resp map[string]any
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["error"] != "forbidden" {
				t.Errorf("expected error 'forbidden', got '%v'", resp["error"])
			}
		})
	})

	t.Run("Given a caller with Developer permissions", func(t *testing.T) {
		t.Run("When creating an application via POST /api/v1/applications then it succeeds with 201 Created", func(t *testing.T) {
			body := map[string]any{
				"name":       "dev-created-app",
				"sourceType": "inline",
				"inlineCode": "export default {}",
				"branch":     "main",
			}
			bodyBytes, _ := json.Marshal(body)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/applications", bytes.NewReader(bodyBytes))
			req.Header.Set("Authorization", devBearer)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			harness.router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201 Created for dev creating app, got %d: %s", w.Code, w.Body.String())
			}
		})

		t.Run("When attempting to create a cluster node via POST /api/v1/nodes then it is rejected with 403 Forbidden", func(t *testing.T) {
			body := map[string]any{
				"name":         "node-dev-forbidden",
				"ipAddress":    "192.168.1.100",
				"internalPort": 8080,
				"workerPort":   8081,
			}
			bodyBytes, _ := json.Marshal(body)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/nodes", bytes.NewReader(bodyBytes))
			req.Header.Set("Authorization", devBearer)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			harness.router.ServeHTTP(w, req)

			if w.Code != http.StatusForbidden {
				t.Fatalf("expected 403 Forbidden for developer creating node, got %d: %s", w.Code, w.Body.String())
			}
		})
	})

	t.Run("Given a caller with Admin superuser permissions", func(t *testing.T) {
		t.Run("When creating a cluster node via POST /api/v1/nodes then it succeeds with 201 Created", func(t *testing.T) {
			body := map[string]any{
				"name":         "node-admin-permitted",
				"ipAddress":    "192.168.1.101",
				"internalPort": 8080,
				"workerPort":   8081,
			}
			bodyBytes, _ := json.Marshal(body)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/nodes", bytes.NewReader(bodyBytes))
			req.Header.Set("Authorization", adminBearer)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			harness.router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201 Created for admin creating node, got %d: %s", w.Code, w.Body.String())
			}
		})

		t.Run("When managing custom roles via POST /api/v1/roles then it succeeds with 201 Created", func(t *testing.T) {
			body := map[string]any{
				"name":        fmt.Sprintf("Custom Admin Role %d", time.Now().UnixNano()),
				"description": "Custom role created by admin",
				"permissions": []string{"apps:*", "nodes:*"},
			}
			bodyBytes, _ := json.Marshal(body)

			req, _ := http.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewReader(bodyBytes))
			req.Header.Set("Authorization", adminBearer)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			harness.router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201 Created for admin creating role, got %d: %s", w.Code, w.Body.String())
			}
		})
	})
}
