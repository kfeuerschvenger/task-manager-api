package controllers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/kfeuerschvenger/task-manager-api/errors"
	"github.com/stretchr/testify/assert"
)

func TestWriteErrorMapsTypedErrors(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		status   int
		contains string
	}{
		{"validation", apperrors.ErrInvalidID("task"), http.StatusBadRequest, "Invalid task ID"},
		{"not found", apperrors.ErrNotFound("task"), http.StatusNotFound, "Task not found"},
		{"forbidden", apperrors.ErrUnauthorizedAction("update", "task"), http.StatusForbidden, "Unauthorized to update task"},
		{"conflict", apperrors.NewConflictError("email already registered"), http.StatusConflict, "Email already registered"},
		{"auth", apperrors.NewAuthError("invalid credentials"), http.StatusUnauthorized, "Invalid credentials"},
		{"wrapped", fmt.Errorf("context: %w", apperrors.ErrNotFound("task")), http.StatusNotFound, "Task not found"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()

			writeError(rec, tc.err, "generic")

			assert.Equal(t, tc.status, rec.Code)
			assert.Contains(t, rec.Body.String(), tc.contains)
		})
	}
}

func TestWriteErrorHidesInternalDetails(t *testing.T) {
	internal := apperrors.WrapInternalServerError("error creating user", fmt.Errorf("pq: password authentication failed for user \"secret\""))

	for _, err := range []error{internal, fmt.Errorf("raw driver failure")} {
		rec := httptest.NewRecorder()

		writeError(rec, err, "Something went wrong")

		assert.Equal(t, http.StatusInternalServerError, rec.Code)
		assert.Contains(t, rec.Body.String(), "Something went wrong")
		assert.NotContains(t, rec.Body.String(), "secret")
		assert.NotContains(t, rec.Body.String(), "driver failure")
	}
}
