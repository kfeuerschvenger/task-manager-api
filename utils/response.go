package utils

import (
	"encoding/json"
	"net/http"

	"github.com/kfeuerschvenger/task-manager-api/dto"
)

// Response utility functions for sending different types of HTTP responses

// JSON sends a JSON response with the specified status code and data.
func JSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// PlainText sends a plain text response with the specified status code and message.
func PlainText(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(message))
}

// Error sends a JSON response with an error message and the specified status code.
func Error(w http.ResponseWriter, status int, message string) {
	resp := dto.ErrorResponse{Code: status, Message: message}
	JSON(w, status, resp)
}
