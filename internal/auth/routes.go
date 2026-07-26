package auth

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	auth := rg.Group("/auth")
	auth.POST("/login", h.Login)
	auth.POST("/register", h.Register)
	auth.POST("/logout", h.Logout)
	auth.GET("/me", h.Me)
	auth.GET("/google", h.GoogleLogin)
	auth.GET("/github", h.GitHubLogin)
	auth.POST("/idp/callback", h.IDPCallback)
}
