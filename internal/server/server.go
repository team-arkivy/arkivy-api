package server

import (
	"fmt"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"arkivy-api/internal/config"
	"arkivy-api/internal/middleware"
)

type Server struct {
	engine *gin.Engine
	port   string
}

func New(cfg *config.Config, client *mongo.Client) *Server {
	// Inicializar Zitadel
	if err := middleware.InitAuth(cfg.ZitadelDomain, cfg.ZitadelKeyPath); err != nil {
		log.Fatalf("Error inicializando Zitadel: %v", err)
	}

	r := gin.New()

	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	db := client.Database(cfg.MongoDB)
	registerRoutes(r, db)

	return &Server{
		engine: r,
		port:   cfg.Port,
	}
}

func (s *Server) Run() error {
	fmt.Printf("Servidor iniciando en el puerto: %s\n", s.port)
	return s.engine.Run(":" + s.port)
}
