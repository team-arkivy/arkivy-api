package config

import (
	"context"
	"fmt"
	"log"
	"os"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// DB es el cliente global de MongoDB, accesible desde otros paquetes.
var DB *mongo.Client

// ConnectMongo establece la conexión con MongoDB usando la URI del entorno.
// Si MONGO_URI no está definida, usa localhost por defecto.
// Guarda el cliente en la variable global DB.
func ConnectMongo() {
	mongoURI := os.Getenv("MONGO_URI")
	if mongoURI == "" {
		mongoURI = "mongodb://localhost:27017"
	}

	opts := options.Client().ApplyURI(mongoURI)

	client, err := mongo.Connect(opts)
	if err != nil {
		log.Fatal("Error al crear cliente MongoDB:", err)
	}

	// Verificar que la conexión funciona con un ping
	if err := client.Ping(context.Background(), nil); err != nil {
		log.Fatal("Error al hacer ping a MongoDB:", err)
	}

	DB = client
	fmt.Println("Conectado exitosamente a MongoDB!")
}

// DisconnectMongo cierra la conexión con MongoDB de forma segura.
// Debe llamarse con defer en main.
func DisconnectMongo() {
	if err := DB.Disconnect(context.Background()); err != nil {
		log.Fatal("Error al desconectar MongoDB:", err)
	}
}
