package server

import (
	"arkivy-api/internal/arkivy/game"
	"net/http"

	_ "arkivy-api/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"arkivy-api/internal/middleware"
)

func registerRoutes(r *gin.Engine, db *mongo.Database) {
	// Health check global
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ─── API v1 ───────────────────────────────────────────────────────────────
	v1_0 := r.Group("/arkivy/v1.0")
	{
		// Rutas públicas (sin autenticación)
		public := v1_0.Group("")
		registerPublicRoutes(public, db)

		// Rutas protegidas (requieren JWT de Zitadel)
		protected := v1_0.Group("")
		protected.Use(middleware.AuthMiddleware())
		registerProtectedRoutes(protected, db)
	}
}

// registerPublicRoutes agrupa las rutas que no requieren autenticación.
func registerPublicRoutes(rg *gin.RouterGroup, db *mongo.Database) {
	testing := rg.Group("/testing")
	{
		_ = testing
		// TODO: montar handlers de test1 aquí
		// testing.POST("/login", test1Module.Handler.Login)
	}
	// Example test
	game.RegisterRoutes(rg, db)
}

// registerProtectedRoutes agrupa las rutas que requieren token válido de Zitadel.
func registerProtectedRoutes(rg *gin.RouterGroup, db *mongo.Database) {
	// TODO: agregar rutas protegidas aquí a medida que crezca el proyecto
	// Ejemplo:
	// files := rg.Group("/files")
	// files.GET("", fileModule.Handler.List)
	_ = db
}
