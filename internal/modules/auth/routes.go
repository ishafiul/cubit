package auth

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes mounts public authentication and setup endpoints under the provided RouterGroup.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	{
		auth.GET("/status", h.Status)
		auth.POST("/setup", h.Setup)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.Refresh)
		auth.POST("/logout", h.Logout)
	}
}

// RegisterRoleRoutes mounts permissions and roles endpoints with optional RBAC middlewares.
func (h *Handler) RegisterRoleRoutes(rg *gin.RouterGroup, rbacMW ...func(string) gin.HandlerFunc) {
	var readMW, manageMW gin.HandlerFunc
	if len(rbacMW) > 0 && rbacMW[0] != nil {
		readMW = rbacMW[0]("roles:read")
		manageMW = rbacMW[0]("roles:manage")
	}

	if readMW != nil {
		rg.GET("/permissions", readMW, h.Permissions)
	} else {
		rg.GET("/permissions", h.Permissions)
	}

	roles := rg.Group("/roles")
	if readMW != nil {
		roles.GET("", readMW, h.ListRoles)
		roles.GET("/:id", readMW, h.GetRole)
	} else {
		roles.GET("", h.ListRoles)
		roles.GET("/:id", h.GetRole)
	}

	if manageMW != nil {
		roles.POST("", manageMW, h.CreateRole)
		roles.PUT("/:id", manageMW, h.UpdateRole)
		roles.DELETE("/:id", manageMW, h.DeleteRole)
	} else {
		roles.POST("", h.CreateRole)
		roles.PUT("/:id", h.UpdateRole)
		roles.DELETE("/:id", h.DeleteRole)
	}
}

// RegisterTokenRoutes mounts personal access token management endpoints with optional RBAC middlewares.
func (h *Handler) RegisterTokenRoutes(rg *gin.RouterGroup, rbacMW ...func(string) gin.HandlerFunc) {
	var readMW, manageMW gin.HandlerFunc
	if len(rbacMW) > 0 && rbacMW[0] != nil {
		readMW = rbacMW[0]("tokens:read")
		manageMW = rbacMW[0]("tokens:manage")
	}

	tokens := rg.Group("/tokens")
	if readMW != nil {
		tokens.GET("", readMW, h.ListTokens)
	} else {
		tokens.GET("", h.ListTokens)
	}

	if manageMW != nil {
		tokens.POST("", manageMW, h.CreateToken)
		tokens.DELETE("/:id", manageMW, h.DeleteToken)
	} else {
		tokens.POST("", h.CreateToken)
		tokens.DELETE("/:id", h.DeleteToken)
	}
}

// RegisterRoutes mounts all authentication, role, and token endpoints under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	h.RegisterPublicRoutes(rg)
	h.RegisterRoleRoutes(rg)
	h.RegisterTokenRoutes(rg)
}
