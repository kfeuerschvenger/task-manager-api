package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCreateTaskWithValidData(t *testing.T) {
    token := SetupTestUser(t)
    
    dueDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
    payload := map[string]interface{}{
        "title":       "Test Task",
        "description": "Test Description",
        "due_date":    dueDate,
        "priority":    "high",
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)

    resp := httptest.NewRecorder()
    Router.ServeHTTP(resp, req)

    assert.Equal(t, http.StatusCreated, resp.Code)
    
    var response map[string]interface{}
    json.Unmarshal(resp.Body.Bytes(), &response)
    
    assert.NotEmpty(t, response["id"])
    assert.Equal(t, "Test Task", response["title"])
    assert.Equal(t, "high", response["priority"])
    assert.Equal(t, "pending", response["status"])
}

func TestCreateTaskWithoutAuthentication(t *testing.T) {
    payload := map[string]interface{}{
        "title":       "Unauthorized Task",
        "description": "Should fail",
        "due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp := httptest.NewRecorder()
    Router.ServeHTTP(resp, req)

    assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestGetTasksWithFilters(t *testing.T) {
    token := SetupTestUser(t)
    
    createTestTask(t, token, "high", "pending")    // This should appear
    createTestTask(t, token, "medium", "pending")  // This should not appear
    createTestTask(t, token, "high", "complete")   // This should not appear

    req := httptest.NewRequest(http.MethodGet, "/tasks?status=pending&priority=high", nil)
    req.Header.Set("Authorization", "Bearer "+token)

    resp := httptest.NewRecorder()
    Router.ServeHTTP(resp, req)
    
    assert.Equal(t, http.StatusOK, resp.Code)
    
    var response []map[string]interface{}
    json.Unmarshal(resp.Body.Bytes(), &response)
    
    assert.Len(t, response, 1)
    assert.Equal(t, "high", response[0]["priority"])
    assert.Equal(t, "pending", response[0]["status"])
}

func TestUpdateTaskAsCreator(t *testing.T) {
    token := SetupTestUser(t)
    taskID := createTestTask(t, token, "medium", "pending")
    
    payload := map[string]interface{}{
        "status":     "in_progress",
        "priority":   "high",
        "due_date":   time.Now().Add(48 * time.Hour).Format(time.RFC3339),
        "description": "Updated description",
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest(http.MethodPut, "/tasks/"+taskID, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)

    resp := httptest.NewRecorder()
    Router.ServeHTTP(resp, req)

    assert.Equal(t, http.StatusOK, resp.Code)
    
    var response map[string]interface{}
    json.Unmarshal(resp.Body.Bytes(), &response)
    
    assert.Equal(t, "in_progress", response["status"])
    assert.Equal(t, "high", response["priority"])
    assert.NotEqual(t, response["created_at"], response["updated_at"])
}

func TestUpdateTaskAsNonCreator(t *testing.T) {
    // Create first user and task
    creatorToken := SetupTestUser(t)
    taskID := createTestTask(t, creatorToken, "medium", "pending")
    
    // Create second user
    nonCreatorToken := registerTestUser(t, "anotheruser@example.com", "password123")
    
    payload := map[string]interface{}{
        "status": "complete",
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest(http.MethodPut, "/tasks/"+taskID, bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+nonCreatorToken)

    resp := httptest.NewRecorder()
    Router.ServeHTTP(resp, req)

    assert.Equal(t, http.StatusForbidden, resp.Code)
}

func TestDeleteTask(t *testing.T) {
    token := SetupTestUser(t)
    taskID := createTestTask(t, token, "low", "pending")

    // Verify task exists
    req := httptest.NewRequest(http.MethodGet, "/tasks/"+taskID, nil)
    req.Header.Set("Authorization", "Bearer "+token)

    resp := httptest.NewRecorder()
    Router.ServeHTTP(resp, req)
    assert.Equal(t, http.StatusOK, resp.Code)

    // Verify that the task no longer exists
    req = httptest.NewRequest(http.MethodDelete, "/tasks/"+taskID, nil)
    req.Header.Set("Authorization", "Bearer "+token)

		resp = httptest.NewRecorder()
    Router.ServeHTTP(resp, req)
    assert.Equal(t, http.StatusNoContent, resp.Code)
}

// Helpers

func createTestTask(t *testing.T, token string, priority string, status string) string {
    dueDate := time.Now().Add(24 * time.Hour).Format(time.RFC3339)
    title := "Test Task"
    if priority == "high" && status == "pending" {
        title = "Filtered Task"
    }
    payload := map[string]interface{}{
        "title":       title,
        "description": "Test Description",
        "due_date":    dueDate,
        "priority":    priority,
        "status":      status,
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest(http.MethodPost, "/tasks", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Authorization", "Bearer "+token)

    resp := httptest.NewRecorder()
    Router.ServeHTTP(resp, req)

    if resp.Code != http.StatusCreated {
        t.Fatalf("Failed to create test task: %s", resp.Body.String())
    }

    var response map[string]interface{}
    json.Unmarshal(resp.Body.Bytes(), &response)
    return response["id"].(string)
}

func registerTestUser(t *testing.T, email string, password string) string {
    payload := map[string]string{
        "first_name": "Test",
        "last_name":  "User",
        "email":      email,
        "password":   password,
    }
    body, _ := json.Marshal(payload)

    req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
    req.Header.Set("Content-Type", "application/json")

    resp := httptest.NewRecorder()
    Router.ServeHTTP(resp, req)

    if resp.Code != http.StatusCreated {
        t.Fatalf("Failed to register test user: %s", resp.Body.String())
    }

    // Log in the user to get a token
    loginPayload := map[string]string{
        "email":    email,
        "password": password,
    }
    loginBody, _ := json.Marshal(loginPayload)

    req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(loginBody))
    req.Header.Set("Content-Type", "application/json")

    resp = httptest.NewRecorder()
    Router.ServeHTTP(resp, req)

    var loginResponse map[string]string
    json.Unmarshal(resp.Body.Bytes(), &loginResponse)
    return loginResponse["token"]
}