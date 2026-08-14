package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	ServerPort string
}

func Load() *Config {
	// Load .env from project root (works when running from backend/ or project root)
	if err := godotenv.Load("../.env"); err != nil {
		if err := godotenv.Load(".env"); err != nil {
			log.Println("No .env file found — using environment variables")
		}
	}

	return &Config{
		DBHost:     getEnvOrFatal("DB_HOST"),
		DBPort:     getEnvOrFatal("DB_PORT"),
		DBUser:     getEnvOrFatal("DB_USER"),
		DBPassword: getEnvOrFatal("DB_PASSWORD"),
		DBName:     getEnvOrFatal("DB_NAME"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}
}

func getEnvOrFatal(key string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	log.Fatalf("Required environment variable %s is not set. Copy .env.example to .env and fill in values.", key)
	return ""
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
