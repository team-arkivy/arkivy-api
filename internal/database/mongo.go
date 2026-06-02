package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Connect crea un único cliente de MongoDB reutilizable.
// Devuelve el cliente y una función disconnect para usar con defer en main.
func Connect(uri string) (*mongo.Client, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	opts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(opts)
	if err != nil {
		log.Fatal("Error al crear cliente MongoDB:", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		log.Fatal("Error al hacer ping a MongoDB:", err)
	}

	fmt.Println("Conectado exitosamente a MongoDB")

	disconnect := func() {
		if err := client.Disconnect(context.Background()); err != nil {
			log.Println("Error al desconectar MongoDB:", err)
		}
	}

	return client, disconnect
}
