package organizations

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the Organizations endpoints under rg, protected by
// sessionMW.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, sessionMW gin.HandlerFunc) {
	protected := rg.Group("", sessionMW)

	protected.GET("/users", h.ListUsers)
}
