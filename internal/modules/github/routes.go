package github

import "github.com/gin-gonic/gin"

// RegisterRoutes registers GitHub integration endpoints under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	gh := rg.Group("/github")
	{
		gh.GET("/settings", h.GetSettings)
		gh.POST("/settings", h.SaveSettings)
		gh.DELETE("/settings", h.ClearSettings)
		gh.GET("/manifest", h.GetManifest)
		gh.GET("/manifest/callback", h.ManifestCallback)
		gh.POST("/manifest/exchange", h.ExchangeManifest)
		gh.GET("/repositories", h.ListRepositories)
		gh.GET("/repositories/:owner/:repo/branches", h.ListBranches)
		gh.POST("/webhook", h.HandleWebhook)
	}
}
