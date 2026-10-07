package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	authModule "github.com/ishaf/cubit/internal/modules/auth"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func setupTestHandler(t *testing.T) (*gin.Engine, authModule.Service, authModule.Repository) {
	t.Helper()
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "auth_handler_test.db")
	dsn := "file:" + dbPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(ON)"

	database, err := db.OpenSQLite(dsn)
	if err != nil {
		t.Fatalf("failed to open test sqlite: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
	})

	repo := authModule.NewRepository(database)
	secret := "cluster-secret-key-32-bytes-long!"
	service := authModule.NewService(repo, secret)
	handler := authModule.NewHandler(service)

	r := gin.New()
	api := r.Group("/api/v1")
	handler.RegisterRoutes(api)

	return r, service, repo
}

func TestAuthHandler_LoginRefreshLogout(t *testing.T) {
	ctx := context.Background()
	router, service, repo := setupTestHandler(t)

	// Seed role and test user
	role := &domain.Role{
		ID:          "role-operator",
		Name:        "Operator",
		Description: "Operator role",
		IsSystem:    true,
		Permissions: []string{"apps:*"},
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	if err := repo.CreateRole(ctx, role); err != nil {
		t.Fatalf("failed to seed role: %v", err)
	}

	password := "SecretPass123!"
	passHash, err := domain.HashPassword(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	user, err := domain.NewUser("usr-op-1", "Bob Operator", "bob@cubit.local", passHash, role.ID)
	if err != nil {
		t.Fatalf("failed to create domain user: %v", err)
	}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("failed to persist user: %v", err)
	}

	t.Run("Given valid user credentials", func(t *testing.T) {
		t.Run("When calling POST /api/v1/auth/login then it returns 200 with tokens and user info", func(t *testing.T) {
			payload := map[string]string{
				"email":    "bob@cubit.local",
				"password": password,
			}
			bodyBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse json response: %v", err)
			}

			accessToken, ok := resp["accessToken"].(string)
			if !ok || accessToken == "" {
				t.Errorf("expected non-empty accessToken in response")
			}
			refreshToken, ok := resp["refreshToken"].(string)
			if !ok || refreshToken == "" {
				t.Errorf("expected non-empty refreshToken in response")
			}

			userResp, ok := resp["user"].(map[string]interface{})
			if !ok || userResp["email"] != "bob@cubit.local" {
				t.Errorf("expected user email bob@cubit.local, got %v", userResp)
			}
		})
	})

	t.Run("Given incorrect password or non-existent user", func(t *testing.T) {
		t.Run("When calling POST /api/v1/auth/login with wrong password then it returns 401 Unauthorized", func(t *testing.T) {
			payload := map[string]string{
				"email":    "bob@cubit.local",
				"password": "WrongPassword!",
			}
			bodyBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized, got %d: %s", w.Code, w.Body.String())
			}
		})

		t.Run("When calling POST /api/v1/auth/login with non-existent email then it returns 401 Unauthorized", func(t *testing.T) {
			payload := map[string]string{
				"email":    "missing@cubit.local",
				"password": password,
			}
			bodyBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized, got %d: %s", w.Code, w.Body.String())
			}
		})

		t.Run("When calling POST /api/v1/auth/login with malformed body then it returns 400 Bad Request", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader([]byte("{invalid-json")))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", w.Code, w.Body.String())
			}
		})
	})

	t.Run("Given an active refresh token", func(t *testing.T) {
		pair, err := service.IssueTokenPair(ctx, user)
		if err != nil {
			t.Fatalf("failed to issue token pair: %v", err)
		}

		t.Run("When calling POST /api/v1/auth/refresh then it rotates the tokens successfully", func(t *testing.T) {
			payload := map[string]string{
				"refreshToken": pair.RefreshToken,
			}
			bodyBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			newRef, ok := resp["refreshToken"].(string)
			if !ok || newRef == pair.RefreshToken {
				t.Errorf("expected rotated new refresh token, got %v", newRef)
			}
		})

		t.Run("When reusing the old refresh token then it returns 401 Unauthorized", func(t *testing.T) {
			payload := map[string]string{
				"refreshToken": pair.RefreshToken,
			}
			bodyBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 Unauthorized, got %d: %s", w.Code, w.Body.String())
			}
		})
	})

	t.Run("Given an active session to logout", func(t *testing.T) {
		pair, err := service.IssueTokenPair(ctx, user)
		if err != nil {
			t.Fatalf("failed to issue token pair: %v", err)
		}

		t.Run("When calling POST /api/v1/auth/logout then it revokes the token and returns 200", func(t *testing.T) {
			payload := map[string]string{
				"refreshToken": pair.RefreshToken,
			}
			bodyBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/logout", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			// Subsequent refresh must fail with 401
			w2 := httptest.NewRecorder()
			req2, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(bodyBytes))
			req2.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w2, req2)

			if w2.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401 on refresh after logout, got %d", w2.Code)
			}
		})
	})
}

func TestAuthHandler_StatusProbe(t *testing.T) {
	router, _, repo := setupTestHandler(t)

	t.Run("Given an empty database", func(t *testing.T) {
		t.Run("When calling GET /api/v1/auth/status then it returns 200 with initialized: false", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse json response: %v", err)
			}
			if resp["initialized"] != false {
				t.Errorf("expected initialized: false, got %v", resp["initialized"])
			}
			if resp["version"] != domain.CubitVersion {
				t.Errorf("expected version %s, got %v", domain.CubitVersion, resp["version"])
			}
		})
	})

	t.Run("Given a database with users", func(t *testing.T) {
		ctx := context.Background()
		role, err := repo.GetRoleByID(ctx, domain.SystemRoleAdminID)
		if err != nil {
			t.Fatalf("failed to retrieve pre-seeded admin role: %v", err)
		}
		user, _ := domain.NewUser("usr-1", "Admin", "admin@cubit.local", "hash", role.ID)
		_ = repo.CreateUser(ctx, user)

		t.Run("When calling GET /api/v1/auth/status then it returns 200 with initialized: true", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/auth/status", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse json response: %v", err)
			}
			if resp["initialized"] != true {
				t.Errorf("expected initialized: true, got %v", resp["initialized"])
			}
			if resp["version"] != domain.CubitVersion {
				t.Errorf("expected version %s, got %v", domain.CubitVersion, resp["version"])
			}
		})
	})
}

func TestAuthHandler_Setup(t *testing.T) {
	router, _, _ := setupTestHandler(t)

	t.Run("Given an uninitialized cluster", func(t *testing.T) {
		t.Run("When calling POST /api/v1/auth/setup with valid credentials then it returns 201 Created and active tokens", func(t *testing.T) {
			payload := map[string]string{
				"name":     "Super Admin",
				"email":    "superadmin@cubit.local",
				"password": "SuperSecretPass123!",
			}
			bodyBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse json response: %v", err)
			}

			if resp["accessToken"] == "" || resp["accessToken"] == nil {
				t.Errorf("expected non-empty accessToken")
			}
			if resp["refreshToken"] == "" || resp["refreshToken"] == nil {
				t.Errorf("expected non-empty refreshToken")
			}
			userMap, ok := resp["user"].(map[string]interface{})
			if !ok || userMap["email"] != "superadmin@cubit.local" {
				t.Errorf("expected user with email superadmin@cubit.local, got %v", userMap)
			}
		})

		t.Run("When calling POST /api/v1/auth/setup again then it returns 409 Conflict with error message", func(t *testing.T) {
			payload := map[string]string{
				"name":     "Another Admin",
				"email":    "another@cubit.local",
				"password": "AnotherPass123!",
			}
			bodyBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
			}

			var resp map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to parse json response: %v", err)
			}
			if resp["error"] != "setup has already been completed" {
				t.Errorf("expected error 'setup has already been completed', got '%v'", resp["error"])
			}
		})
	})

	t.Run("Given invalid setup payloads", func(t *testing.T) {
		cleanRouter, _, _ := setupTestHandler(t)

		t.Run("When name or email or password is missing then it returns 400 Bad Request", func(t *testing.T) {
			payload := map[string]string{
				"email": "onlyemail@cubit.local",
			}
			bodyBytes, _ := json.Marshal(payload)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/setup", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			cleanRouter.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request, got %d: %s", w.Code, w.Body.String())
			}
		})
	})
}

func TestAuthHandler_RolesAndPermissions(t *testing.T) {
	router, _, _ := setupTestHandler(t)

	t.Run("Given permissions catalog endpoint", func(t *testing.T) {
		t.Run("When GET /api/v1/permissions is called then it returns 200 with permissions list", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/permissions", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			var resp map[string][]string
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if len(resp["permissions"]) == 0 {
				t.Error("expected non-empty permissions list")
			}
		})
	})

	t.Run("Given role management CRUD endpoints", func(t *testing.T) {
		t.Run("When GET /api/v1/roles is called then it returns list of roles", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/roles", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			var roles []domain.Role
			if err := json.Unmarshal(w.Body.Bytes(), &roles); err != nil {
				t.Fatalf("failed to decode roles: %v", err)
			}
			if len(roles) < 3 {
				t.Errorf("expected at least 3 roles, got %d", len(roles))
			}
		})

		var createdRoleID string

		t.Run("When POST /api/v1/roles creates a custom role then it returns 201 Created", func(t *testing.T) {
			body := map[string]any{
				"name":        "Security Auditor",
				"description": "Compliance and auditing role",
				"permissions": []string{"*:read"},
			}
			bodyBytes, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
			}

			var created domain.Role
			if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
				t.Fatalf("failed to decode created role: %v", err)
			}
			if created.ID == "" || created.Name != "Security Auditor" || created.IsSystem {
				t.Errorf("unexpected created role: %+v", created)
			}
			createdRoleID = created.ID
		})

		t.Run("When POST /api/v1/roles has a duplicate name then it returns 409 Conflict", func(t *testing.T) {
			body := map[string]any{
				"name":        "Security Auditor",
				"description": "Duplicate name",
			}
			bodyBytes, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusConflict {
				t.Fatalf("expected 409 Conflict, got %d: %s", w.Code, w.Body.String())
			}
		})

		t.Run("When GET /api/v1/roles/:id is called with valid ID then it returns 200 OK", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/roles/"+createdRoleID, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}
		})

		t.Run("When GET /api/v1/roles/:id is called with unknown ID then it returns 404 Not Found", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/roles/non-existent-id", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusNotFound {
				t.Fatalf("expected 404 Not Found, got %d: %s", w.Code, w.Body.String())
			}
		})

		t.Run("When PUT /api/v1/roles/:id targets a system role then it returns 400 Bad Request", func(t *testing.T) {
			body := map[string]any{
				"name":        "New Admin",
				"description": "Try modifying root admin",
				"permissions": []string{"*"},
			}
			bodyBytes, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, "/api/v1/roles/admin", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for system role update, got %d: %s", w.Code, w.Body.String())
			}

			var resp map[string]string
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["error"] != "cannot modify system roles" {
				t.Errorf("expected error 'cannot modify system roles', got '%s'", resp["error"])
			}
		})

		t.Run("When PUT /api/v1/roles/:id updates a custom role then it returns 200 OK", func(t *testing.T) {
			body := map[string]any{
				"name":        "Lead Security Auditor",
				"description": "Updated auditing description",
				"permissions": []string{"*:read", "nodes:read"},
			}
			bodyBytes, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPut, "/api/v1/roles/"+createdRoleID, bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			var updated domain.Role
			_ = json.Unmarshal(w.Body.Bytes(), &updated)
			if updated.Name != "Lead Security Auditor" {
				t.Errorf("expected updated name, got %s", updated.Name)
			}
		})

		t.Run("When DELETE /api/v1/roles/:id targets a system role then it returns 400 Bad Request", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/roles/developer", nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400 Bad Request for system role deletion, got %d: %s", w.Code, w.Body.String())
			}

			var resp map[string]string
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
			if resp["error"] != "cannot delete system roles" {
				t.Errorf("expected error 'cannot delete system roles', got '%s'", resp["error"])
			}
		})

		t.Run("When DELETE /api/v1/roles/:id deletes a custom role then it returns 200 OK", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/roles/"+createdRoleID, nil)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			// Verify 404 subsequent fetch
			wFetch := httptest.NewRecorder()
			reqFetch, _ := http.NewRequest(http.MethodGet, "/api/v1/roles/"+createdRoleID, nil)
			router.ServeHTTP(wFetch, reqFetch)
			if wFetch.Code != http.StatusNotFound {
				t.Errorf("expected 404 Not Found after deletion, got %d", wFetch.Code)
			}
		})
	})
}

func TestAuthHandler_APITokens(t *testing.T) {
	ctx := context.Background()
	router, _, repo := setupTestHandler(t)

	// Create user
	user, err := domain.NewUser("usr-tok-test", "Token Tester", "tok@cubit.local", "hash", domain.SystemRoleDeveloperID)
	if err != nil {
		t.Fatalf("failed to create user: %v", err)
	}
	_ = repo.CreateUser(ctx, user)

	t.Run("Given token management endpoints", func(t *testing.T) {
		var createdTokenID string
		var plainSecret string

		t.Run("When POST /api/v1/tokens creates a new token then it returns 201 Created with secret", func(t *testing.T) {
			body := map[string]any{
				"name":          "Automation Pipeline",
				"expiresInDays": 14,
			}
			bodyBytes, _ := json.Marshal(body)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodPost, "/api/v1/tokens", bytes.NewReader(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", user.ID)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusCreated {
				t.Fatalf("expected 201 Created, got %d: %s", w.Code, w.Body.String())
			}

			var resp struct {
				Token    string          `json:"token"`
				APIToken domain.APIToken `json:"apiToken"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}

			if !strings.HasPrefix(resp.Token, "cbt_") {
				t.Errorf("expected token to start with cbt_, got %s", resp.Token)
			}
			if resp.APIToken.ID == "" || resp.APIToken.Name != "Automation Pipeline" {
				t.Errorf("unexpected api token payload: %+v", resp.APIToken)
			}

			createdTokenID = resp.APIToken.ID
			plainSecret = resp.Token
		})

		t.Run("When GET /api/v1/tokens is called then it returns active tokens without leaking hash", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodGet, "/api/v1/tokens", nil)
			req.Header.Set("X-User-ID", user.ID)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			var tokens []map[string]any
			if err := json.Unmarshal(w.Body.Bytes(), &tokens); err != nil {
				t.Fatalf("failed to decode tokens: %v", err)
			}

			if len(tokens) != 1 {
				t.Fatalf("expected 1 token, got %d", len(tokens))
			}
			if tokens[0]["name"] != "Automation Pipeline" {
				t.Errorf("expected token name 'Automation Pipeline', got %v", tokens[0]["name"])
			}
			if _, hasHash := tokens[0]["token_hash"]; hasHash {
				t.Error("token_hash must not be exposed in serialized JSON")
			}
		})

		t.Run("When DELETE /api/v1/tokens/:id deletes the token then it is revoked", func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(http.MethodDelete, "/api/v1/tokens/"+createdTokenID, nil)
			req.Header.Set("X-User-ID", user.ID)
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Fatalf("expected 200 OK, got %d: %s", w.Code, w.Body.String())
			}

			// Verify empty tokens list
			wList := httptest.NewRecorder()
			reqList, _ := http.NewRequest(http.MethodGet, "/api/v1/tokens", nil)
			reqList.Header.Set("X-User-ID", user.ID)
			router.ServeHTTP(wList, reqList)

			var tokens []domain.APIToken
			_ = json.Unmarshal(wList.Body.Bytes(), &tokens)
			if len(tokens) != 0 {
				t.Errorf("expected 0 tokens after deletion, got %d", len(tokens))
			}
			_ = plainSecret
		})
	})
}


