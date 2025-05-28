package database

import (
	"fmt"
	"os"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// getDatabaseURL constructs the migration URL from DATABASE_URL or DB_* vars
func getDatabaseURL() (string, error) {
    if url := os.Getenv("DATABASE_URL"); url != "" {
        return url, nil
    }
    host := os.Getenv("DB_HOST")
    port := os.Getenv("DB_PORT")
    user := os.Getenv("DB_USER")
    password := os.Getenv("DB_PASSWORD")
    dbname := os.Getenv("DB_NAME")

    if host == "" || port == "" || user == "" || dbname == "" {
        return "", fmt.Errorf("database connection variables are not set")
    }
    // Construct URL in postgres URI format
    return fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s?sslmode=disable",
        user, password, host, port, dbname,
    ), nil
}

// MigrateUp applies all up migrations; reads MIGRATIONS_PATH and constructs the DB URL
func MigrateUp() error {
    mPath := os.Getenv("MIGRATIONS_PATH")
    if mPath == "" {
        mPath = "file://database/migrations"
    }

    dbURL, err := getDatabaseURL()
    if err != nil {
        return err
    }

    m, err := migrate.New(mPath, dbURL)
    if err != nil {
        return err
    }
    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    return nil
}

// MigrateDown reverts all migrations in reverse; reads MIGRATIONS_PATH and constructs the DB URL
func MigrateDown() error {
    mPath := os.Getenv("MIGRATIONS_PATH")
    if mPath == "" {
        mPath = "file://database/migrations"
    }

    dbURL, err := getDatabaseURL()
    if err != nil {
        return err
    }

    m, err := migrate.New(mPath, dbURL)
    if err != nil {
        return err
    }
    if err := m.Down(); err != nil && err != migrate.ErrNoChange {
        return err
    }
    return nil
}
