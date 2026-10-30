package controllers

import (
	stderrors "errors"
	"log"
	"net/http"
	"strings"

	apperrors "github.com/kfeuerschvenger/task-manager-api/errors"
	"github.com/kfeuerschvenger/task-manager-api/utils"
)

// writeError maps a service error to an HTTP response using errors.As on the typed errors.
// Client-facing categories (validation, not found, forbidden, conflict, auth) expose their message.
// Anything else is logged and answered with internalMsg, so internal error text never reaches the client.
func writeError(w http.ResponseWriter, err error, internalMsg string) {
	var (
		validationErr *apperrors.ValidationError
		notFoundErr   *apperrors.NotFoundError
		forbiddenErr  *apperrors.ForbiddenError
		conflictErr   *apperrors.ConflictError
		authErr       *apperrors.AuthError
	)

	switch {
	case stderrors.As(err, &validationErr):
		utils.Error(w, http.StatusBadRequest, capitalize(validationErr.Message))
	case stderrors.As(err, &notFoundErr):
		utils.Error(w, http.StatusNotFound, capitalize(notFoundErr.Error()))
	case stderrors.As(err, &forbiddenErr):
		utils.Error(w, http.StatusForbidden, capitalize(forbiddenErr.Error()))
	case stderrors.As(err, &conflictErr):
		utils.Error(w, http.StatusConflict, capitalize(conflictErr.Message))
	case stderrors.As(err, &authErr):
		utils.Error(w, http.StatusUnauthorized, capitalize(authErr.Message))
	default:
		log.Printf("internal error: %v", err)
		utils.Error(w, http.StatusInternalServerError, internalMsg)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
