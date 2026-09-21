package runtime

import "github.com/gin-gonic/gin"

// RegisterRoutes registers runtime routes under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	rt := rg.Group("/runtime")
	{
		rt.GET("/status", h.Status)
		rt.POST("/upgrade", h.Upgrade)
	}
}
