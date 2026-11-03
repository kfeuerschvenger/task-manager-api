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
		{"dev with a valid password", "dev", "test12345", ""},
		{"local with a valid password", "local", "test12345", ""},
		{"test with a valid password", "test", "test12345", ""},
		{"allowed value in any case and padded", " Development ", "test12345", ""},
		{"unset environment is refused", "", "test12345", "seed refused"},
		{"blank environment is refused", "   ", "test12345", "seed refused"},
		{"production is refused", "production", "test12345", "seed refused"},
		{"production is refused in any case", " Production ", "test12345", "seed refused"},
		{"staging is refused", "staging", "test12345", "seed refused"},
		{"a misspelled value is refused", "prod", "test12345", "seed refused"},
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
