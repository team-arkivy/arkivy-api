package server

import (
	"arkivy-api/internal/auth"
	"arkivy-api/internal/content"
	"arkivy-api/internal/devauth"
	"arkivy-api/internal/groups"
	"arkivy-api/internal/middleware"
	"arkivy-api/internal/organizations"
	"arkivy-api/internal/zitadel"
	"net/http"

	_ "arkivy-api/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func registerRoutes(r *gin.Engine, db *mongo.Database, zClient *zitadel.Client, authHandler *auth.Handler, groupsHandler *groups.Handler, contentHandler *content.Handler, organizationsHandler *organizations.Handler, devBypass bool) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/arkivy/api/v1.0")

	auth.RegisterRoutes(v1, authHandler)

	if devBypass {
		// Only exists at all when DEV_AUTH_BYPASS=true — see internal/devauth.
		v1.POST("/auth/dev-login", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"sessionId":    devauth.SessionID,
				"sessionToken": devauth.SessionToken,
			})
		})
	}

	sessionMW := middleware.SessionMiddleware(zClient, devBypass, devauth.UserID)
	groups.RegisterRoutes(v1, groupsHandler, sessionMW)
	content.RegisterRoutes(v1, contentHandler, sessionMW)
	organizations.RegisterRoutes(v1, organizationsHandler, sessionMW)
}
