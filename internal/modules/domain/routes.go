package domain

import "github.com/gin-gonic/gin"

// RegisterRoutes registers custom domain routes under the provided Gin RouterGroup.
func (h *Handler) RegisterRoutes(rg *gin.RouterGroup, rbacMW ...func(string) gin.HandlerFunc) {
	var readMW, writeMW gin.HandlerFunc
	if len(rbacMW) > 0 && rbacMW[0] != nil {
		readMW = rbacMW[0]("domains:read")
		writeMW = rbacMW[0]("domains:write")
	}

	wrap := func(mw gin.HandlerFunc, handler gin.HandlerFunc) []gin.HandlerFunc {
		if mw != nil {
			return []gin.HandlerFunc{mw, handler}
		}
		return []gin.HandlerFunc{handler}
	}

	doms := rg.Group("/domains")
	{
		doms.GET("", append(wrap(readMW, h.List))...)
		doms.POST("", append(wrap(writeMW, h.Create))...)
		doms.GET("/:id", append(wrap(readMW, h.GetByID))...)
		doms.DELETE("/:id", append(wrap(writeMW, h.Delete))...)
	}
}
