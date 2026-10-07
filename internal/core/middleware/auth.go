package middleware

import (
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

// TokenValidator defines the contract required by Authenticate middleware to validate access tokens.
type TokenValidator interface {
	ValidateAccessToken(tokenStr string) (*domain.JWTClaims, error)
}

// Authenticate returns a Gin middleware that extracts and validates a Bearer JWT access token.
// On success, it sets caller identity and permissions in the Gin context.
// On failure, it aborts the request with 401 Unauthorized.
func Authenticate(validator TokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		c.Next()
	}
}
