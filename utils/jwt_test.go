package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTReadsSecretAtCallTime(t *testing.T) {
	// The secret is set after the package is initialized, as happens when main() loads a .env file.
	t.Setenv("JWT_SECRET", "late-bound-secret")

	token, err := GenerateJWT("user-1")
	require.NoError(t, err)

	userID, err := VerifyJWT(token)
	require.NoError(t, err)
	assert.Equal(t, "user-1", userID)
}

func TestValidateJWTSecret(t *testing.T) {
	tests := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{"empty", "", true},
		{"too short", "short-secret", true},
		{"one byte short", strings.Repeat("a", MinJWTSecretLength-1), true},
		{"exact minimum", strings.Repeat("a", MinJWTSecretLength), false},
		{"long", strings.Repeat("a", 64), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", tt.secret)

			err := ValidateJWTSecret()

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestJWTWithoutSecretFails(t *testing.T) {
	t.Setenv("JWT_SECRET", "")

	_, err := GenerateJWT("user-1")

	assert.Error(t, err)
}

func TestVerifyJWTRejectsTokenSignedWithAnotherSecret(t *testing.T) {
	t.Setenv("JWT_SECRET", "secret-a")
	token, err := GenerateJWT("user-1")
	require.NoError(t, err)

	t.Setenv("JWT_SECRET", "secret-b")
	_, err = VerifyJWT(token)

	assert.Error(t, err)
}
