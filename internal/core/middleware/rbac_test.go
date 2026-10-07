package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/core/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	gin.SetMode(gin.TestMode)
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
