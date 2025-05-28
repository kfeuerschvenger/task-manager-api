package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kfeuerschvenger/task-manager-api/errors"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// GenerateJWT creates a new JWT token for the given user ID.
// The token is valid for 72 hours and includes the user ID in the claims.
func GenerateJWT(userID string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 72).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(jwtSecret))
}

// VerifyJWT checks the validity of a JWT token and returns the user ID if valid.
// It returns an error if the token is invalid or if the claims are not as expected.
func VerifyJWT(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return "", errors.ErrInvalidField("token")
	}

	claims := token.Claims.(jwt.MapClaims)
	userID, ok := claims["user_id"].(string)
	if !ok {
		return "", errors.ErrInvalidField("claims")
	}
	return userID, nil
}