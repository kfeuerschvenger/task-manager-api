package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// seedTestPassword is a test fixture that satisfies the password policy. It is not a credential.
const seedTestPassword = "test12345"

func TestCheckSeedAllowed(t *testing.T) {
	tests := []struct {
		name     string
		appEnv   string
		password string
		wantErr  string
	}{
		{"development with a valid password", "development", seedTestPassword, ""},
		{"dev with a valid password", "dev", seedTestPassword, ""},
		{"local with a valid password", "local", seedTestPassword, ""},
		{"test with a valid password", "test", seedTestPassword, ""},
		{"allowed value in any case and padded", " Development ", seedTestPassword, ""},
		{"unset environment is refused", "", seedTestPassword, "seed refused"},
		{"blank environment is refused", "   ", seedTestPassword, "seed refused"},
		{"production is refused", "production", seedTestPassword, "seed refused"},
		{"production is refused in any case", " Production ", seedTestPassword, "seed refused"},
		{"staging is refused", "staging", seedTestPassword, "seed refused"},
		{"a misspelled value is refused", "prod", seedTestPassword, "seed refused"},
		{"the environment is checked before the password", "production", "", "seed refused"},
		{"missing password", "development", "", "DEMO_PASSWORD is not set"},
		{"short password", "development", "short", "at least 8 characters"},
		{"password over 72 bytes", "development", strings.Repeat("p", 73), "at most 72 bytes"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := checkSeedAllowed(tt.appEnv, tt.password)

			if tt.wantErr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tt.password, password)
				return
			}
			assert.ErrorContains(t, err, tt.wantErr)
			assert.Empty(t, password)
		})
	}
}

func TestIsLocalSeedHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{"localhost", true},
		{"LocalHost", true},
		{" localhost ", true},
		{"127.0.0.1", true},
		{"::1", true},
		{"db", true},
		{"host.docker.internal", true},
		{"", false},
		{"db.internal.example.com", false},
		{"10.0.0.5", false},
		{"localhost.example.com", false},
		{"mydb", false},
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			assert.Equal(t, tt.want, isLocalSeedHost(tt.host))
		})
	}
}

func TestSeedTarget(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		dbHost   string
		dbName   string
		wantHost string
		wantName string
		wantErr  string
	}{
		{"the DB variables when there is no URL", "", "db", "taskmanager", "db", "taskmanager", ""},
		{"a blank URL falls back to the DB variables", "  ", "localhost", "tm", "localhost", "tm", ""},
		{"the URL wins over the DB variables", "postgres://u:secret@remote.example.com:5432/prod?sslmode=require", "db", "tm", "remote.example.com", "prod", ""},
		{"an IPv6 host loses its brackets", "postgres://u:secret@[::1]:5432/tm", "", "", "::1", "tm", ""},
		{"a URL without a path has no database name", "postgres://u:secret@localhost", "", "", "localhost", "", ""},
		{"an unparsable URL is refused without echoing it", "postgres://u:secret@db:5432/%zz", "", "", "", "", "DATABASE_URL cannot be parsed"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, name, err := seedTarget(tt.url, tt.dbHost, tt.dbName)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				assert.NotContains(t, err.Error(), "secret")
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantHost, host)
			assert.Equal(t, tt.wantName, name)
		})
	}
}

func TestCheckSeedTarget(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		allowRemote string
		wantErr     bool
	}{
		{"a local host passes", "localhost", "", false},
		{"the compose service passes", "db", "", false},
		{"a remote host is refused", "db.example.com", "", true},
		{"an empty host is refused", "", "", true},
		{"a remote host passes with the flag", "db.example.com", "1", false},
		{"the flag is compared trimmed", "db.example.com", " 1 ", false},
		{"any other flag value is refused", "db.example.com", "true", true},
		{"zero does not enable it", "db.example.com", "0", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkSeedTarget(tt.host, tt.allowRemote)

			if tt.wantErr {
				assert.ErrorContains(t, err, "seed refused")
				assert.ErrorContains(t, err, seedAllowRemoteVariable)
				return
			}
			assert.NoError(t, err)
		})
	}
}
