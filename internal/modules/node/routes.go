package node

import "github.com/gin-gonic/gin"

// RegisterRoutes registers fleet node routes under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	nodes := rg.Group("/nodes")
	{
		nodes.GET("", h.List)
		nodes.POST("", h.Create)
		nodes.GET("/:id", h.GetByID)
		nodes.DELETE("/:id", h.Delete)
		nodes.POST("/:id/drain", h.Drain)
	}
}
