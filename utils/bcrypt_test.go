package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestBurnPasswordCheckAlwaysFalse(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	require.NoError(t, err)
	require.True(t, CheckPasswordHash("correct-horse", hash))

	for _, input := range []string{"", "anything", "correct-horse", "dummy-password-for-timing"} {
		assert.False(t, BurnPasswordCheck(input), "input %q", input)
	}
}

func TestDummyHashIsValidBcryptAtDefaultCost(t *testing.T) {
	require.NotEmpty(t, dummyHash)

	cost, err := bcrypt.Cost(dummyHash)

	require.NoError(t, err)
	assert.Equal(t, bcrypt.DefaultCost, cost)
}
