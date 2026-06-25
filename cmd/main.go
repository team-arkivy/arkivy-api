package main

import (
	"log"

	"arkivy-api/internal/config"
	"arkivy-api/internal/database"
	"arkivy-api/internal/server"
)

func main() {
	cfg := config.Load()

	client, disconnect := database.Connect(cfg.MongoURI)
	defer disconnect()

	srv := server.New(cfg, client)
	if err := srv.Run(); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
