// @title           Task Manager API
// @version         1.0
// @description     RESTful API for collaborative task management with JWT authentication.

// @contact.name   Kevin Feuerschvenger
// @contact.url   https://feuerschvenger.com/

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8080
// @BasePath  /
// @schemes http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization

package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/kfeuerschvenger/task-manager-api/database"
	_ "github.com/kfeuerschvenger/task-manager-api/docs"
	"github.com/kfeuerschvenger/task-manager-api/middleware"
	"github.com/kfeuerschvenger/task-manager-api/routes"
	"github.com/kfeuerschvenger/task-manager-api/utils"
)

// maxBodyBytes caps the size of any request body (1 MiB).
const maxBodyBytes = 1 << 20

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	// Parse flags and determine mode
	flag.Parse()
	args := flag.Args()
	mode := "serve"
	if len(args) > 0 {
		mode = args[0]
	}

	switch mode {
	case "migrate":
		direction := "up"
		if len(args) > 1 {
			direction = args[1]
		}
		runMigrations(direction)
		return
	case "serve":
		if err := validateStartupConfig(); err != nil {
			log.Fatalf("Invalid configuration: %v", err)
		}

		// Automatically apply migrations before starting
		if err := database.MigrateUp(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}

		// Establish database connection
		if err := database.Connect(); err != nil {
			log.Fatalf("Database connection error: %v", err)
		}

		startServer()
	default:
		fmt.Println("Usage:")
		fmt.Println("  migrate [up|down]  Run database migrations")
		fmt.Println("  serve              Start the server (default)")
		os.Exit(1)
	}
}

// validateStartupConfig checks the configuration that must be valid before the server starts:
// a strong JWT_SECRET, a recognized DB_SSLMODE, and a supported sslmode inside DATABASE_URL when it is set.
func validateStartupConfig() error {
	if err := utils.ValidateJWTSecret(); err != nil {
		return err
	}
	if err := database.ValidateSSLMode(); err != nil {
		return err
	}
	return database.ValidateDatabaseURL()
}

// runMigrations applies or reverts migrations based on the provided direction
func runMigrations(direction string) {
	switch direction {
	case "up":
		if err := database.MigrateUp(); err != nil {
			log.Fatalf("Failed to apply migrations: %v", err)
		}
		log.Println("Migrations applied successfully")
	case "down":
		if err := database.MigrateDown(); err != nil {
			log.Fatalf("Failed to revert migrations: %v", err)
		}
		log.Println("Migrations reverted successfully")
	default:
		log.Fatalf("Invalid migration direction: %s", direction)
	}
}

// startServer bootstraps and runs the HTTP server with graceful shutdown
func startServer() {
	router := routes.SetupRoutes()
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           middleware.LimitBody(maxBodyBytes, router),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig

		log.Println("Shutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("Shutdown error: %v", err)
		}
	}()

	log.Printf("Server listening on port %s", port)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}