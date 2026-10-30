package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateStartupConfig(t *testing.T) {
	strong := strings.Repeat("s", 32)

	tests := []struct {
		name    string
		secret  string
		sslmode string
		dbURL   string
		wantErr string
	}{
		{"valid", strong, "require", "", ""},
		{"valid default sslmode", strong, "", "", ""},
		{"empty secret", "", "disable", "", "JWT_SECRET is not set"},
		{"short secret", "short", "disable", "", "JWT_SECRET must be at least"},
		{"invalid sslmode", strong, "bogus", "", "invalid DB_SSLMODE"},
		{"unsupported prefer", strong, "prefer", "", "invalid DB_SSLMODE"},
		{"valid database url", strong, "require", "postgres://u:p@db:5432/n?sslmode=verify-full", ""},
		{"invalid database url sslmode", strong, "require", "postgres://u:p@db:5432/n?sslmode=prefer", "invalid sslmode"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", tt.secret)
			t.Setenv("DB_SSLMODE", tt.sslmode)
			t.Setenv("DATABASE_URL", tt.dbURL)

			err := validateStartupConfig()

			if tt.wantErr == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.wantErr)
			}
		})
	}
}
