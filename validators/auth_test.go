package validators

import (
	"testing"

	"github.com/kfeuerschvenger/task-manager-api/dto"
	"github.com/stretchr/testify/assert"
)

func TestValidateRegisterInputPasswordLength(t *testing.T) {
	base := dto.RegisterRequest{FirstName: "John", LastName: "Doe", Email: "john@example.com"}

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"seven characters", "1234567", true},
		{"eight characters", "12345678", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := base
			req.Password = tt.password

			err := ValidateRegisterInput(req)

			if tt.wantErr {
				assert.EqualError(t, err, "password must be at least 8 characters")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
