package main

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckSeedAllowed(t *testing.T) {
	tests := []struct {
		name     string
		appEnv   string
		password string
		wantErr  string
	}{
		{"development with a valid password", "development", "test12345", ""},
		{"unset environment with a valid password", "", "test12345", ""},
		{"production is refused", "production", "test12345", "APP_ENV is production"},
		{"production is refused in any case", " Production ", "test12345", "APP_ENV is production"},
		{"production is refused before the password is checked", "production", "", "APP_ENV is production"},
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
