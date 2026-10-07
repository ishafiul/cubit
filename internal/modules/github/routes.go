package github

import "github.com/gin-gonic/gin"

// RegisterPublicRoutes registers unauthenticated GitHub webhooks and callbacks.
func (h *Handler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	gh := rg.Group("/github")
	{
		gh.GET("/manifest/callback", h.ManifestCallback)
		gh.POST("/webhook", h.HandleWebhook)
	}
}

// RegisterRoutes registers GitHub integration endpoints under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, rbacMW ...func(string) gin.HandlerFunc) {
	var readMW, updateMW gin.HandlerFunc
	if len(rbacMW) > 0 && rbacMW[0] != nil {
		readMW = rbacMW[0]("apps:read")
		updateMW = rbacMW[0]("apps:update")
	}

	wrap := func(mw gin.HandlerFunc, handler gin.HandlerFunc) []gin.HandlerFunc {
		if mw != nil {
			return []gin.HandlerFunc{mw, handler}
		}
		return []gin.HandlerFunc{handler}
	}

	gh := rg.Group("/github")
	{
		gh.GET("/settings", append(wrap(readMW, h.GetSettings))...)
		gh.POST("/settings", append(wrap(updateMW, h.SaveSettings))...)
		gh.DELETE("/settings", append(wrap(updateMW, h.ClearSettings))...)
		gh.GET("/manifest", append(wrap(readMW, h.GetManifest))...)
		gh.POST("/manifest/exchange", append(wrap(updateMW, h.ExchangeManifest))...)
		gh.GET("/sync", append(wrap(readMW, h.SyncInstallations))...)
		gh.POST("/sync", append(wrap(updateMW, h.SyncInstallations))...)
		gh.GET("/repositories", append(wrap(readMW, h.ListRepositories))...)
		gh.GET("/repositories/:owner/:repo/branches", append(wrap(readMW, h.ListBranches))...)
		gh.GET("/repositories/:owner/:repo/folders", append(wrap(readMW, h.ListFolders))...)

		// If called without RBAC (e.g. standalone in module tests), register webhook and callback here as well
		if len(rbacMW) == 0 {
			gh.GET("/manifest/callback", h.ManifestCallback)
			gh.POST("/webhook", h.HandleWebhook)
		}
	}
}
