package auth

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"arkivy-api/internal/zitadel"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Handler struct {
	zClient   *zitadel.Client
	db        *mongo.Database
	googleIDP string
	githubIDP string
	frontURL  string
}

func NewHandler(zClient *zitadel.Client, db *mongo.Database, googleIDP, githubIDP, frontURL string) *Handler {
	return &Handler{
		zClient:   zClient,
		db:        db,
		googleIDP: googleIDP,
		githubIDP: githubIDP,
		frontURL:  frontURL,
	}
}

// syncSysAdminRole checks if the user is in the sys_admins collection
// and syncs the role in Zitadel
func (h *Handler) syncSysAdminRole(ctx context.Context, userID, loginName string) {
	isSysAdmin := h.db.Collection("sys_admins").
		FindOne(ctx, bson.M{"email": loginName}).Err() == nil

	currentRoles, err := h.zClient.GetUserRoles(ctx, userID)
	if err != nil {
		return
	}

	if len(currentRoles) == 0 {
		if err := h.zClient.AssignRole(ctx, userID, zitadel.RoleInvited); err != nil {
			fmt.Printf("Error assigning invited role to %s: %v\n", loginName, err)
			return
		}
		currentRoles = []string{string(zitadel.RoleInvited)}
	}

	hasSysAdminRole := contains(currentRoles, string(zitadel.RoleSysAdmin))

	if isSysAdmin && !hasSysAdminRole {
		if err := h.zClient.AssignRole(ctx, userID, zitadel.RoleSysAdmin); err != nil {
			fmt.Printf("Error assigning sys-admin to %s: %v\n", loginName, err)
		}
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Login handles POST /auth/login — returns session credentials only
// @Summary Login with email/password
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body zitadel.LoginRequest true "Credentials"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /arkivy/api/v1.0/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req zitadel.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data: loginName and password are required"})
		return
	}

	session, err := h.zClient.Login(c.Request.Context(), req.LoginName, req.Password)
	if err != nil {
		log.Printf("[Login] Zitadel error for %s: %v", req.LoginName, err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Sync sys-admin role if applicable (in background, does not block the response)
	if session.UserID != "" {
		h.syncSysAdminRole(c.Request.Context(), session.UserID, session.LoginName)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionId":    session.SessionID,
		"sessionToken": session.SessionToken,
	})
}

// Register handles POST /auth/register — creates a user and returns session credentials
// @Summary Register a new user
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body zitadel.RegisterRequest true "Registration data"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /arkivy/api/v1.0/auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req zitadel.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid data: username and password are required"})
		return
	}

	session, userID, err := h.zClient.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Sync sys-admin role if applicable
	if userID != "" {
		h.syncSysAdminRole(c.Request.Context(), userID, req.Username)
	}

	c.JSON(http.StatusCreated, gin.H{
		"sessionId":    session.SessionID,
		"sessionToken": session.SessionToken,
	})
}

// Logout handles POST /auth/logout
// @Summary Log out
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /arkivy/api/v1.0/auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req struct {
		SessionID    string `json:"sessionId" binding:"required"`
		SessionToken string `json:"sessionToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "sessionId and sessionToken are required"})
		return
	}

	if err := h.zClient.DeleteSession(c.Request.Context(), req.SessionID, req.SessionToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error closing session"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Session closed"})
}

// Me handles GET /auth/me — returns full user info
// @Summary Current user information
// @Tags Auth
// @Produce json
// @Param X-Session-Id header string true "Session ID"
// @Param X-Session-Token header string true "Session Token"
// @Success 200 {object} map[string]any
// @Failure 401 {object} map[string]string
// @Router /arkivy/api/v1.0/auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	sessionID := c.GetHeader("X-Session-Id")
	sessionToken := c.GetHeader("X-Session-Token")

	if sessionID == "" || sessionToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session not provided"})
		return
	}

	info, err := h.zClient.GetSession(c.Request.Context(), sessionID, sessionToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired session"})
		return
	}

	if info.UserID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session without verified user"})
		return
	}

	// Get user roles
	roles, _ := h.zClient.GetUserRoles(c.Request.Context(), info.UserID)

	c.JSON(http.StatusOK, gin.H{
		"userId":      info.UserID,
		"loginName":   info.LoginName,
		"displayName": info.DisplayName,
		"roles":       roles,
	})
}

// GoogleLogin handles GET /auth/google
// @Summary Start login with Google
// @Tags Auth
// @Produce json
// @Router /arkivy/api/v1.0/auth/google [get]
func (h *Handler) GoogleLogin(c *gin.Context) {
	if h.googleIDP == "" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Google login not configured"})
		return
	}

	successURL := h.frontURL + "/auth/callback?provider=google"
	failureURL := h.frontURL + "/login?error=google_failed"

	authURL, err := h.zClient.StartIDPIntent(c.Request.Context(), h.googleIDP, successURL, failureURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error starting login with Google"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"authUrl": authURL})
}

// GitHubLogin handles GET /auth/github
// @Summary Start login with GitHub
// @Tags Auth
// @Produce json
// @Router /arkivy/api/v1.0/auth/github [get]
func (h *Handler) GitHubLogin(c *gin.Context) {
	if h.githubIDP == "" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "GitHub login not configured"})
		return
	}

	successURL := h.frontURL + "/auth/callback?provider=github"
	failureURL := h.frontURL + "/login?error=github_failed"

	authURL, err := h.zClient.StartIDPIntent(c.Request.Context(), h.githubIDP, successURL, failureURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error starting login with GitHub"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"authUrl": authURL})
}

// IDPCallback handles POST /auth/idp/callback
// @Summary IDP callback (Google/GitHub)
// @Tags Auth
// @Accept json
// @Produce json
// @Router /arkivy/api/v1.0/auth/idp/callback [post]
func (h *Handler) IDPCallback(c *gin.Context) {
	var req struct {
		IntentID    string `json:"intentId" binding:"required"`
		IntentToken string `json:"intentToken" binding:"required"`
		UserID      string `json:"userId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid callback data"})
		return
	}

	session, err := h.zClient.CreateSessionWithIDP(
		c.Request.Context(),
		req.UserID,
		req.IntentID,
		req.IntentToken,
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error completing login with provider"})
		return
	}

	// Sync sys-admin role if applicable
	if session.UserID != "" && session.LoginName != "" {
		h.syncSysAdminRole(c.Request.Context(), session.UserID, session.LoginName)
	}

	c.JSON(http.StatusOK, gin.H{
		"sessionId":    session.SessionID,
		"sessionToken": session.SessionToken,
	})
}
