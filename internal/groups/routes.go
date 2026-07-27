package groups

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes wires the /groups endpoints under rg, protected by
// sessionMW (RF-GRP requires an authenticated user; mutations additionally
// require the caller to be the organization's Platform Admin, enforced in
// the handlers themselves via userOrgContext).
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, sessionMW gin.HandlerFunc) {
	g := rg.Group("/groups", sessionMW)

	g.POST("", h.CreateGroup)
	g.GET("", h.ListGroups)
	g.GET("/:id", h.GetGroup)
	g.PATCH("/:id", h.RenameGroup)
	g.DELETE("/:id", h.DeleteGroup)

	g.POST("/:id/members", h.AddMember)
	g.PATCH("/:id/members/:userId", h.ChangeMemberRole)
	g.DELETE("/:id/members/:userId", h.RemoveMember)

	g.POST("/:id/spaces", h.GrantSpaceAccess)
	g.DELETE("/:id/spaces/:spaceId", h.RevokeSpaceAccess)

	g.POST("/:id/pages", h.GrantPageAccess)
	g.DELETE("/:id/pages/:pageId", h.RevokePageAccess)
}
