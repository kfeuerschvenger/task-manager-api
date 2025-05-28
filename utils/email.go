package utils

import "strings"

// CleanEmail normalizes an email address by trimming whitespace and converting it to lowercase.
func CleanEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}