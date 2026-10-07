package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts all authentication endpoints under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", h.Logout)
	}
}
