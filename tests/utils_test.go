package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	"github.com/gorilla/mux"
	"github.com/kfeuerschvenger/task-manager-api/database"
	"github.com/kfeuerschvenger/task-manager-api/routes"
)

var (
	// Router is the shared HTTP router for all tests
	Router *mux.Router

	// once ensures the test user setup runs only once
	once sync.Once

	// testToken holds the JWT for the test user
	testToken string

	// credentials for the test user
	testEmail = "testuser@example.com"
	testPass  = "test12345"
)

// TestMain sets up the test database (env via Docker), applies migrations, connects GORM, and initializes the router before running tests.
func TestMain(m *testing.M) {
	// Migrations and environment variables are provided by Docker env_file, so skip file loading.

	// Apply database migrations for tests
	if err := database.MigrateUp(); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	// Connect to the test database
	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	// Initialize the HTTP router
	Router = routes.SetupRoutes()

	// Run the tests
	code := m.Run()
	os.Exit(code)
}

// SetupTestUser registers a test user and retrieves a valid JWT token.
// It uses sync.Once to ensure setup only happens once per test suite.
func SetupTestUser(t *testing.T) string {
	once.Do(func() {
		// Register the test user
		testUser := map[string]string{
			"first_name": "Test",
			"last_name":  "User",
			"email":      testEmail,
			"password":   testPass,
		}
		// Registration
		token, err := doAuthRequest("/auth/register", testUser, false)
		if err != nil {
			log.Fatalf("Registration failed: %v", err)
		}

		// Login to get JWT
		loginCreds := map[string]string{
			"email":    testEmail,
			"password": testPass,
		}
		token, err = doAuthRequest("/auth/login", loginCreds, true)
		if err != nil {
			log.Fatalf("Login failed: %v", err)
		}
		testToken = token
	})
	return testToken
}

// doAuthRequest helper sends POST to path with payload.
// If expectToken==true returns token from response body.
func doAuthRequest(path string, payload interface{}, expectToken bool) (string, error) {
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	Router.ServeHTTP(resp, req)

	if expectToken {
		if resp.Code != http.StatusOK {
			return "", fmt.Errorf("expected 200 OK, got %d", resp.Code)
		}
		var data map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
			return "", err
		}
		return data["token"], nil
	}
	// If not expecting a token, just check success
	if resp.Code >= 300 {
		return "", fmt.Errorf("unexpected status %d", resp.Code)
	}
	return "", nil
}
