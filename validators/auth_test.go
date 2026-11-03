package validators

import (
	"strings"
	"testing"

	"github.com/kfeuerschvenger/task-manager-api/dto"
	"github.com/stretchr/testify/assert"
)

func TestValidateRegisterInputPasswordLength(t *testing.T) {
	base := dto.RegisterRequest{FirstName: "John", LastName: "Doe", Email: "john@example.com"}

	const (
		errMin = "password must be at least 8 characters"
		errMax = "password must be at most 72 bytes"
	)

	tests := []struct {
		name     string
		password string
		wantErr  string
	}{
		{"seven characters", "1234567", errMin},
		{"eight characters", "12345678", ""},
		{"72 bytes", strings.Repeat("a", 72), ""},
		{"73 bytes", strings.Repeat("a", 73), errMax},
		// 8 runes of 2 bytes each: 16 bytes but exactly 8 characters.
		{"eight multibyte characters", strings.Repeat("é", 8), ""},
		// 7 runes of 2 bytes each: 14 bytes but only 7 characters.
		{"seven multibyte characters", strings.Repeat("é", 7), errMin},
		// 36 runes of 2 bytes each: 72 bytes.
		{"72 bytes multibyte", strings.Repeat("é", 36), ""},
		// 37 runes of 2 bytes each: 74 bytes.
		{"74 bytes multibyte", strings.Repeat("é", 37), errMax},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base
			req.Password = tt.password

			err := ValidateRegisterInput(req)

			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
