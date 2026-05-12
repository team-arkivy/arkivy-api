package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	// Server
	Port string

	// MongoDB
	MongoURI string
	MongoDB  string

	// CORS
	AllowedOrigins []string

	// Zitadel (JWT/OIDC)
	ZitadelDomain   string
	ZitadelClientID string
	ZitadelKeyPath  string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: no se encontró .env, usando variables de entorno del sistema")
	}

	return &Config{
		Port:            getEnv("PORT", "9090"),
		MongoURI:        getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:         getEnv("MONGO_DB", "arkivy"),
		AllowedOrigins:  []string{getEnv("FRONTEND_URL", "http://localhost:4200")},
		ZitadelDomain:   getEnv("ZITADEL_DOMAIN", ""),
		ZitadelClientID: getEnv("ZITADEL_CLIENT_ID", ""),
		ZitadelKeyPath:  getEnv("ZITADEL_KEY_PATH", "key.json"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
