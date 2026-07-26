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
//
// @tag.name Auth
// @tag.description Authentication endpoints
// @tag.name Games
// @tag.description Game endpoints (just as an example)
//
// @securityDefinitions.apikey BearerAuth
// 	@in header
// 	@name Authorization

func main() {
	cfg := config.Load()

	client, disconnect := database.Connect(cfg.MongoURI)
	defer disconnect()

	srv := server.New(cfg, client)
	if err := srv.Run(); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
