package node

import "github.com/gin-gonic/gin"

// RegisterRoutes registers fleet node routes under the provided Gin RouterGroup.
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

	nodes := rg.Group("/nodes")
	{
		nodes.GET("", append(wrap(readMW, h.List))...)
		nodes.POST("", append(wrap(writeMW, h.Create))...)
		nodes.GET("/:id", append(wrap(readMW, h.GetByID))...)
		nodes.DELETE("/:id", append(wrap(writeMW, h.Delete))...)
		nodes.POST("/:id/drain", append(wrap(writeMW, h.Drain))...)
		nodes.POST("/:id/activate", append(wrap(writeMW, h.Activate))...)
	}
}
