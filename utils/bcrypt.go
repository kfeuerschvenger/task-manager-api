package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes the given password using bcrypt and returns the hashed password as a string.
// It returns an error if the hashing fails.
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// dummyHash is computed once at package initialization so the first unknown-email login costs the same
// as any later one. Initialization panics on failure: a missing hash would make the comparison instant
// and silently disable the timing equalization.
var dummyHash = mustDummyHash()

func mustDummyHash() []byte {
	hash, err := bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)
	if err != nil {
		panic("utils: failed to generate dummy bcrypt hash: " + err.Error())
	}
	return hash
}

// BurnPasswordCheck runs a bcrypt comparison against a precomputed hash and always returns false.
// Call it when no stored hash exists (for example an unknown email at login) so that path costs
// the same as a real comparison and response timing does not reveal which emails are registered.
func BurnPasswordCheck(password string) bool {
	_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
	return false
}

// CheckPasswordHash compares a plaintext password with a hashed password and returns true if they match.
// It returns false if they do not match or if there is an error during comparison.
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}