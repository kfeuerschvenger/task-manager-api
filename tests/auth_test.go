package tests

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegistrationWithExistingEmail(t *testing.T) {
	user := newTestUser(t)

	payload := map[string]string{
		"first_name": "Test",
		"last_name":  "User",
		"email":      user.Email, // Email already exists
		"password":   "anotherpassword",
	}

	code, _ := postJSON(t, "/auth/register", payload)

	assert.Equal(t, http.StatusConflict, code)
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
			code, _ := postJSON(t, "/auth/register", tc.payload)

			assert.Equal(t, tc.expectedCode, code)
		})
	}
}

func TestLoginWithValidCredentials(t *testing.T) {
	user := newTestUser(t)

	code, body := postJSON(t, "/auth/login", map[string]string{
		"email":    user.Email,
		"password": user.Password,
	})
	assert.Equal(t, http.StatusOK, code)

	var response map[string]string
	assert.NoError(t, json.Unmarshal(body, &response))

	assert.NotEmpty(t, response["token"])
	assert.Empty(t, response["error"])
}

func TestLoginWithWrongPassword(t *testing.T) {
	user := newTestUser(t)

	code, _ := postJSON(t, "/auth/login", map[string]string{
		"email":    user.Email,
		"password": "not-the-password",
	})

	assert.Equal(t, http.StatusUnauthorized, code)
}
