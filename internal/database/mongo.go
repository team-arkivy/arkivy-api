package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect creates a single reusable MongoDB client.
// Returns the client and a disconnect function to use with defer in main.
func Connect(uri string) (*mongo.Client, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(opts)
	if err != nil {
		log.Fatal("Failed to create MongoDB client:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Failed to ping MongoDB:", err)
	}

	fmt.Println("Successfully connected to MongoDB")

	disconnect := func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Println("Failed to disconnect MongoDB:", err)
		}
	}

	return client, disconnect
}
