package server

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/mongo"

	"arkivy-api/internal/config"
)

type Server struct {
	engine *gin.Engine
	port   string
}

func New(cfg *config.Config, client *mongo.Client) *Server {
	r := gin.New()

	// Logger y recovery controlados (en lugar de gin.Default())
	r.Use(gin.Logger())
	r.Use(gin.Recovery())

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Registrar todas las rutas
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
