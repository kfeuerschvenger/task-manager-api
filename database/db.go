package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// SSLMode returns the PostgreSQL sslmode taken from DB_SSLMODE, defaulting to "disable".
func SSLMode() string {
	if mode := os.Getenv("DB_SSLMODE"); mode != "" {
		return mode
	}
	return "disable"
}

// Connect initializes the database connection using GORM with PostgreSQL.
// It reads the database connection parameters from environment variables
func Connect() error {
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	dbname := os.Getenv("DB_NAME")

	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		host, user, password, dbname, port, SSLMode(),
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db
	return nil
}