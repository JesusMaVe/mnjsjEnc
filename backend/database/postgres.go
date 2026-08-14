package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	"securemessage/config"
)

func Connect(cfg *config.Config) (*sql.DB, error) {
	sslmode := getSSLMode()

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, sslmode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func getSSLMode() string {
	if v := os.Getenv("DB_SSLMODE"); v != "" {
		return v
	}
	return "disable" // default for local dev; set "require" in production
}
