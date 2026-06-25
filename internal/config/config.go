package config

import (
	"log"
	"os"
	"sync"

	"github.com/joho/godotenv"
)

var (
	instance *Config
	once     sync.Once
)

type Config struct {
	// Server
	Port string

	// MongoDB
	MongoURI string
	MongoDB  string

	// Zitadel
	ZitadelDomain       string
	ZitadelClientID     string
	ZitadelKeyPath      string
	ZitadelServiceToken string
	ZitadelProjectID    string
	ZitadelGoogleIDP    string
	ZitadelGitHubIDP    string

	// Frontend
	FrontendURL string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() *Config {
	once.Do(func() {
		if err := godotenv.Load(); err != nil {
			log.Println("Warning: .env file not found, using system environment variables")
		}

		instance = &Config{
			Port:                getEnv("PORT", "9090"),
			MongoURI:            getEnv("MONGO_URI", "mongodb://localhost:27017"),
			MongoDB:             getEnv("MONGO_DB", "arkivy"),
			ZitadelDomain:       getEnv("ZITADEL_DOMAIN", ""),
			ZitadelClientID:     getEnv("ZITADEL_CLIENT_ID", ""),
			ZitadelKeyPath:      getEnv("ZITADEL_KEY_PATH", "key.json"),
			ZitadelServiceToken: getEnv("ZITADEL_SERVICE_TOKEN", ""),
			ZitadelProjectID:    getEnv("ZITADEL_PROJECT_ID", ""),
			ZitadelGoogleIDP:    getEnv("ZITADEL_GOOGLE_IDP", ""),
			ZitadelGitHubIDP:    getEnv("ZITADEL_GITHUB_IDP", ""),
			FrontendURL:         getEnv("FRONTEND_URL", "http://localhost:4200"),
		}
	})

	return instance
}
