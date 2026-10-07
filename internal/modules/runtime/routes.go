package runtime

import "github.com/gin-gonic/gin"

// RegisterRoutes registers runtime routes under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, rbacMW ...func(string) gin.HandlerFunc) {
	var readMW, writeMW gin.HandlerFunc
	if len(rbacMW) > 0 && rbacMW[0] != nil {
		readMW = rbacMW[0]("nodes:read")
		writeMW = rbacMW[0]("nodes:write")
	}

	wrap := func(mw gin.HandlerFunc, handler gin.HandlerFunc) []gin.HandlerFunc {
		if mw != nil {
			return []gin.HandlerFunc{mw, handler}
		}
		return []gin.HandlerFunc{handler}
	}

	rt := rg.Group("/runtime")
	{
		rt.GET("/status", append(wrap(readMW, h.Status))...)
		rt.POST("/upgrade", append(wrap(writeMW, h.Upgrade))...)
	}
}
