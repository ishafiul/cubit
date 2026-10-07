package application

import "github.com/gin-gonic/gin"

// RegisterRoutes registers application endpoints under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, rbacMW ...func(string) gin.HandlerFunc) {
	var getMW, createMW, updateMW, deleteMW gin.HandlerFunc
	if len(rbacMW) > 0 && rbacMW[0] != nil {
		getMW = rbacMW[0]("apps:read")
		createMW = rbacMW[0]("apps:create")
		updateMW = rbacMW[0]("apps:update")
		deleteMW = rbacMW[0]("apps:delete")
	}

	wrap := func(mw gin.HandlerFunc, handler gin.HandlerFunc) []gin.HandlerFunc {
		if mw != nil {
			return []gin.HandlerFunc{mw, handler}
		}
		return []gin.HandlerFunc{handler}
	}

	apps := rg.Group("/applications")
	{
		apps.GET("", append(wrap(getMW, h.List))...)
		apps.POST("", append(wrap(createMW, h.Create))...)
		apps.GET("/:id", append(wrap(getMW, h.GetByID))...)
		apps.PUT("/:id", append(wrap(updateMW, h.Update))...)
		apps.DELETE("/:id", append(wrap(deleteMW, h.Delete))...)
		apps.POST("/:id/test", append(wrap(updateMW, h.Test))...)
		apps.POST("/:id/wrangler/import", append(wrap(updateMW, h.ImportWranglerConfig))...)
		apps.GET("/:id/bundle", append(wrap(getMW, h.GetBundle))...)
		apps.GET("/:id/assets/*filepath", append(wrap(getMW, h.GetAsset))...)
		apps.GET("/:id/metrics", append(wrap(getMW, h.GetMetrics))...)
		apps.GET("/:id/logs/stream", append(wrap(getMW, h.StreamLiveLogs))...)
	}
}
