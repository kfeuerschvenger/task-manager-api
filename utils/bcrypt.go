package utils

import (
	"sync"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes the given password using bcrypt and returns the hashed password as a string.
// It returns an error if the hashing fails.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

var (
	dummyHashOnce sync.Once
	dummyHash     []byte
)

// BurnPasswordCheck runs a bcrypt comparison against a precomputed hash and always returns false.
// Call it when no stored hash exists (for example an unknown email at login) so that path costs
// the same as a real comparison and response timing does not reveal which emails are registered.
func BurnPasswordCheck(password string) bool {
	dummyHashOnce.Do(func() {
		dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	})
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
	return false
}

// CheckPasswordHash compares a plaintext password with a hashed password and returns true if they match.
// It returns false if they do not match or if there is an error during comparison.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}