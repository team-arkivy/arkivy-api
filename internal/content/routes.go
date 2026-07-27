package content

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the Content endpoints under rg, protected by
// sessionMW. Fine-grained access (Platform Admin vs. Group-derived
// editor/reader) is enforced inside the handlers via callerContext/
// requireSpaceAccess/requirePageAccess.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, sessionMW gin.HandlerFunc) {
	protected := rg.Group("", sessionMW)

	protected.POST("/spaces", h.CreateSpace)
	protected.GET("/spaces", h.ListSpaces)
	protected.GET("/spaces/:id", h.GetSpace)
	protected.PATCH("/spaces/:id", h.RenameSpace)
	protected.DELETE("/spaces/:id", h.DeleteSpace)

	protected.POST("/spaces/:id/pages", h.CreatePage)
	protected.GET("/spaces/:id/pages", h.ListPages)

	protected.GET("/pages/:id", h.GetPage)
	protected.PATCH("/pages/:id", h.UpdatePage)
	protected.PATCH("/pages/:id/order", h.ReorderPage)
	protected.PUT("/pages/:id/blocks", h.ReplaceBlocks)
	protected.DELETE("/pages/:id", h.DeletePage)

	protected.POST("/blocks/:id/attachments", h.UploadAttachment)
	protected.GET("/attachments/:id", h.DownloadAttachment)
	protected.DELETE("/attachments/:id", h.RevokeAttachment)

	protected.GET("/search", h.Search)
}
