package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistrationWithExistingEmail(t *testing.T) {
	// Ensure a user is already registered
	SetupTestUser(t)

	payload := map[string]string{
		"first_name": "Test",
		"last_name":  "User",
		"email":      testEmail, // Email already exists
		"password":   "anotherpassword",
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	Router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusConflict, resp.Code)
}

func TestRegistrationWithInvalidData(t *testing.T) {
	testCases := []struct {
		name         string
		payload      map[string]string
		expectedCode int
	}{
		{
			name: "Short password",
			payload: map[string]string{
				"first_name": "John",
				"last_name":  "Doe",
				"email":      "shortpass@example.com",
				"password":   "123",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Invalid email",
			payload: map[string]string{
				"first_name": "John",
				"last_name":  "Doe",
				"email":      "invalid-email",
				"password":   "validpass",
			},
			expectedCode: http.StatusBadRequest,
		},
		{
			name: "Missing fields",
			payload: map[string]string{
				"first_name": "",
				"email":      "missing@fields.com",
				"password":   "validpass",
			},
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body, _ := json.Marshal(tc.payload)

			req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			resp := httptest.NewRecorder()
			Router.ServeHTTP(resp, req)

			assert.Equal(t, tc.expectedCode, resp.Code)
		})
	}
}

func TestLoginWithValidCredentials(t *testing.T) {
	SetupTestUser(t) // Ensure the test user exists

	payload := map[string]string{
		"email":    testEmail,
		"password": testPass,
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp := httptest.NewRecorder()
	Router.ServeHTTP(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)

	var response map[string]string
	json.Unmarshal(resp.Body.Bytes(), &response)

	assert.NotEmpty(t, response["token"])
	assert.Empty(t, response["error"])
}
