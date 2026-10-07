package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
)

const (
	ContextKeyUserID      = "user_id"
	ContextKeyEmail       = "user_email"
	ContextKeyRoleID      = "role_id"
	ContextKeyPermissions = "permissions"
)

// RequirePermission verifies that the authenticated caller has the specified permission.
// If the permission check fails, it aborts the request with 403 Forbidden.
func RequirePermission(perm string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permsVal, exists := c.Get(ContextKeyPermissions)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":               "forbidden",
				"required_permission": perm,
			})
			return
		}

		userPerms, ok := permsVal.([]string)
		if !ok || !domain.HasPermission(userPerms, perm) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":               "forbidden",
				"required_permission": perm,
			})
			return
		}

		c.Next()
	}
}

// RequireAny verifies that the authenticated caller has at least one of the specified permissions.
// If none of the permissions match, it aborts the request with 403 Forbidden.
func RequireAny(perms ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		permsVal, exists := c.Get(ContextKeyPermissions)
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":                "forbidden",
				"required_permissions": perms,
			})
			return
		}

		userPerms, ok := permsVal.([]string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error":                "forbidden",
				"required_permissions": perms,
			})
			return
		}

		for _, required := range perms {
			if domain.HasPermission(userPerms, required) {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error":                "forbidden",
			"required_permissions": perms,
		})
	}
}
