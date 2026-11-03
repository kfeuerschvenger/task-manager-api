package main

import (
	"errors"
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
		{"[::1]", true},
		{" [::1] ", true},
		{"[2001:db8::1]", false},
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
		{"a host query parameter is refused", "postgres://u:secret@localhost/x?host=remote.example.com", "", "", "", "", `"host" query parameter`},
		{"a hostaddr query parameter is refused", "postgres://u:secret@localhost/x?sslmode=disable&hostaddr=10.0.0.5", "", "", "", "", `"hostaddr" query parameter`},
		{"a service query parameter is refused", "postgres://u:secret@localhost/x?service=prod", "", "", "", "", `"service" query parameter`},
		{"other query parameters are fine", "postgres://u:secret@localhost/x?sslmode=disable", "", "", "localhost", "x", ""},
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

func TestSeedTargetsCollectsBothRoles(t *testing.T) {
	targets, err := seedTargets("postgres://u:secret@migrate.example.com/mdb", "db", "conn")

	assert.NoError(t, err)
	assert.Equal(t, []seedDatabaseTarget{
		{Role: "migrations", Host: "migrate.example.com", Name: "mdb"},
		{Role: "connection", Host: "db", Name: "conn"},
	}, targets)
}

func TestSeedTargetsWithoutURLUsesDBHostForBoth(t *testing.T) {
	targets, err := seedTargets("", "db", "tm")

	assert.NoError(t, err)
	assert.Equal(t, []seedDatabaseTarget{
		{Role: "migrations", Host: "db", Name: "tm"},
		{Role: "connection", Host: "db", Name: "tm"},
	}, targets)
}

func TestSeedTargetsRefusesAHostOverride(t *testing.T) {
	_, err := seedTargets("postgres://u:secret@localhost/x?host=remote.example.com", "localhost", "x")

	assert.ErrorContains(t, err, "seed refused")
	assert.NotContains(t, err.Error(), "secret")
}

func TestCheckSeedTargets(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
		dbHost      string
		allowRemote string
		wantRole    string
	}{
		{"both local passes", "postgres://u:p@localhost:5432/x", "db", "", ""},
		{"no URL and a local DB_HOST passes", "", "localhost", "", ""},
		{"a bracketed IPv6 DB_HOST passes", "", "[::1]", "", ""},
		{"a local URL with a remote DB_HOST is refused", "postgres://u:p@localhost:5432/x", "db.example.com", "", "connection"},
		{"a remote URL with a local DB_HOST is refused", "postgres://u:p@remote.example.com:5432/x", "localhost", "", "migrations"},
		{"no URL and a remote DB_HOST is refused", "", "db.example.com", "", "migrations"},
		{"an empty DB_HOST is refused", "", "", "", "migrations"},
		{"the flag overrides a remote DB_HOST", "postgres://u:p@localhost/x", "db.example.com", "1", ""},
		{"the flag overrides a remote URL", "postgres://u:p@remote.example.com/x", "localhost", "1", ""},
		{"the flag is compared trimmed", "", "db.example.com", " 1 ", ""},
		{"any other flag value is refused", "", "db.example.com", "true", "migrations"},
		{"zero does not enable it", "", "db.example.com", "0", "migrations"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			targets, err := seedTargets(tt.databaseURL, tt.dbHost, "x")
			assert.NoError(t, err)

			err = checkSeedTargets(targets, tt.allowRemote)

			if tt.wantRole != "" {
				assert.ErrorContains(t, err, "seed refused")
				assert.ErrorContains(t, err, "the "+tt.wantRole+" database host")
				assert.ErrorContains(t, err, seedAllowRemoteVariable)
				assert.NotContains(t, err.Error(), "p@")
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestOpenSeedDatabaseChecksTargetsBeforeAnyStep(t *testing.T) {
	var calls []string
	migrate := func() error { calls = append(calls, "migrate"); return nil }
	connect := func() error { calls = append(calls, "connect"); return nil }
	remote := []seedDatabaseTarget{{Role: "migrations", Host: "db.example.com"}, {Role: "connection", Host: "db"}}
	local := []seedDatabaseTarget{{Role: "migrations", Host: "localhost"}, {Role: "connection", Host: "db"}}

	err := openSeedDatabase(remote, "", migrate, connect)
	assert.ErrorContains(t, err, "seed refused")
	assert.Empty(t, calls)

	err = openSeedDatabase(local, "", migrate, connect)
	assert.NoError(t, err)
	assert.Equal(t, []string{"migrate", "connect"}, calls)

	calls = nil
	err = openSeedDatabase(remote, "1", migrate, connect)
	assert.NoError(t, err)
	assert.Equal(t, []string{"migrate", "connect"}, calls)
}

func TestOpenSeedDatabaseStopsWhenMigrationFails(t *testing.T) {
	connected := false
	local := []seedDatabaseTarget{{Role: "migrations", Host: "localhost"}}

	err := openSeedDatabase(local, "",
		func() error { return errors.New("boom") },
		func() error { connected = true; return nil })

	assert.ErrorContains(t, err, "migration failed")
	assert.False(t, connected)
}
