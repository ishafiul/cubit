package deployment

import "github.com/gin-gonic/gin"

// RegisterRoutes registers deployment routes under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, rbacMW ...func(string) gin.HandlerFunc) {
	var deployMW, rollbackMW, readMW gin.HandlerFunc
	if len(rbacMW) > 0 && rbacMW[0] != nil {
		deployMW = rbacMW[0]("apps:deploy")
		rollbackMW = rbacMW[0]("deployments:rollback")
		readMW = rbacMW[0]("deployments:read")
	}

	wrap := func(mw gin.HandlerFunc, handler gin.HandlerFunc) []gin.HandlerFunc {
		if mw != nil {
			return []gin.HandlerFunc{mw, handler}
		}
		return []gin.HandlerFunc{handler}
	}

	// Under /applications/:id
	apps := rg.Group("/applications")
	{
		apps.POST("/:id/deploy", append(wrap(deployMW, h.Deploy))...)
		apps.POST("/:id/deploy/direct", append(wrap(deployMW, h.DeployDirect))...)
		apps.POST("/:id/rollback", append(wrap(rollbackMW, h.Rollback))...)
		apps.GET("/:id/deployments", append(wrap(readMW, h.ListByApp))...)
	}

	// Under /deployments
	deps := rg.Group("/deployments")
	{
		deps.GET("/:id", append(wrap(readMW, h.GetByID))...)
		deps.GET("/:id/logs", append(wrap(readMW, h.GetLogs))...)
		deps.GET("/:id/logs/stream", append(wrap(readMW, h.StreamLogs))...)
	}
}
