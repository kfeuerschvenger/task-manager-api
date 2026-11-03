package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseDemoReference(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    time.Time
		wantErr string
	}{
		{"unset keeps the deterministic default", "", DefaultDemoReference, ""},
		{"blank keeps the deterministic default", "  ", DefaultDemoReference, ""},
		{"a valid date is taken at 09:00 UTC", "2027-03-15", time.Date(2027, time.March, 15, 9, 0, 0, 0, time.UTC), ""},
		{"surrounding spaces are ignored", " 2027-03-15 ", time.Date(2027, time.March, 15, 9, 0, 0, 0, time.UTC), ""},
		{"another layout is rejected", "15/03/2027", time.Time{}, "DEMO_REFERENCE must be a date in YYYY-MM-DD format"},
		{"a timestamp is rejected", "2027-03-15T09:00:00Z", time.Time{}, "YYYY-MM-DD"},
		{"a day that does not exist is rejected", "2027-02-30", time.Time{}, "YYYY-MM-DD"},
		{"text is rejected", "tomorrow", time.Time{}, "YYYY-MM-DD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDemoReference(tt.value)

			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.True(t, tt.want.Equal(got), "want %s, got %s", tt.want, got)
		})
	}
}

func TestDefaultDemoReferenceIsFixed(t *testing.T) {
	assert.True(t, time.Date(2026, time.November, 2, 9, 0, 0, 0, time.UTC).Equal(DefaultDemoReference))
}
