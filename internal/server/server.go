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
	"arkivy-api/internal/zitadel"
)

type Server struct {
	engine *gin.Engine
	port   string
}

func New(cfg *config.Config, client *mongo.Client) *Server {
	db := client.Database(cfg.MongoDB)

	// Inicializar cliente de Zitadel
	zClient := zitadel.NewClient(
		cfg.ZitadelDomain,
		cfg.ZitadelServiceToken,
		cfg.ZitadelProjectID,
	)

	// Inicializar handler de auth
	authHandler := auth.NewHandler(
		zClient,
		db,
		cfg.ZitadelGoogleIDP,
		cfg.ZitadelGitHubIDP,
		cfg.FrontendURL,
	)

	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Session-Id", "X-Session-Token"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	registerRoutes(r, db, zClient, authHandler)

	if cfg.ZitadelServiceToken == "" {
		log.Println("⚠️  ZITADEL_SERVICE_TOKEN no configurado — los endpoints de auth no funcionarán")
	}

	return &Server{
		engine: r,
		port:   cfg.Port,
	}
}

func (s *Server) Run() error {
	fmt.Printf("🚀 Servidor iniciando en el puerto: %s\n", s.port)
	return s.engine.Run(":" + s.port)
}
