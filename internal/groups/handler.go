package groups

import (
	"net/http"

	"arkivy-api/internal/organizations"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Handler struct {
	db *mongo.Database
}

func NewHandler(db *mongo.Database) *Handler {
	return &Handler{db: db}
}

// userOrgContext resolves the caller's organization from the userID that
// SessionMiddleware injected into the Gin context. requireAdmin additionally
// checks RF-GRP-01/06 (only the organization's Platform Admin manages
// Groups). On failure it writes the HTTP response itself and returns ok=false.
func (h *Handler) userOrgContext(c *gin.Context, requireAdmin bool) (userID, orgID string, ok bool) {
	userID = c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session not provided"})
		return "", "", false
	}

	user, _, err := organizations.GetUserWithOrganization(c.Request.Context(), h.db, userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User has no organization"})
		return "", "", false
	}

	if requireAdmin && !user.IsPlatformAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the organization's Platform Admin can manage groups"})
		return "", "", false
	}

	return userID, user.OrganizationID, true
}

func parseAccessLevel(raw string) (AccessLevel, bool) {
	switch AccessLevel(raw) {
	case AccessEditor, AccessReader:
		return AccessLevel(raw), true
	default:
		return "", false
	}
}

// CreateGroup handles POST /groups
// @Summary Create a group
// @Tags Groups
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 201 {object} Group
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /groups [post]
func (h *Handler) CreateGroup(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	group, err := CreateGroup(c.Request.Context(), h.db, orgID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating group"})
		return
	}

	c.JSON(http.StatusCreated, group)
}

// ListGroups handles GET /groups
// @Summary List groups in the caller's organization
// @Tags Groups
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {array} Group
// @Failure 401 {object} map[string]string
// @Router /groups [get]
func (h *Handler) ListGroups(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, false)
	if !ok {
		return
	}

	list, err := ListGroups(c.Request.Context(), h.db, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listing groups"})
		return
	}

	c.JSON(http.StatusOK, list)
}

// GetGroup handles GET /groups/:id
// @Summary Get a group's detail (members and access grants)
// @Tags Groups
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} GroupDetail
// @Failure 404 {object} map[string]string
// @Router /groups/{id} [get]
func (h *Handler) GetGroup(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, false)
	if !ok {
		return
	}

	detail, err := GetGroupDetail(c.Request.Context(), h.db, orgID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

// RenameGroup handles PATCH /groups/:id
// @Summary Rename a group
// @Tags Groups
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id} [patch]
func (h *Handler) RenameGroup(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if err := RenameGroup(c.Request.Context(), h.db, orgID, c.Param("id"), req.Name); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group renamed"})
}

// DeleteGroup handles DELETE /groups/:id
// @Summary Delete a group (RF-GRP-06)
// @Tags Groups
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id} [delete]
func (h *Handler) DeleteGroup(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	if err := DeleteGroup(c.Request.Context(), h.db, orgID, c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Group deleted"})
}

// AddMember handles POST /groups/:id/members
// @Summary Add a member to a group with a role (RF-GRP-03)
// @Tags Groups
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id}/members [post]
func (h *Handler) AddMember(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	groupID := c.Param("id")
	if _, err := GetGroup(c.Request.Context(), h.db, orgID, groupID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	var req struct {
		UserID string `json:"userId" binding:"required"`
		Role   string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId and role are required"})
		return
	}
	role, ok := parseAccessLevel(req.Role)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'editor' or 'reader'"})
		return
	}

	if err := UpsertMember(c.Request.Context(), h.db, groupID, req.UserID, role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error adding member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member added"})
}

// ChangeMemberRole handles PATCH /groups/:id/members/:userId
// @Summary Change a member's role within a group (RF-GRP-03)
// @Tags Groups
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id}/members/{userId} [patch]
func (h *Handler) ChangeMemberRole(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	groupID := c.Param("id")
	if _, err := GetGroup(c.Request.Context(), h.db, orgID, groupID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	var req struct {
		Role string `json:"role" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role is required"})
		return
	}
	role, ok := parseAccessLevel(req.Role)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "role must be 'editor' or 'reader'"})
		return
	}

	if err := UpsertMember(c.Request.Context(), h.db, groupID, c.Param("userId"), role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated"})
}

// RemoveMember handles DELETE /groups/:id/members/:userId
// @Summary Remove a member from a group (RF-GRP-06)
// @Tags Groups
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id}/members/{userId} [delete]
func (h *Handler) RemoveMember(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	groupID := c.Param("id")
	if _, err := GetGroup(c.Request.Context(), h.db, orgID, groupID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if err := RemoveMember(c.Request.Context(), h.db, groupID, c.Param("userId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error removing member"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Member removed"})
}

// GrantSpaceAccess handles POST /groups/:id/spaces
// @Summary Grant a group access to a space (RF-GRP-02)
// @Tags Groups
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id}/spaces [post]
func (h *Handler) GrantSpaceAccess(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	groupID := c.Param("id")
	if _, err := GetGroup(c.Request.Context(), h.db, orgID, groupID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	var req struct {
		SpaceID string `json:"spaceId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "spaceId is required"})
		return
	}

	if err := GrantSpaceAccess(c.Request.Context(), h.db, groupID, req.SpaceID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error granting space access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Space access granted"})
}

// RevokeSpaceAccess handles DELETE /groups/:id/spaces/:spaceId
// @Summary Revoke a group's access to a space (RF-GRP-06)
// @Tags Groups
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id}/spaces/{spaceId} [delete]
func (h *Handler) RevokeSpaceAccess(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	groupID := c.Param("id")
	if _, err := GetGroup(c.Request.Context(), h.db, orgID, groupID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if err := RevokeSpaceAccess(c.Request.Context(), h.db, groupID, c.Param("spaceId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error revoking space access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Space access revoked"})
}

// GrantPageAccess handles POST /groups/:id/pages
// @Summary Grant a group access to a loose page (RF-GRP-02)
// @Tags Groups
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id}/pages [post]
func (h *Handler) GrantPageAccess(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	groupID := c.Param("id")
	if _, err := GetGroup(c.Request.Context(), h.db, orgID, groupID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	var req struct {
		PageID string `json:"pageId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "pageId is required"})
		return
	}

	if err := GrantPageAccess(c.Request.Context(), h.db, groupID, req.PageID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error granting page access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Page access granted"})
}

// RevokePageAccess handles DELETE /groups/:id/pages/:pageId
// @Summary Revoke a group's access to a loose page (RF-GRP-06)
// @Tags Groups
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /groups/{id}/pages/{pageId} [delete]
func (h *Handler) RevokePageAccess(c *gin.Context) {
	_, orgID, ok := h.userOrgContext(c, true)
	if !ok {
		return
	}

	groupID := c.Param("id")
	if _, err := GetGroup(c.Request.Context(), h.db, orgID, groupID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Group not found"})
		return
	}

	if err := RevokePageAccess(c.Request.Context(), h.db, groupID, c.Param("pageId")); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error revoking page access"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Page access revoked"})
}
