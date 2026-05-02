package main

import (
	"log"

	"arkivy-api/internal/config"
	"arkivy-api/internal/database"
	"arkivy-api/internal/server"
)

// @title           Arkivy API
// @version         1.0
// @description     API del proyecto Arkivy
// @host            localhost:9090
// @BasePath        /arkivy/v1.0
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	client, disconnect := database.Connect(cfg.MongoURI)
	defer disconnect()

	srv := server.New(cfg, client)
	if err := srv.Run(); err != nil {
		log.Fatal("Error iniciando el servidor:", err)
	}
}
