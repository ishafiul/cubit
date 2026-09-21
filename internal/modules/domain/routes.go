package domain

import "github.com/gin-gonic/gin"

// RegisterRoutes registers custom domain routes under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	doms := rg.Group("/domains")
	{
		doms.GET("", h.List)
		doms.POST("", h.Create)
		doms.GET("/:id", h.GetByID)
		doms.DELETE("/:id", h.Delete)
	}
}
