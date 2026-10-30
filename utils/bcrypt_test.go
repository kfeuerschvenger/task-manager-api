package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBurnPasswordCheckAlwaysFalseAndCostsABcryptCompare(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	assert.NoError(t, err)

	// Warm the dummy hash so only the comparison is measured.
	assert.False(t, BurnPasswordCheck("anything"))

	start := time.Now()
	CheckPasswordHash("wrong", hash)
	real := time.Since(start)

	start = time.Now()
	result := BurnPasswordCheck("wrong")
	burned := time.Since(start)

	assert.False(t, result)
	// Both paths run one bcrypt comparison at the same cost; allow generous scheduling noise.
	assert.Greater(t, burned, real/3)
	assert.Less(t, burned, real*3)
}
