package server

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"arkivy-api/internal/auth"
	"arkivy-api/internal/config"
	"arkivy-api/internal/content"
	"arkivy-api/internal/devauth"
	"arkivy-api/internal/groups"
	"arkivy-api/internal/organizations"
	"arkivy-api/internal/zitadel"
)

type Server struct {
	engine *gin.Engine
	port   string
}

func New(cfg *config.Config, client *mongo.Client) *Server {
	db := client.Database(cfg.MongoDB)

	// Initialize Zitadel client
	zClient := zitadel.NewClient(
		cfg.ZitadelDomain,
		cfg.ZitadelServiceToken,
		cfg.ZitadelProjectID,
	)

	// Initialize auth handler
	authHandler := auth.NewHandler(
		zClient,
		db,
		cfg.ZitadelGoogleIDP,
		cfg.ZitadelGitHubIDP,
		cfg.FrontendURL,
		cfg.DevAuthBypass,
		devauth.UserID,
	)

	groupsHandler := groups.NewHandler(db)
	contentHandler := content.NewHandler(db)
	organizationsHandler := organizations.NewHandler(db)

	r := gin.New()

	// Don't trust any proxy by default (no reverse proxy in front).
	// If you deploy behind nginx/traefik, set the proxy's IP/CIDR here instead.
	if err := r.SetTrustedProxies(nil); err != nil {
		log.Fatalf("failed to set trusted proxies: %v", err)
	}

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Session-Id", "X-Session-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	registerRoutes(r, db, zClient, authHandler, groupsHandler, contentHandler, organizationsHandler, cfg.DevAuthBypass)

	if cfg.ZitadelServiceToken == "" {
		log.Println("⚠️  ZITADEL_SERVICE_TOKEN not configured — auth endpoints will not work")
	}

	return &Server{
		engine: r,
		port:   cfg.Port,
	}
}

func (s *Server) Run() error {
	fmt.Printf("🚀 Server starting on port: %s\n", s.port)
	return s.engine.Run(":" + s.port)
}
