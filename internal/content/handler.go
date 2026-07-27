package content

import (
	"net/http"

	"arkivy-api/internal/groups"
	"arkivy-api/internal/organizations"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Handler struct {
	db     *mongo.Database
	bucket *mongo.GridFSBucket
}

func NewHandler(db *mongo.Database) *Handler {
	return &Handler{db: db, bucket: db.GridFSBucket()}
}

// callerContext resolves the caller's organization and Platform Admin status
// from the userID SessionMiddleware injected into the Gin context. On
// failure it writes the HTTP response itself and returns ok=false.
func (h *Handler) callerContext(c *gin.Context) (userID, orgID string, isAdmin bool, ok bool) {
	userID = c.GetString("userID")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session not provided"})
		return "", "", false, false
	}

	user, _, err := organizations.GetUserWithOrganization(c.Request.Context(), h.db, userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User has no organization"})
		return "", "", false, false
	}

	return userID, user.OrganizationID, user.IsPlatformAdmin, true
}

// requireSpaceAccess checks the caller has at least min access to space,
// writing a 404 (not 403 — content is private by default, we don't reveal
// existence) if not.
func (h *Handler) requireSpaceAccess(c *gin.Context, userID string, isAdmin bool, spaceID string, min groups.AccessLevel) bool {
	access := SpaceAccess(c.Request.Context(), h.db, userID, isAdmin, spaceID)
	if access == nil || !meetsLevel(*access, min) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Space not found"})
		return false
	}
	return true
}

// requirePageAccess is requireSpaceAccess's Page equivalent.
func (h *Handler) requirePageAccess(c *gin.Context, userID string, isAdmin bool, page Page, min groups.AccessLevel) bool {
	access := PageAccess(c.Request.Context(), h.db, userID, isAdmin, page)
	if access == nil || !meetsLevel(*access, min) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
		return false
	}
	return true
}

// ── Spaces ──────────────────────────────────────────────────────────────

// CreateSpace handles POST /spaces
// @Summary Create a space (RF-DOC-01)
// @Tags Content
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 201 {object} Space
// @Failure 403 {object} map[string]string
// @Router /spaces [post]
func (h *Handler) CreateSpace(c *gin.Context) {
	_, orgID, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the organization's Platform Admin can create spaces"})
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	space, err := CreateSpace(c.Request.Context(), h.db, orgID, req.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating space"})
		return
	}
	c.JSON(http.StatusCreated, space)
}

// ListSpaces handles GET /spaces
// @Summary List spaces visible to the caller
// @Tags Content
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {array} Space
// @Router /spaces [get]
func (h *Handler) ListSpaces(c *gin.Context) {
	userID, orgID, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	all, err := ListSpacesByOrg(c.Request.Context(), h.db, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listing spaces"})
		return
	}

	visible := []Space{}
	for _, s := range all {
		if isAdmin {
			visible = append(visible, s)
			continue
		}
		if access := SpaceAccess(c.Request.Context(), h.db, userID, isAdmin, s.ID); access != nil {
			visible = append(visible, s)
		}
	}
	c.JSON(http.StatusOK, visible)
}

// GetSpace handles GET /spaces/:id
// @Summary Get a space's detail
// @Tags Content
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} Space
// @Failure 404 {object} map[string]string
// @Router /spaces/{id} [get]
func (h *Handler) GetSpace(c *gin.Context) {
	userID, orgID, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	space, err := GetSpace(c.Request.Context(), h.db, orgID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Space not found"})
		return
	}
	if !h.requireSpaceAccess(c, userID, isAdmin, space.ID, groups.AccessReader) {
		return
	}
	c.JSON(http.StatusOK, space)
}

// RenameSpace handles PATCH /spaces/:id
// @Summary Rename a space
// @Tags Content
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /spaces/{id} [patch]
func (h *Handler) RenameSpace(c *gin.Context) {
	userID, orgID, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	space, err := GetSpace(c.Request.Context(), h.db, orgID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Space not found"})
		return
	}
	if !h.requireSpaceAccess(c, userID, isAdmin, space.ID, groups.AccessEditor) {
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
		return
	}

	if err := RenameSpace(c.Request.Context(), h.db, orgID, space.ID, req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error renaming space"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Space renamed"})
}

// DeleteSpace handles DELETE /spaces/:id
// @Summary Delete a space, cascading its pages/blocks/attachments (RF-DOC-06)
// @Tags Content
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /spaces/{id} [delete]
func (h *Handler) DeleteSpace(c *gin.Context) {
	_, orgID, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}
	if !isAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only the organization's Platform Admin can delete spaces"})
		return
	}

	if err := DeleteSpace(c.Request.Context(), h.db, h.bucket, orgID, c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Space not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Space deleted"})
}

// ── Pages ───────────────────────────────────────────────────────────────

// CreatePage handles POST /spaces/:id/pages
// @Summary Create a page in a space (RF-DOC-03)
// @Tags Content
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 201 {object} Page
// @Failure 404 {object} map[string]string
// @Router /spaces/{id}/pages [post]
func (h *Handler) CreatePage(c *gin.Context) {
	userID, orgID, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	space, err := GetSpace(c.Request.Context(), h.db, orgID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Space not found"})
		return
	}
	if !h.requireSpaceAccess(c, userID, isAdmin, space.ID, groups.AccessEditor) {
		return
	}

	var req struct {
		Category string `json:"category" binding:"required"`
		Title    string `json:"title" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category and title are required"})
		return
	}
	category, ok := parseCategory(req.Category)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
		return
	}

	page, err := CreatePage(c.Request.Context(), h.db, orgID, space.ID, category, req.Title, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating page"})
		return
	}
	c.JSON(http.StatusCreated, page)
}

// ListPages handles GET /spaces/:id/pages
// @Summary List a space's pages grouped by category (RF-DOC-02, table of contents)
// @Tags Content
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {array} TocCategory
// @Failure 404 {object} map[string]string
// @Router /spaces/{id}/pages [get]
func (h *Handler) ListPages(c *gin.Context) {
	userID, orgID, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	space, err := GetSpace(c.Request.Context(), h.db, orgID, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Space not found"})
		return
	}

	toc, err := ListPagesBySpace(c.Request.Context(), h.db, space.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error listing pages"})
		return
	}

	if !isAdmin {
		for i := range toc {
			visible := []Page{}
			for _, p := range toc[i].Pages {
				if access := PageAccess(c.Request.Context(), h.db, userID, isAdmin, p); access != nil {
					visible = append(visible, p)
				}
			}
			toc[i].Pages = visible
		}
	}

	c.JSON(http.StatusOK, toc)
}

// GetPage handles GET /pages/:id
// @Summary Get a page's detail with its blocks
// @Tags Content
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]any
// @Failure 404 {object} map[string]string
// @Router /pages/{id} [get]
func (h *Handler) GetPage(c *gin.Context) {
	userID, _, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	page, err := GetPage(c.Request.Context(), h.db, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
		return
	}
	if !h.requirePageAccess(c, userID, isAdmin, *page, groups.AccessReader) {
		return
	}

	blocks, err := ListBlocks(c.Request.Context(), h.db, page.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error loading blocks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"page": page, "blocks": blocks})
}

// UpdatePage handles PATCH /pages/:id
// @Summary Update a page's title and/or category
// @Tags Content
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /pages/{id} [patch]
func (h *Handler) UpdatePage(c *gin.Context) {
	userID, _, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	page, err := GetPage(c.Request.Context(), h.db, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
		return
	}
	if !h.requirePageAccess(c, userID, isAdmin, *page, groups.AccessEditor) {
		return
	}

	var req struct {
		Title    *string `json:"title"`
		Category *string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	var category *Category
	if req.Category != nil {
		parsed, ok := parseCategory(*req.Category)
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid category"})
			return
		}
		category = &parsed
	}

	if err := UpdatePage(c.Request.Context(), h.db, page.ID, req.Title, category); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error updating page"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page updated"})
}

// ReorderPage handles PATCH /pages/:id/order
// @Summary Reorder a page within its category (RF-DOC-03b)
// @Tags Content
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /pages/{id}/order [patch]
func (h *Handler) ReorderPage(c *gin.Context) {
	userID, _, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	page, err := GetPage(c.Request.Context(), h.db, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
		return
	}
	if !h.requirePageAccess(c, userID, isAdmin, *page, groups.AccessEditor) {
		return
	}

	var req struct {
		NewIndex int `json:"newIndex"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "newIndex is required"})
		return
	}

	if err := ReorderPage(c.Request.Context(), h.db, page.ID, req.NewIndex); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error reordering page"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page reordered"})
}

// ReplaceBlocks handles PUT /pages/:id/blocks
// @Summary Replace a page's full ordered block list (RF-DOC-04/05)
// @Tags Content
// @Accept json
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /pages/{id}/blocks [put]
func (h *Handler) ReplaceBlocks(c *gin.Context) {
	userID, _, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	page, err := GetPage(c.Request.Context(), h.db, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
		return
	}
	if !h.requirePageAccess(c, userID, isAdmin, *page, groups.AccessEditor) {
		return
	}

	var req struct {
		Blocks []Block `json:"blocks"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid blocks payload"})
		return
	}

	if err := ReplaceBlocks(c.Request.Context(), h.db, page.ID, req.Blocks); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Blocks saved"})
}

// DeletePage handles DELETE /pages/:id
// @Summary Delete a page, cascading its blocks/attachments (RF-DOC-06)
// @Tags Content
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /pages/{id} [delete]
func (h *Handler) DeletePage(c *gin.Context) {
	userID, _, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	page, err := GetPage(c.Request.Context(), h.db, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Page not found"})
		return
	}
	if !h.requirePageAccess(c, userID, isAdmin, *page, groups.AccessEditor) {
		return
	}

	if err := DeletePage(c.Request.Context(), h.db, h.bucket, page.ID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting page"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Page deleted"})
}

// ── Attachments ─────────────────────────────────────────────────────────

// UploadAttachment handles POST /blocks/:id/attachments
// @Summary Upload a file to a file-type block (RF-DOC-11)
// @Tags Content
// @Accept multipart/form-data
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 201 {object} Attachment
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /blocks/{id}/attachments [post]
func (h *Handler) UploadAttachment(c *gin.Context) {
	userID, _, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	blockID := c.Param("id")
	pageID, err := FindPageIDByBlockID(c.Request.Context(), h.db, blockID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found"})
		return
	}
	page, err := GetPage(c.Request.Context(), h.db, pageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Block not found"})
		return
	}
	if !h.requirePageAccess(c, userID, isAdmin, *page, groups.AccessEditor) {
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error reading file"})
		return
	}
	defer f.Close()

	attachment, err := UploadAttachment(c.Request.Context(), h.db, h.bucket, page.OrganizationID, blockID, fileHeader.Filename, fileHeader.Size, userID, f)
	if err == ErrUnsupportedFileType || err == ErrFileTooLarge {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error uploading attachment"})
		return
	}
	c.JSON(http.StatusCreated, attachment)
}

// DownloadAttachment handles GET /attachments/:id
// @Summary Download an attachment
// @Tags Content
// @Produce application/octet-stream
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {file} file
// @Failure 404 {object} map[string]string
// @Router /attachments/{id} [get]
func (h *Handler) DownloadAttachment(c *gin.Context) {
	userID, _, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	attachmentID := c.Param("id")
	pageID, err := h.pageIDForAttachment(c, attachmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}
	page, err := GetPage(c.Request.Context(), h.db, pageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}
	if !h.requirePageAccess(c, userID, isAdmin, *page, groups.AccessReader) {
		return
	}

	attachment, stream, err := OpenAttachmentDownload(c.Request.Context(), h.db, h.bucket, attachmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}
	defer stream.Close()

	c.Header("Content-Disposition", "attachment; filename=\""+attachment.FileName+"\"")
	c.DataFromReader(http.StatusOK, attachment.FileSizeBytes, "application/octet-stream", stream, nil)
}

// RevokeAttachment handles DELETE /attachments/:id
// @Summary Delete an attachment
// @Tags Content
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /attachments/{id} [delete]
func (h *Handler) RevokeAttachment(c *gin.Context) {
	userID, _, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	attachmentID := c.Param("id")
	pageID, err := h.pageIDForAttachment(c, attachmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}
	page, err := GetPage(c.Request.Context(), h.db, pageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Attachment not found"})
		return
	}
	if !h.requirePageAccess(c, userID, isAdmin, *page, groups.AccessEditor) {
		return
	}

	if err := DeleteAttachment(c.Request.Context(), h.db, h.bucket, attachmentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error deleting attachment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Attachment deleted"})
}

func (h *Handler) pageIDForAttachment(c *gin.Context, attachmentID string) (string, error) {
	a, err := GetAttachment(c.Request.Context(), h.db, attachmentID)
	if err != nil {
		return "", err
	}
	return FindPageIDByBlockID(c.Request.Context(), h.db, a.BlockID)
}

// ── Search ──────────────────────────────────────────────────────────────

// Search handles GET /search
// @Summary Search pages by title/content within the organization (RF-DOC-07)
// @Tags Content
// @Produce json
// @Param q query string true "Search query"
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {array} Page
// @Router /search [get]
func (h *Handler) Search(c *gin.Context) {
	userID, orgID, isAdmin, ok := h.callerContext(c)
	if !ok {
		return
	}

	results, err := SearchPages(c.Request.Context(), h.db, orgID, c.Query("q"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error searching"})
		return
	}

	visible := []Page{}
	for _, p := range results {
		if access := PageAccess(c.Request.Context(), h.db, userID, isAdmin, p); access != nil {
			visible = append(visible, p)
		}
	}
	c.JSON(http.StatusOK, visible)
}

func parseCategory(raw string) (Category, bool) {
	switch Category(raw) {
	case CategoryTutorial, CategoryHowTo, CategoryReference, CategoryExplanation, CategoryMultiple:
		return Category(raw), true
	default:
		return "", false
	}
}
