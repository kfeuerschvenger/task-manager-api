package tests

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kfeuerschvenger/task-manager-api/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTaskWithValidData(t *testing.T) {
	token := SetupTestUser(t)

	payload := map[string]interface{}{
		"title":       "Test Task",
		"description": "Test Description",
		"due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"priority":    "high",
	}

	resp := doAuthed(t, http.MethodPost, "/tasks", token, payload)
	assert.Equal(t, http.StatusCreated, resp.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &response))

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

	resp := doAuthed(t, http.MethodPost, "/tasks", "", payload)

	assert.Equal(t, http.StatusUnauthorized, resp.Code)
}

func TestCreateTaskWithUnknownAssignee(t *testing.T) {
	token := SetupTestUser(t)

	payload := map[string]interface{}{
		"title":       "Orphan Task",
		"description": "Assignee does not exist",
		"due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"assignee_id": uuid.NewString(),
	}

	resp := doAuthed(t, http.MethodPost, "/tasks", token, payload)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "Assignee does not exist", errorMessage(t, resp))
}

func TestCreateTaskWithMalformedAssignee(t *testing.T) {
	token := SetupTestUser(t)

	payload := map[string]interface{}{
		"title":       "Bad Assignee",
		"description": "Assignee is not a UUID",
		"due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"assignee_id": "not-a-uuid",
	}

	resp := doAuthed(t, http.MethodPost, "/tasks", token, payload)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestCreateTaskWithInvalidPriority(t *testing.T) {
	token := SetupTestUser(t)

	payload := map[string]interface{}{
		"title":       "Bad Priority",
		"description": "Priority is not allowed",
		"due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"priority":    "urgent",
	}

	resp := doAuthed(t, http.MethodPost, "/tasks", token, payload)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "Invalid priority", errorMessage(t, resp))
}

func TestCreateTaskAssignedToAnotherUser(t *testing.T) {
	creator := newTestUser(t)
	assignee := newTestUser(t)
	assigneeID := userIDFromToken(t, assignee.Token)

	payload := map[string]interface{}{
		"title":       "Delegated Task",
		"description": "Assigned to someone else",
		"due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"assignee_id": assigneeID,
	}
	resp := doAuthed(t, http.MethodPost, "/tasks", creator.Token, payload)
	require.Equal(t, http.StatusCreated, resp.Code)

	var created map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &created))

	// The assignee can read it
	got := doAuthed(t, http.MethodGet, "/tasks/"+created["id"].(string), assignee.Token, nil)
	assert.Equal(t, http.StatusOK, got.Code)
}

func TestGetTasksWithFilters(t *testing.T) {
	token := SetupTestUser(t)

	createTestTask(t, token, "high", "pending")   // This should appear
	createTestTask(t, token, "medium", "pending") // This should not appear
	createTestTask(t, token, "high", "complete")  // This should not appear

	resp := doAuthed(t, http.MethodGet, "/tasks?status=pending&priority=high", token, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	var response []map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &response))

	require.Len(t, response, 1)
	assert.Equal(t, "high", response[0]["priority"])
	assert.Equal(t, "pending", response[0]["status"])
}

func TestGetTaskByIDWithMalformedID(t *testing.T) {
	token := SetupTestUser(t)

	resp := doAuthed(t, http.MethodGet, "/tasks/not-a-uuid", token, nil)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "Invalid task ID", errorMessage(t, resp))
}

func TestGetTaskByIDNotFound(t *testing.T) {
	token := SetupTestUser(t)

	resp := doAuthed(t, http.MethodGet, "/tasks/"+uuid.NewString(), token, nil)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestUpdateTaskAsCreator(t *testing.T) {
	token := SetupTestUser(t)
	taskID := createTestTask(t, token, "medium", "pending")

	newDue := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	payload := map[string]interface{}{
		"status":      "in_progress",
		"priority":    "high",
		"due_date":    newDue.Format(time.RFC3339),
		"description": "Updated description",
	}

	resp := doAuthed(t, http.MethodPut, "/tasks/"+taskID, token, payload)
	assert.Equal(t, http.StatusOK, resp.Code)

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &response))

	// TaskResponse does not expose created_at or updated_at, so assert on the fields it does return.
	assert.Equal(t, taskID, response["id"])
	assert.Equal(t, "in_progress", response["status"])
	assert.Equal(t, "high", response["priority"])
	assert.Equal(t, "Updated description", response["description"])
	assert.Equal(t, newDue.Format(time.RFC3339), response["due_date"])

	// The change is persisted, not only echoed back
	got := doAuthed(t, http.MethodGet, "/tasks/"+taskID, token, nil)
	require.Equal(t, http.StatusOK, got.Code)
	var persisted map[string]interface{}
	require.NoError(t, json.Unmarshal(got.Body.Bytes(), &persisted))
	assert.Equal(t, "in_progress", persisted["status"])
	assert.Equal(t, "Updated description", persisted["description"])
}

func TestUpdateTaskAsNonCreator(t *testing.T) {
	creatorToken := SetupTestUser(t)
	taskID := createTestTask(t, creatorToken, "medium", "pending")
	nonCreatorToken := SetupTestUser(t)

	resp := doAuthed(t, http.MethodPut, "/tasks/"+taskID, nonCreatorToken, map[string]interface{}{"status": "complete"})

	assert.Equal(t, http.StatusForbidden, resp.Code)

	// The task was not modified
	got := doAuthed(t, http.MethodGet, "/tasks/"+taskID, creatorToken, nil)
	var persisted map[string]interface{}
	require.NoError(t, json.Unmarshal(got.Body.Bytes(), &persisted))
	assert.Equal(t, "pending", persisted["status"])
}

func TestUpdateTaskWithMalformedID(t *testing.T) {
	token := SetupTestUser(t)

	resp := doAuthed(t, http.MethodPut, "/tasks/not-a-uuid", token, map[string]interface{}{"status": "complete"})

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Equal(t, "Invalid task ID", errorMessage(t, resp))
}

func TestUpdateTaskNotFound(t *testing.T) {
	token := SetupTestUser(t)

	resp := doAuthed(t, http.MethodPut, "/tasks/"+uuid.NewString(), token, map[string]interface{}{"status": "complete"})

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestUpdateTaskWithInvalidFields(t *testing.T) {
	token := SetupTestUser(t)
	taskID := createTestTask(t, token, "medium", "pending")

	testCases := []struct {
		name    string
		payload map[string]interface{}
		message string
	}{
		{"Bad due date", map[string]interface{}{"due_date": "tomorrow"}, "Invalid due_date"},
		{"Bad status", map[string]interface{}{"status": "done"}, "Invalid status"},
		{"Bad assignee ID", map[string]interface{}{"assignee_id": "nope"}, "Invalid assignee ID"},
		{"Unknown assignee", map[string]interface{}{"assignee_id": uuid.NewString()}, "Assignee does not exist"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resp := doAuthed(t, http.MethodPut, "/tasks/"+taskID, token, tc.payload)

			assert.Equal(t, http.StatusBadRequest, resp.Code)
			assert.Equal(t, tc.message, errorMessage(t, resp))
		})
	}
}

func TestDeleteTask(t *testing.T) {
	token := SetupTestUser(t)
	taskID := createTestTask(t, token, "low", "pending")

	// Verify task exists
	resp := doAuthed(t, http.MethodGet, "/tasks/"+taskID, token, nil)
	assert.Equal(t, http.StatusOK, resp.Code)

	resp = doAuthed(t, http.MethodDelete, "/tasks/"+taskID, token, nil)
	assert.Equal(t, http.StatusNoContent, resp.Code)

	// Verify that the task no longer exists
	resp = doAuthed(t, http.MethodGet, "/tasks/"+taskID, token, nil)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestDeleteTaskAsNonCreator(t *testing.T) {
	creatorToken := SetupTestUser(t)
	taskID := createTestTask(t, creatorToken, "low", "pending")
	nonCreatorToken := SetupTestUser(t)

	resp := doAuthed(t, http.MethodDelete, "/tasks/"+taskID, nonCreatorToken, nil)
	assert.Equal(t, http.StatusForbidden, resp.Code)

	// The task still exists for its creator
	resp = doAuthed(t, http.MethodGet, "/tasks/"+taskID, creatorToken, nil)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestDeleteTaskNotFound(t *testing.T) {
	token := SetupTestUser(t)

	resp := doAuthed(t, http.MethodDelete, "/tasks/"+uuid.NewString(), token, nil)

	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestDeleteTaskWithMalformedID(t *testing.T) {
	token := SetupTestUser(t)

	resp := doAuthed(t, http.MethodDelete, "/tasks/not-a-uuid", token, nil)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

// Helpers

func createTestTask(t *testing.T, token string, priority string, status string) string {
	t.Helper()
	payload := map[string]interface{}{
		"title":       "Test Task",
		"description": "Test Description",
		"due_date":    time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		"priority":    priority,
		"status":      status,
	}

	resp := doAuthed(t, http.MethodPost, "/tasks", token, payload)
	if resp.Code != http.StatusCreated {
		t.Fatalf("Failed to create test task: %s", resp.Body.String())
	}

	var response map[string]interface{}
	require.NoError(t, json.Unmarshal(resp.Body.Bytes(), &response))
	return response["id"].(string)
}

// userIDFromToken returns the user ID carried by a JWT. The API does not expose user IDs,
// so tests that assign tasks to another user read it from the token claims.
func userIDFromToken(t *testing.T, token string) string {
	t.Helper()
	id, err := utils.VerifyJWT(token)
	require.NoError(t, err)
	return id
}
