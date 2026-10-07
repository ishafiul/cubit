package middleware_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/core/middleware"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockTokenValidator struct {
	validateFn func(tokenStr string) (*domain.JWTClaims, error)
}

func (m *mockTokenValidator) ValidateAccessToken(tokenStr string) (*domain.JWTClaims, error) {
	if m.validateFn != nil {
		return m.validateFn(tokenStr)
	}
	return nil, errors.New("unimplemented")
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestAuthenticateMiddleware(t *testing.T) {
	t.Run("Given an incoming request without Authorization header", func(t *testing.T) {
		validator := &mockTokenValidator{}
		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When the endpoint is invoked then it returns 401 Unauthorized", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
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
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When the endpoint is invoked then it returns 401 Unauthorized", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
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

	t.Run("Given an incoming request with an invalid Bearer token", func(t *testing.T) {
		validator := &mockTokenValidator{
			validateFn: func(tokenStr string) (*domain.JWTClaims, error) {
				return nil, errors.New("token expired")
			},
		}
		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/protected", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When the endpoint is invoked then it returns 401 Unauthorized", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer invalid-or-expired-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusUnauthorized, w.Code)
			var resp map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "invalid or expired token", resp["error"])
		})
	})

	t.Run("Given an incoming request with a valid Bearer token", func(t *testing.T) {
		expectedClaims := &domain.JWTClaims{
			UserID:      "usr-123",
			Email:       "alice@example.com",
			RoleID:      "developer",
			Permissions: []string{"apps:read", "apps:deploy"},
		}
		validator := &mockTokenValidator{
			validateFn: func(tokenStr string) (*domain.JWTClaims, error) {
				if tokenStr == "valid-token" {
					return expectedClaims, nil
				}
				return nil, errors.New("bad token")
			},
		}
		router := gin.New()
		router.Use(middleware.Authenticate(validator))
		router.GET("/protected", func(c *gin.Context) {
			uid, _ := c.Get(middleware.ContextKeyUserID)
			email, _ := c.Get(middleware.ContextKeyEmail)
			roleID, _ := c.Get(middleware.ContextKeyRoleID)
			perms, _ := c.Get(middleware.ContextKeyPermissions)

			c.JSON(http.StatusOK, gin.H{
				"userId":      uid,
				"email":       email,
				"roleId":      roleID,
				"permissions": perms,
			})
		})

		t.Run("When the endpoint is invoked then it sets caller context and returns 200 OK", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			req.Header.Set("Authorization", "Bearer valid-token")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "usr-123", resp["userId"])
			assert.Equal(t, "alice@example.com", resp["email"])
			assert.Equal(t, "developer", resp["roleId"])
		})
	})
}

func TestRequirePermissionMiddleware(t *testing.T) {
	t.Run("Given an unauthenticated request without permissions in context", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.RequirePermission("apps:deploy"))
		router.POST("/deploy", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "deployed"})
		})

		t.Run("When the endpoint is invoked then it returns 403 Forbidden", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/deploy", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
			var resp map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "forbidden", resp["error"])
			assert.Equal(t, "apps:deploy", resp["required_permission"])
		})
	})

	t.Run("Given an authenticated user with insufficient permissions", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.ContextKeyPermissions, []string{"apps:read", "nodes:read"})
			c.Next()
		})
		router.Use(middleware.RequirePermission("apps:deploy"))
		router.POST("/deploy", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "deployed"})
		})

		t.Run("When calling a write endpoint then it returns 403 Forbidden", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/deploy", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
			var resp map[string]string
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "forbidden", resp["error"])
		})
	})

	t.Run("Given an authenticated user with wildcard superuser permission", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.ContextKeyPermissions, []string{"*"})
			c.Next()
		})
		router.Use(middleware.RequirePermission("apps:deploy"))
		router.POST("/deploy", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "deployed"})
		})

		t.Run("When calling a write endpoint then it succeeds with 200 OK", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/deploy", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	})

	t.Run("Given an authenticated user with prefix wildcard permission", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.ContextKeyPermissions, []string{"apps:*"})
			c.Next()
		})
		router.Use(middleware.RequirePermission("apps:deploy"))
		router.POST("/deploy", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "deployed"})
		})

		t.Run("When calling an apps action endpoint then it succeeds with 200 OK", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/deploy", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	})
}

func TestRequireAnyMiddleware(t *testing.T) {
	t.Run("Given an authenticated user with one of the accepted permissions", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.ContextKeyPermissions, []string{"apps:read"})
			c.Next()
		})
		router.Use(middleware.RequireAny("apps:read", "apps:manage"))
		router.GET("/apps", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When invoking endpoint then it succeeds with 200 OK", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/apps", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	})

	t.Run("Given an authenticated user without any of the accepted permissions", func(t *testing.T) {
		router := gin.New()
		router.Use(func(c *gin.Context) {
			c.Set(middleware.ContextKeyPermissions, []string{"nodes:read"})
			c.Next()
		})
		router.Use(middleware.RequireAny("apps:read", "apps:manage"))
		router.GET("/apps", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "ok"})
		})

		t.Run("When invoking endpoint then it returns 403 Forbidden", func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/apps", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusForbidden, w.Code)
			var resp map[string]any
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			require.NoError(t, err)
			assert.Equal(t, "forbidden", resp["error"])
		})
	})
}
