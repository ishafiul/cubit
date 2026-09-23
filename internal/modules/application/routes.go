package application

import "github.com/gin-gonic/gin"

// RegisterRoutes registers application endpoints under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	apps := rg.Group("/applications")
	{
		apps.GET("", h.List)
		apps.POST("", h.Create)
		apps.GET("/:id", h.GetByID)
		apps.PUT("/:id", h.Update)
		apps.DELETE("/:id", h.Delete)
		apps.POST("/:id/test", h.Test)
		apps.POST("/:id/wrangler/import", h.ImportWranglerConfig)
		apps.GET("/:id/bundle", h.GetBundle)
		apps.GET("/:id/assets/*filepath", h.GetAsset)
		apps.GET("/:id/metrics", h.GetMetrics)
		apps.GET("/:id/logs/stream", h.StreamLiveLogs)
	}
}
