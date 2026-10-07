package service

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts all 12 celld service endpoints under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, rbacMW ...func(string) gin.HandlerFunc) {
	var readMW, writeMW gin.HandlerFunc
	if len(rbacMW) > 0 && rbacMW[0] != nil {
		readMW = rbacMW[0]("services:read")
		writeMW = rbacMW[0]("services:*")
	}

	wrap := func(mw gin.HandlerFunc, handler gin.HandlerFunc) []gin.HandlerFunc {
		if mw != nil {
			return []gin.HandlerFunc{mw, handler}
		}
		return []gin.HandlerFunc{handler}
	}

	// 1. KV Namespaces & Values
	kv := rg.Group("/kv")
	{
		kv.GET("/namespaces", append(wrap(readMW, h.ListKVNamespaces))...)
		kv.POST("/namespaces", append(wrap(writeMW, h.CreateKVNamespace))...)
		kv.DELETE("/namespaces/:id", append(wrap(writeMW, h.DeleteKVNamespace))...)
		kv.GET("/namespaces/:id/keys", append(wrap(readMW, h.ListKVPairs))...)
		kv.GET("/namespaces/:id/values/:key", append(wrap(readMW, h.GetKVPair))...)
		kv.PUT("/namespaces/:id/values/:key", append(wrap(writeMW, h.PutKVPair))...)
		kv.DELETE("/namespaces/:id/values/:key", append(wrap(writeMW, h.DeleteKVPair))...)
		kv.POST("/namespaces/:id/import", append(wrap(writeMW, h.ImportKVBulk))...)
	}

	// 2. D1 SQL Databases
	d1 := rg.Group("/d1")
	{
		d1.GET("/databases", append(wrap(readMW, h.ListD1Databases))...)
		d1.POST("/databases", append(wrap(writeMW, h.CreateD1Database))...)
		d1.DELETE("/databases/:id", append(wrap(writeMW, h.DeleteD1Database))...)
		d1.POST("/databases/:id/query", append(wrap(writeMW, h.ExecuteD1Query))...)
		d1.POST("/databases/:id/import", append(wrap(writeMW, h.ImportD1SQL))...)
	}

	// 3. R2 Object Storage
	r2 := rg.Group("/r2")
	{
		r2.GET("/buckets", append(wrap(readMW, h.ListR2Buckets))...)
		r2.POST("/buckets", append(wrap(writeMW, h.CreateR2Bucket))...)
		r2.DELETE("/buckets/:name", append(wrap(writeMW, h.DeleteR2Bucket))...)
		r2.GET("/buckets/:name/objects", append(wrap(readMW, h.ListR2Objects))...)
		r2.GET("/buckets/:name/objects/*key", append(wrap(readMW, h.GetR2Object))...)
		r2.DELETE("/buckets/:name/objects/*key", append(wrap(writeMW, h.DeleteR2Object))...)
		r2.POST("/buckets/:name/upload", append(wrap(writeMW, h.UploadR2Object))...)
		r2.POST("/buckets/:name/import", append(wrap(writeMW, h.ImportR2Objects))...)
	}

	// 4. Queues
	queues := rg.Group("/queues")
	{
		queues.GET("", append(wrap(readMW, h.ListQueues))...)
		queues.POST("", append(wrap(writeMW, h.CreateQueue))...)
		queues.DELETE("/:id", append(wrap(writeMW, h.DeleteQueue))...)
		queues.POST("/:id/messages", append(wrap(writeMW, h.SendQueueMessage))...)
		queues.GET("/:id/messages", append(wrap(readMW, h.ListQueueMessages))...)
	}

	// 5. Cron Triggers
	cron := rg.Group("/cron")
	{
		cron.GET("", append(wrap(readMW, h.ListCronTriggers))...)
		cron.POST("", append(wrap(writeMW, h.CreateCronTrigger))...)
		cron.DELETE("/:id", append(wrap(writeMW, h.DeleteCronTrigger))...)
		cron.POST("/:id/run", append(wrap(writeMW, h.RunTriggerNow))...)
		cron.GET("/:id/runs", append(wrap(readMW, h.ListCronRuns))...)
	}

	// 6. Workflows
	workflows := rg.Group("/workflows")
	{
		workflows.GET("", append(wrap(readMW, h.ListWorkflows))...)
		workflows.POST("", append(wrap(writeMW, h.CreateWorkflow))...)
		workflows.DELETE("/:id", append(wrap(writeMW, h.DeleteWorkflow))...)
		workflows.POST("/:id/trigger", append(wrap(writeMW, h.TriggerWorkflowRun))...)
		workflows.GET("/:id/runs", append(wrap(readMW, h.ListWorkflowRuns))...)
	}

	// 7. Durable Objects
	do := rg.Group("/durable-objects")
	{
		do.GET("", append(wrap(readMW, h.ListDurableObjectClasses))...)
		do.POST("", append(wrap(writeMW, h.CreateDurableObjectClass))...)
		do.DELETE("/:id", append(wrap(writeMW, h.DeleteDurableObjectClass))...)
		do.GET("/:id/instances", append(wrap(readMW, h.ListDOInstances))...)
		do.POST("/:id/instances", append(wrap(writeMW, h.CreateDOInstance))...)
	}

	// 8. Containers
	containers := rg.Group("/containers")
	{
		containers.GET("", append(wrap(readMW, h.ListContainers))...)
		containers.POST("", append(wrap(writeMW, h.CreateContainer))...)
		containers.DELETE("/:id", append(wrap(writeMW, h.DeleteContainer))...)
	}

	// 9. Static Assets
	staticAssets := rg.Group("/static-assets")
	{
		staticAssets.GET("", append(wrap(readMW, h.ListStaticSites))...)
		staticAssets.POST("", append(wrap(writeMW, h.CreateStaticSite))...)
		staticAssets.DELETE("/:id", append(wrap(writeMW, h.DeleteStaticSite))...)
	}

	// 10. Dynamic Workers
	dynamicWorkers := rg.Group("/dynamic-workers")
	{
		dynamicWorkers.POST("/eval", append(wrap(writeMW, h.EvalDynamicWorker))...)
	}
}
