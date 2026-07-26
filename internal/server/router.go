package server

import (
	"arkivy-api/internal/arkivy/game"
	"arkivy-api/internal/auth"
	"net/http"

	_ "arkivy-api/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func registerRoutes(r *gin.Engine, db *mongo.Database, authHandler *auth.Handler) {
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/arkivy/api/v1.0")

	auth.RegisterRoutes(v1, authHandler)
	game.RegisterRoutes(v1, db)
}
