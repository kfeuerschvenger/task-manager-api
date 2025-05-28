package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/kfeuerschvenger/task-manager-api/utils"
)

// Middleware for authentication
// It checks for a valid JWT token in the Authorization header and extracts the user ID.

type contextKey string

const UserIDKey = contextKey("userID")

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			utils.Error(w, http.StatusUnauthorized, "Missing or invalid token")
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		userID, err := utils.VerifyJWT(tokenString)
		if err != nil {
			utils.Error(w, http.StatusUnauthorized, "Invalid token")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}