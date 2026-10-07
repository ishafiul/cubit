package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
