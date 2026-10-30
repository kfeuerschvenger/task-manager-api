package database

import (
	"net/url"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// trickyPassword contains every character that breaks naive URL or DSN construction.
const trickyPassword = `p@ss/w#rd:100% x'y\z`

func TestSSLModeDefaultsToDisable(t *testing.T) {
	t.Setenv("DB_SSLMODE", "")

	assert.Equal(t, "disable", SSLMode())
}

func TestSSLModeReadsEnvironment(t *testing.T) {
	t.Setenv("DB_SSLMODE", "require")

	assert.Equal(t, "require", SSLMode())
}

func TestValidateSSLMode(t *testing.T) {
	for _, mode := range []string{"", "disable", "allow", "prefer", "require", "verify-ca", "verify-full"} {
		t.Run("valid "+mode, func(t *testing.T) {
			t.Setenv("DB_SSLMODE", mode)

			assert.NoError(t, ValidateSSLMode())
		})
	}

	for _, mode := range []string{"bogus", "Require", "verify_full", "disable sslrootcert=/etc/passwd"} {
		t.Run("invalid "+mode, func(t *testing.T) {
			t.Setenv("DB_SSLMODE", mode)

			err := ValidateSSLMode()

			require.Error(t, err)
			assert.Contains(t, err.Error(), "invalid DB_SSLMODE")
		})
	}
}

func TestDatabaseURLUsesSSLMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "u")
	t.Setenv("DB_PASSWORD", "p")
	t.Setenv("DB_NAME", "n")
	t.Setenv("DB_SSLMODE", "verify-full")
	t.Setenv("DB_SSLROOTCERT", "")

	dbURL, err := getDatabaseURL()

	assert.NoError(t, err)
	assert.Equal(t, "postgres://u:p@db:5432/n?sslmode=verify-full", dbURL)
}

func TestDatabaseURLEscapesSpecialCharacters(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "us@r:name")
	t.Setenv("DB_PASSWORD", trickyPassword)
	t.Setenv("DB_NAME", "my db")
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("DB_SSLROOTCERT", "/certs/root cert.crt")

	dbURL, err := getDatabaseURL()
	require.NoError(t, err)

	parsed, err := url.Parse(dbURL)
	require.NoError(t, err)
	password, _ := parsed.User.Password()
	assert.Equal(t, "us@r:name", parsed.User.Username())
	assert.Equal(t, trickyPassword, password)
	assert.Equal(t, "db:5432", parsed.Host)
	assert.Equal(t, "/my db", parsed.Path)
	assert.Equal(t, "require", parsed.Query().Get("sslmode"))
	assert.Equal(t, "/certs/root cert.crt", parsed.Query().Get("sslrootcert"))
}

func TestDatabaseURLRejectsInvalidSSLMode(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("DB_HOST", "db")
	t.Setenv("DB_PORT", "5432")
	t.Setenv("DB_USER", "u")
	t.Setenv("DB_PASSWORD", "p")
	t.Setenv("DB_NAME", "n")
	t.Setenv("DB_SSLMODE", "bogus")

	_, err := getDatabaseURL()

	assert.ErrorContains(t, err, "invalid DB_SSLMODE")
}

func TestBuildDSNQuotesSpecialCharacters(t *testing.T) {
	t.Setenv("DB_SSLMODE", "require")
	t.Setenv("DB_SSLROOTCERT", "")

	dsn, err := buildDSN("db", "5432", `us er'x`, trickyPassword, "my db")
	require.NoError(t, err)

	cfg, err := pgconn.ParseConfig(dsn)
	require.NoError(t, err)
	assert.Equal(t, "db", cfg.Host)
	assert.Equal(t, uint16(5432), cfg.Port)
	assert.Equal(t, `us er'x`, cfg.User)
	assert.Equal(t, trickyPassword, cfg.Password)
	assert.Equal(t, "my db", cfg.Database)
}

func TestBuildDSNRejectsInvalidSSLMode(t *testing.T) {
	t.Setenv("DB_SSLMODE", "bogus")

	_, err := buildDSN("db", "5432", "u", "p", "n")

	assert.ErrorContains(t, err, "invalid DB_SSLMODE")
}

func TestQuoteDSNValue(t *testing.T) {
	assert.Equal(t, `'plain'`, quoteDSNValue("plain"))
	assert.Equal(t, `'a b'`, quoteDSNValue("a b"))
	assert.Equal(t, `'it\'s'`, quoteDSNValue("it's"))
	assert.Equal(t, `'back\\slash'`, quoteDSNValue(`back\slash`))
}
