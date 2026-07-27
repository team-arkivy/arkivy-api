package main

import (
	"context"
	"log"
	"time"

	"arkivy-api/internal/config"
	"arkivy-api/internal/content"
	"arkivy-api/internal/database"
	"arkivy-api/internal/devauth"
	"arkivy-api/internal/groups"
	"arkivy-api/internal/organizations"
	"arkivy-api/internal/server"
)

func main() {
	cfg := config.Load()

	client, disconnect := database.Connect(cfg.MongoURI)
	defer disconnect()

	db := client.Database(cfg.MongoDB)

	seedCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := organizations.SeedPlans(seedCtx, db); err != nil {
		log.Fatal("Error seeding plans:", err)
	}
	if err := organizations.EnsureIndexes(seedCtx, db); err != nil {
		log.Fatal("Error creating indexes:", err)
	}
	if err := groups.EnsureIndexes(seedCtx, db); err != nil {
		log.Fatal("Error creating group indexes:", err)
	}
	if err := content.EnsureIndexes(seedCtx, db); err != nil {
		log.Fatal("Error creating content indexes:", err)
	}

	if cfg.DevAuthBypass {
		log.Println("⚠️⚠️⚠️  DEV_AUTH_BYPASS=true — Zitadel session validation is DISABLED, every request is authenticated as the fixed dev user. Never use this outside local development. ⚠️⚠️⚠️")
		if err := devauth.SeedUser(seedCtx, db); err != nil {
			log.Fatal("Error seeding dev user:", err)
		}
	}

	srv := server.New(cfg, client)
	if err := srv.Run(); err != nil {
		log.Fatal("Error starting server:", err)
	}
}
