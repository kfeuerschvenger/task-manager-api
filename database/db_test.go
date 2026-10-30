package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSSLModeDefaultsToDisable(t *testing.T) {
	t.Setenv("DB_SSLMODE", "")

	assert.Equal(t, "disable", SSLMode())
}

func TestSSLModeReadsEnvironment(t *testing.T) {
	t.Setenv("DB_SSLMODE", "require")

	assert.Equal(t, "require", SSLMode())
}

func TestDatabaseURLUsesSSLMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "u")
	t.Setenv("DB_PASSWORD", "p")
	t.Setenv("DB_NAME", "n")
	t.Setenv("DB_SSLMODE", "verify-full")

	url, err := getDatabaseURL()

	assert.NoError(t, err)
	assert.Equal(t, "postgres://u:p@db:5432/n?sslmode=verify-full", url)
}
