package tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/kfeuerschvenger/task-manager-api/database"
	"github.com/kfeuerschvenger/task-manager-api/routes"
)

// Router is the shared HTTP router for all tests.
var Router *mux.Router

// testUser holds the credentials and JWT of a user created for a single test.
type testUser struct {
	Email    string
	Password string
	Token    string
}

// TestMain applies migrations, connects GORM, and initializes the router before running tests.
// Connection settings come from the environment (DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME, JWT_SECRET).
func TestMain(m *testing.M) {
	// Tests run with the package directory as the working directory, so the default
	// relative migrations path used by the app would not resolve.
	if os.Getenv("MIGRATIONS_PATH") == "" {
		os.Setenv("MIGRATIONS_PATH", "file://../database/migrations")
	}

	if err := database.MigrateUp(); err != nil {
		log.Fatalf("Failed to apply migrations: %v", err)
	}

	if err := database.Connect(); err != nil {
		log.Fatalf("Failed to connect to test database: %v", err)
	}

	Router = routes.SetupRoutes()

	os.Exit(m.Run())
}

// uniqueEmail returns an email that has never been used, so tests do not depend on prior database state.
func uniqueEmail(prefix string) string {
	return fmt.Sprintf("%s-%s@example.com", prefix, uuid.NewString())
}

// newTestUser registers a brand-new user through the API and logs in to get a JWT.
// Every call creates an isolated user, so tests never share tasks and can run against a persisted database.
func newTestUser(t *testing.T) testUser {
	t.Helper()
	const password = "test12345"
	email := uniqueEmail("user")

	registerBody := map[string]string{
		"first_name": "Test",
		"last_name":  "User",
		"email":      email,
		"password":   password,
	}
	if code, body := postJSON(t, "/auth/register", registerBody); code != http.StatusCreated {
		t.Fatalf("registration failed: status %d body %s", code, body)
	}

	loginBody := map[string]string{"email": email, "password": password}
	code, body := postJSON(t, "/auth/login", loginBody)
	if code != http.StatusOK {
		t.Fatalf("login failed: status %d body %s", code, body)
	}
	var data map[string]string
	if err := json.Unmarshal(body, &data); err != nil || data["token"] == "" {
		t.Fatalf("login returned no token: %s", body)
	}

	return testUser{Email: email, Password: password, Token: data["token"]}
}

// SetupTestUser returns the JWT of a fresh, isolated test user.
func SetupTestUser(t *testing.T) string {
	t.Helper()
	return newTestUser(t).Token
}

// postJSON sends an unauthenticated JSON POST and returns the status code and raw body.
func postJSON(t *testing.T, path string, payload interface{}) (int, []byte) {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	Router.ServeHTTP(resp, req)
	return resp.Code, resp.Body.Bytes()
}

// doAuthed sends a request with a bearer token and an optional JSON payload.
func doAuthed(t *testing.T, method, path, token string, payload interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var reader *bytes.Reader
	if payload != nil {
		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		reader = bytes.NewReader(body)
	} else {
		reader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp := httptest.NewRecorder()
	Router.ServeHTTP(resp, req)
	return resp
}

// errorMessage extracts the message field from an ErrorResponse body.
func errorMessage(t *testing.T, resp *httptest.ResponseRecorder) string {
	t.Helper()
	var e struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &e); err != nil {
		t.Fatalf("response is not an error body: %s", resp.Body.String())
	}
	return e.Message
}
