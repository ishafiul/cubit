package service

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts all 12 celld service endpoints under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup) {
	// 1. KV Namespaces & Values
	kv := rg.Group("/kv")
	{
		kv.GET("/namespaces", h.ListKVNamespaces)
		kv.POST("/namespaces", h.CreateKVNamespace)
		kv.DELETE("/namespaces/:id", h.DeleteKVNamespace)
		kv.GET("/namespaces/:id/keys", h.ListKVPairs)
		kv.GET("/namespaces/:id/values/:key", h.GetKVPair)
		kv.PUT("/namespaces/:id/values/:key", h.PutKVPair)
		kv.DELETE("/namespaces/:id/values/:key", h.DeleteKVPair)
	}

	// 2. D1 SQL Databases
	d1 := rg.Group("/d1")
	{
		d1.GET("/databases", h.ListD1Databases)
		d1.POST("/databases", h.CreateD1Database)
		d1.DELETE("/databases/:id", h.DeleteD1Database)
		d1.POST("/databases/:id/query", h.ExecuteD1Query)
	}

	// 3. R2 Object Storage
	r2 := rg.Group("/r2")
	{
		r2.GET("/buckets", h.ListR2Buckets)
		r2.POST("/buckets", h.CreateR2Bucket)
		r2.DELETE("/buckets/:name", h.DeleteR2Bucket)
		r2.GET("/buckets/:name/objects", h.ListR2Objects)
		r2.POST("/buckets/:name/upload", h.UploadR2Object)
	}

	// 4. Queues
	queues := rg.Group("/queues")
	{
		queues.GET("", h.ListQueues)
		queues.POST("", h.CreateQueue)
		queues.DELETE("/:id", h.DeleteQueue)
		queues.POST("/:id/messages", h.SendQueueMessage)
		queues.GET("/:id/messages", h.ListQueueMessages)
	}

	// 5. Cron Triggers
	cron := rg.Group("/cron")
	{
		cron.GET("", h.ListCronTriggers)
		cron.POST("", h.CreateCronTrigger)
		cron.DELETE("/:id", h.DeleteCronTrigger)
		cron.POST("/:id/run", h.RunTriggerNow)
		cron.GET("/:id/runs", h.ListCronRuns)
	}

	// 6. Workflows
	workflows := rg.Group("/workflows")
	{
		workflows.GET("", h.ListWorkflows)
		workflows.POST("", h.CreateWorkflow)
		workflows.DELETE("/:id", h.DeleteWorkflow)
		workflows.POST("/:id/trigger", h.TriggerWorkflowRun)
		workflows.GET("/:id/runs", h.ListWorkflowRuns)
	}

	// 7. Durable Objects
	do := rg.Group("/durable-objects")
	{
		do.GET("", h.ListDurableObjectClasses)
		do.POST("", h.CreateDurableObjectClass)
		do.DELETE("/:id", h.DeleteDurableObjectClass)
		do.GET("/:id/instances", h.ListDOInstances)
		do.POST("/:id/instances", h.CreateDOInstance)
	}

	// 8. Containers
	containers := rg.Group("/containers")
	{
		containers.GET("", h.ListContainers)
		containers.POST("", h.CreateContainer)
		containers.DELETE("/:id", h.DeleteContainer)
	}

	// 9. Static Assets
	staticAssets := rg.Group("/static-assets")
	{
		staticAssets.GET("", h.ListStaticSites)
		staticAssets.POST("", h.CreateStaticSite)
		staticAssets.DELETE("/:id", h.DeleteStaticSite)
	}

	// 10. Dynamic Workers
	dynamicWorkers := rg.Group("/dynamic-workers")
	{
		dynamicWorkers.POST("/eval", h.EvalDynamicWorker)
	}

}
