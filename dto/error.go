package dto

// ErrorResponse represents the structure of an error response returned by the API.
// It includes a code and a message to provide details about the error.
type ErrorResponse struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"Invalid input or missing required fields"`
}