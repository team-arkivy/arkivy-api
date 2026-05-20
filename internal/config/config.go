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

	// Zitadel
	ZitadelDomain       string // https://dev-arkivy-ybl40k.us1.zitadel.cloud
	ZitadelClientID     string
	ZitadelKeyPath      string
	ZitadelServiceToken string // PAT del service account
	ZitadelProjectID    string
	ZitadelGoogleIDP    string // ID del IDP de Google en Zitadel (opcional)
	ZitadelGitHubIDP    string // ID del IDP de GitHub en Zitadel (opcional)

	// Frontend
	FrontendURL string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("Aviso: no se encontró .env, usando variables de entorno del sistema")
	}

	return &Config{
		Port:                getEnv("PORT", "9090"),
		MongoURI:            getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDB:             getEnv("MONGO_DB", "arkivy"),
		AllowedOrigins:      []string{getEnv("FRONTEND_URL", "http://localhost:4200")},
		ZitadelDomain:       getEnv("ZITADEL_DOMAIN", ""),
		ZitadelClientID:     getEnv("ZITADEL_CLIENT_ID", ""),
		ZitadelKeyPath:      getEnv("ZITADEL_KEY_PATH", "key.json"),
		ZitadelServiceToken: getEnv("ZITADEL_SERVICE_TOKEN", ""),
		ZitadelProjectID:    getEnv("ZITADEL_PROJECT_ID", ""),
		ZitadelGoogleIDP:    getEnv("ZITADEL_GOOGLE_IDP", ""),
		ZitadelGitHubIDP:    getEnv("ZITADEL_GITHUB_IDP", ""),
		FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:4200"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
