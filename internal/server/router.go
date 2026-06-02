package server

import (
	"arkivy-api/internal/arkivy/game"
	"arkivy-api/internal/auth"
	"arkivy-api/internal/zitadel"
	"net/http"

	_ "arkivy-api/docs"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"arkivy-api/internal/middleware"
)

func registerRoutes(r *gin.Engine, db *mongo.Database, zClient *zitadel.Client, authHandler *auth.Handler) {
	// Health check global
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Swagger UI
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// ─── API v1 ───────────────────────────────────────────────────────────────
	v1_0 := r.Group("/arkivy/v1.0")
	{
		// ─── Rutas de autenticación (públicas) ────────────────────────────
		authGroup := v1_0.Group("/auth")
		{
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/logout", authHandler.Logout)
			authGroup.GET("/me", authHandler.Me)
			authGroup.GET("/google", authHandler.GoogleLogin)
			authGroup.GET("/github", authHandler.GitHubLogin)
			authGroup.POST("/idp/callback", authHandler.IDPCallback)
		}

		// ─── Rutas públicas (sin autenticación) ───────────────────────────
		public := v1_0.Group("")
		registerPublicRoutes(public, db)

		// ─── Rutas protegidas (requieren sesión válida) ───────────────────
		protected := v1_0.Group("")
		protected.Use(middleware.SessionMiddleware(zClient))
		registerProtectedRoutes(protected, db)

		// ─── Rutas solo admin ─────────────────────────────────────────────
		admin := v1_0.Group("/admin")
		admin.Use(middleware.SessionMiddleware(zClient))
		admin.Use(middleware.RequireRole("sys-admin", "plat-admin"))
		registerAdminRoutes(admin, db)
	}
}

// registerPublicRoutes agrupa las rutas que no requieren autenticación.
func registerPublicRoutes(rg *gin.RouterGroup, db *mongo.Database) {
	testing := rg.Group("/testing")
	{
		_ = testing
	}
	game.RegisterRoutes(rg, db)
}

// registerProtectedRoutes agrupa las rutas que requieren sesión válida.
func registerProtectedRoutes(rg *gin.RouterGroup, db *mongo.Database) {
	// TODO: agregar rutas protegidas aquí
	_ = db
}

// registerAdminRoutes agrupa las rutas que requieren rol admin.
func registerAdminRoutes(rg *gin.RouterGroup, db *mongo.Database) {
	// TODO: agregar rutas de admin aquí
	_ = db
}
