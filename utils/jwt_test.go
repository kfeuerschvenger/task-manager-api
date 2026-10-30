package utils

import (
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
