package deployment

import "github.com/gin-gonic/gin"

// RegisterRoutes registers deployment routes under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	// Under /applications/:id
	apps := rg.Group("/applications")
	{
		apps.POST("/:id/deploy", h.Deploy)
		apps.POST("/:id/rollback", h.Rollback)
		apps.GET("/:id/deployments", h.ListByApp)
	}

	// Under /deployments
	deps := rg.Group("/deployments")
	{
		deps.GET("/:id", h.GetByID)
		deps.GET("/:id/logs", h.GetLogs)
		deps.GET("/:id/logs/stream", h.StreamLogs)
	}
}
