package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
)

var (
	ErrMissingAuthHeader = errors.New("missing or invalid authorization header")
	ErrInvalidToken      = errors.New("invalid or expired token")
)

// TokenValidator defines the contract required by Authenticate middleware to validate access and API tokens.
type TokenValidator interface {
	ValidateAccessToken(tokenStr string) (*domain.JWTClaims, error)
	ValidateAPIToken(ctx context.Context, tokenPlain string) (*domain.APIToken, *domain.Role, error)
}

// isPublicPath checks whether an incoming request path should bypass authentication.
func isPublicPath(path string) bool {
	if path == "/health" || !strings.HasPrefix(path, "/api/") {
		return true
	}
	clean := strings.TrimSuffix(path, "/")
	switch clean {
	case "/api/v1/auth/status",
		"/api/v1/auth/setup",
		"/api/v1/auth/login",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout",
		"/api/v1/github/webhook",
		"/api/v1/github/manifest/callback":
		return true
	}
	return false
}

// Authenticate returns a Gin middleware that extracts and validates a Bearer token.
// If the token starts with "cbt_", it is validated as a high-entropy stateful personal access token.
// Otherwise, it is parsed and validated as an HMAC-SHA256 JWT access token.
// Public routes (/health, /api/v1/auth/status, /api/v1/auth/setup, /api/v1/auth/login, /api/v1/auth/refresh,
// /api/v1/auth/logout, GitHub webhooks/callbacks, and static assets) bypass authentication.
// On success, caller identity and permissions are injected into the Gin context.
// On failure, it aborts the request with 401 Unauthorized.
func Authenticate(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublicPath(c.Request.URL.Path) {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing or invalid authorization header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "missing or invalid authorization header",
			})
			return
		}

		tokenStr := strings.TrimSpace(parts[1])
		if strings.HasPrefix(tokenStr, "cbt_") {
			apiToken, role, err := validator.ValidateAPIToken(c.Request.Context(), tokenStr)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "invalid or expired token",
				})
				return
			}

			c.Set(ContextKeyUserID, apiToken.UserID)
			c.Set(ContextKeyRoleID, role.ID)
			c.Set(ContextKeyPermissions, role.Permissions)
			c.Set(ContextKeyTokenID, apiToken.ID)
			c.Set(ContextKeyAuthType, AuthTypeAPIToken)
		} else {
			claims, err := validator.ValidateAccessToken(tokenStr)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error": "invalid or expired token",
				})
				return
			}

			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyEmail, claims.Email)
			c.Set(ContextKeyRoleID, claims.RoleID)
			c.Set(ContextKeyPermissions, claims.Permissions)
			c.Set(ContextKeyAuthType, AuthTypeJWT)
		}

		c.Next()
	}
}
