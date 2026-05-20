package auth

import (
	"context"
	"fmt"
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

// syncSysAdminRole verifica si el usuario está en la colección sys_admins
// y sincroniza el rol en Zitadel
func (h *Handler) syncSysAdminRole(ctx context.Context, userID, loginName string) {
	// Checar si el email está en la colección sys_admins
	isSysAdmin := h.db.Collection("sys_admins").
		FindOne(ctx, bson.M{"email": loginName}).Err() == nil

	// Obtener roles actuales
	currentRoles, err := h.zClient.GetUserRoles(ctx, userID)
	if err != nil {
		return
	}

	hasSysAdminRole := contains(currentRoles, "sys-admin")

	if isSysAdmin && !hasSysAdminRole {
		// Está en la tabla segura pero no tiene el rol → asignar
		if err := h.zClient.AssignRole(ctx, userID, "sys-admin"); err != nil {
			fmt.Printf("Error asignando sys-admin a %s: %v\n", loginName, err)
		}
	}
	// Nota: no quitamos el rol automáticamente por seguridad.
	// Si quieres también remover el rol cuando no esté en la tabla,
	// implementa RemoveRole en el client de Zitadel.
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Login maneja POST /auth/login
// @Summary Login con email/password
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body zitadel.LoginRequest true "Credenciales"
// @Success 200 {object} zitadel.SessionResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req zitadel.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: se requiere loginName y password"})
		return
	}

	session, err := h.zClient.Login(c.Request.Context(), req.LoginName, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Credenciales inválidas"})
		return
	}

	// Sincronizar rol sys-admin si aplica
	if session.UserID != "" {
		h.syncSysAdminRole(c.Request.Context(), session.UserID, session.LoginName)
	}

	// Obtener roles para incluirlos en la respuesta
	roles, _ := h.zClient.GetUserRoles(c.Request.Context(), session.UserID)

	c.JSON(http.StatusOK, gin.H{
		"sessionId":    session.SessionID,
		"sessionToken": session.SessionToken,
		"userId":       session.UserID,
		"displayName":  session.DisplayName,
		"loginName":    session.LoginName,
		"roles":        roles,
	})
}

// Register maneja POST /auth/register
// @Summary Registrar nuevo usuario
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body zitadel.RegisterRequest true "Datos de registro"
// @Success 201 {object} zitadel.SessionResponse
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req zitadel.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos inválidos: se requiere username y password"})
		return
	}

	session, userID, err := h.zClient.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Sincronizar rol sys-admin si aplica
	if userID != "" {
		h.syncSysAdminRole(c.Request.Context(), userID, req.Username)
	}

	// Obtener roles actualizados
	roles, _ := h.zClient.GetUserRoles(c.Request.Context(), userID)

	c.JSON(http.StatusCreated, gin.H{
		"sessionId":    session.SessionID,
		"sessionToken": session.SessionToken,
		"userId":       userID,
		"displayName":  session.DisplayName,
		"loginName":    session.LoginName,
		"roles":        roles,
	})
}

// Logout maneja POST /auth/logout
// @Summary Cerrar sesión
// @Tags Auth
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Router /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	var req struct {
		SessionID    string `json:"sessionId" binding:"required"`
		SessionToken string `json:"sessionToken" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Se requiere sessionId y sessionToken"})
		return
	}

	if err := h.zClient.DeleteSession(c.Request.Context(), req.SessionID, req.SessionToken); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Error cerrando sesión"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Sesión cerrada"})
}

// Me maneja GET /auth/me — devuelve info de la sesión actual
// @Summary Información del usuario actual
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]any
// @Failure 401 {object} map[string]string
// @Router /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	sessionID := c.GetHeader("X-Session-Id")
	sessionToken := c.GetHeader("X-Session-Token")

	if sessionID == "" || sessionToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesión no proporcionada"})
		return
	}

	info, err := h.zClient.GetSession(c.Request.Context(), sessionID, sessionToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Sesión inválida"})
		return
	}

	// Obtener roles del usuario
	roles, _ := h.zClient.GetUserRoles(c.Request.Context(), info.UserID)

	c.JSON(http.StatusOK, gin.H{
		"userId":      info.UserID,
		"loginName":   info.LoginName,
		"displayName": info.DisplayName,
		"roles":       roles,
	})
}

// GoogleLogin maneja GET /auth/google — redirige a Google
// @Summary Iniciar login con Google
// @Tags Auth
// @Produce json
// @Router /auth/google [get]
func (h *Handler) GoogleLogin(c *gin.Context) {
	if h.googleIDP == "" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "Google login no configurado"})
		return
	}

	successURL := h.frontURL + "/auth/callback?provider=google"
	failureURL := h.frontURL + "/login?error=google_failed"

	authURL, err := h.zClient.StartIDPIntent(c.Request.Context(), h.googleIDP, successURL, failureURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iniciando login con Google"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"authUrl": authURL})
}

// GitHubLogin maneja GET /auth/github — redirige a GitHub
// @Summary Iniciar login con GitHub
// @Tags Auth
// @Produce json
// @Router /auth/github [get]
func (h *Handler) GitHubLogin(c *gin.Context) {
	if h.githubIDP == "" {
		c.JSON(http.StatusNotImplemented, gin.H{"error": "GitHub login no configurado"})
		return
	}

	successURL := h.frontURL + "/auth/callback?provider=github"
	failureURL := h.frontURL + "/login?error=github_failed"

	authURL, err := h.zClient.StartIDPIntent(c.Request.Context(), h.githubIDP, successURL, failureURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error iniciando login con GitHub"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"authUrl": authURL})
}

// IDPCallback maneja POST /auth/idp/callback — procesa el callback de un IDP externo
// @Summary Callback de IDP (Google/GitHub)
// @Tags Auth
// @Accept json
// @Produce json
// @Router /auth/idp/callback [post]
func (h *Handler) IDPCallback(c *gin.Context) {
	var req struct {
		IntentID    string `json:"intentId" binding:"required"`
		IntentToken string `json:"intentToken" binding:"required"`
		UserID      string `json:"userId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Datos de callback inválidos"})
		return
	}

	session, err := h.zClient.CreateSessionWithIDP(
		c.Request.Context(),
		req.UserID,
		req.IntentID,
		req.IntentToken,
	)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Error completando login con proveedor"})
		return
	}

	// Sincronizar rol sys-admin si aplica
	if session.UserID != "" && session.LoginName != "" {
		h.syncSysAdminRole(c.Request.Context(), session.UserID, session.LoginName)
	}

	// Obtener roles
	roles, _ := h.zClient.GetUserRoles(c.Request.Context(), session.UserID)

	c.JSON(http.StatusOK, gin.H{
		"sessionId":    session.SessionID,
		"sessionToken": session.SessionToken,
		"userId":       session.UserID,
		"displayName":  session.DisplayName,
		"loginName":    session.LoginName,
		"roles":        roles,
	})
}
