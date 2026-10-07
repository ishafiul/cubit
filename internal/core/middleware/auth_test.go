package middleware_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/core/middleware"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTokenValidator struct {
	validateAccessFn func(tokenStr string) (*domain.JWTClaims, error)
	validateAPIFn    func(ctx context.Context, tokenPlain string) (*domain.APIToken, *domain.Role, error)
}

func (m *mockTokenValidator) ValidateAccessToken(tokenStr string) (*domain.JWTClaims, error) {
	if m.validateAccessFn != nil {
		return m.validateAccessFn(tokenStr)
	}
	return nil, errors.New("unimplemented access token validator")
}

func (m *mockTokenValidator) ValidateAPIToken(ctx context.Context, tokenPlain string) (*domain.APIToken, *domain.Role, error) {
	if m.validateAPIFn != nil {
		return m.validateAPIFn(ctx, tokenPlain)
	}
	return nil, nil, errors.New("unimplemented api token validator")
}

func TestAuthenticateMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Given an incoming request without Authorization header on a protected route", func(t *testing.T) {
		validator := &mockTokenValidator{}
		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/api/v1/apps", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When the endpoint is invoked then it returns 401 Unauthorized", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			var resp map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "missing or invalid authorization header", resp["error"])
		})
	})

	t.Run("Given an incoming request with non-Bearer Authorization header", func(t *testing.T) {
		validator := &mockTokenValidator{}
		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/api/v1/apps", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When the endpoint is invoked then it returns 401 Unauthorized", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
			req.Header.Set("Authorization", "Basic user:pass")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			var resp map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "missing or invalid authorization header", resp["error"])
		})
	})

	t.Run("Given an incoming request with an invalid Bearer JWT", func(t *testing.T) {
		validator := &mockTokenValidator{
			validateAccessFn: func(tokenStr string) (*domain.JWTClaims, error) {
				return nil, errors.New("token expired")
			},
		}
		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/api/v1/apps", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When the endpoint is invoked then it returns 401 Unauthorized", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
			req.Header.Set("Authorization", "Bearer invalid-jwt-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			var resp map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "invalid or expired token", resp["error"])
		})
	})

	t.Run("Given an incoming request with a valid Bearer JWT", func(t *testing.T) {
		expectedClaims := &domain.JWTClaims{
			UserID:      "usr-123",
			Email:       "alice@example.com",
			RoleID:      "developer",
			Permissions: []string{"apps:read", "apps:deploy"},
		}
		validator := &mockTokenValidator{
			validateAccessFn: func(tokenStr string) (*domain.JWTClaims, error) {
				if tokenStr == "valid-jwt-token" {
					return expectedClaims, nil
				}
				return nil, errors.New("bad token")
			},
		}
		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/api/v1/apps", func(c *gin.Context) {
			uid, _ := c.Get(middleware.ContextKeyUserID)
			email, _ := c.Get(middleware.ContextKeyEmail)
			roleID, _ := c.Get(middleware.ContextKeyRoleID)
			perms, _ := c.Get(middleware.ContextKeyPermissions)
			authType, _ := c.Get(middleware.ContextKeyAuthType)

			c.JSON(http.StatusOK, gin.H{
				"userId":      uid,
				"email":       email,
				"roleId":      roleID,
				"permissions": perms,
				"authType":    authType,
			})
		})

		t.Run("When the endpoint is invoked then it sets caller context and returns 200 OK", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
			req.Header.Set("Authorization", "Bearer valid-jwt-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "usr-123", resp["userId"])
			assert.Equal(t, "alice@example.com", resp["email"])
			assert.Equal(t, "developer", resp["roleId"])
			assert.Equal(t, middleware.AuthTypeJWT, resp["authType"])
		})
	})

	t.Run("Given an incoming request with a valid cbt_ API token", func(t *testing.T) {
		tokenID := "tok-abc-123"
		userID := "usr-ci-runner"
		now := time.Now().UTC()
		mockToken := &domain.APIToken{
			ID:        tokenID,
			UserID:    userID,
			Name:      "CI Deploy Token",
			RoleID:    domain.SystemRoleDeveloperID,
			CreatedAt: now,
		}
		mockRole := &domain.Role{
			ID:          domain.SystemRoleDeveloperID,
			Name:        "Developer",
			Permissions: []string{"apps:read", "apps:deploy"},
		}

		validator := &mockTokenValidator{
			validateAPIFn: func(ctx context.Context, tokenPlain string) (*domain.APIToken, *domain.Role, error) {
				if tokenPlain == "cbt_valid_secret_hash_token" {
					return mockToken, mockRole, nil
				}
				return nil, nil, errors.New("invalid api token")
			},
		}

		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/api/v1/apps", func(c *gin.Context) {
			uid, _ := c.Get(middleware.ContextKeyUserID)
			roleID, _ := c.Get(middleware.ContextKeyRoleID)
			perms, _ := c.Get(middleware.ContextKeyPermissions)
			tokenKeyID, _ := c.Get(middleware.ContextKeyTokenID)
			authType, _ := c.Get(middleware.ContextKeyAuthType)

			c.JSON(http.StatusOK, gin.H{
				"userId":      uid,
				"roleId":      roleID,
				"permissions": perms,
				"tokenId":     tokenKeyID,
				"authType":    authType,
			})
		})

		t.Run("When the endpoint is invoked then it validates API token and sets context", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
			req.Header.Set("Authorization", "Bearer cbt_valid_secret_hash_token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, userID, resp["userId"])
			assert.Equal(t, domain.SystemRoleDeveloperID, resp["roleId"])
			assert.Equal(t, tokenID, resp["tokenId"])
			assert.Equal(t, middleware.AuthTypeAPIToken, resp["authType"])
		})
	})

	t.Run("Given an incoming request with an invalid cbt_ API token", func(t *testing.T) {
		validator := &mockTokenValidator{
			validateAPIFn: func(ctx context.Context, tokenPlain string) (*domain.APIToken, *domain.Role, error) {
				return nil, nil, errors.New("api token not found")
			},
		}

		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/api/v1/apps", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When the endpoint is invoked then it returns 401 Unauthorized", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
			req.Header.Set("Authorization", "Bearer cbt_unknown_token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			var resp map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "invalid or expired token", resp["error"])
		})
	})

	t.Run("Given an incoming request with an expired cbt_ API token", func(t *testing.T) {
		validator := &mockTokenValidator{
			validateAPIFn: func(ctx context.Context, tokenPlain string) (*domain.APIToken, *domain.Role, error) {
				return nil, nil, errors.New("api token expired")
			},
		}

		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/api/v1/apps", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When the endpoint is invoked then it returns 401 Unauthorized", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/apps", nil)
			req.Header.Set("Authorization", "Bearer cbt_expired_token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			var resp map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "invalid or expired token", resp["error"])
		})
	})

	t.Run("Given requests to public endpoints without Authorization headers", func(t *testing.T) {
		validator := &mockTokenValidator{}
		router := gin.New()
		router.Use(middleware.Authenticate(validator))

		publicEndpoints := []string{
			"/health",
			"/api/v1/auth/status",
			"/api/v1/auth/setup",
			"/api/v1/auth/login",
			"/api/v1/auth/refresh",
			"/api/v1/auth/logout",
			"/api/v1/github/webhook",
			"/api/v1/github/manifest/callback",
			"/",
			"/index.html",
			"/dashboard",
			"/assets/app.js",
		}

		for _, endpoint := range publicEndpoints {
			router.Any(endpoint, func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"endpoint": c.Request.URL.Path})
			})
		}

		for _, endpoint := range publicEndpoints {
			t.Run("When calling public route "+endpoint+" then it bypasses authentication", func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, endpoint, nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				assert.Equal(t, http.StatusOK, w.Code)
			})
		}
	})
}
