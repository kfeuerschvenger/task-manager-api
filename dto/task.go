package dto

import "time"

// CreateTaskInput represents the input data required to create a new task.
type CreateTaskInput struct {
	Title       string    `json:"title" binding:"required" example:"Complete project documentation"`
	Description string    `json:"description" binding:"required" example:"Write detailed documentation for the project including setup, usage, and API endpoints."`
	DueDate     time.Time `json:"due_date" binding:"required" example:"2023-12-31T23:59:59Z"` // ISO string
	Priority    string    `json:"priority" binding:"omitempty,oneof=low medium high" example:"high"` // default: medium
	Status      string    `json:"status" binding:"omitempty,oneof=pending in_progress complete" example:"in_progress"` // default: pending
	AssigneeID  string    `json:"assignee_id,omitempty" example:"123e4567-e89b-12d3-a456-426614174000"` // UUID of the user assigned to the task
}

// UpdateTaskDTO represents the data transfer object for updating a task.
// It allows partial updates to a task's fields, with each field being optional.
type UpdateTaskDTO struct {
	Status      string `json:"status,omitempty" binding:"omitempty,oneof=pending in_progress complete" example:"in_progress"`
	Priority    string `json:"priority,omitempty" binding:"omitempty,oneof=low medium high" example:"high"`
	DueDate     string `json:"due_date,omitempty" example:"2023-12-31T23:59:59Z"` // ISO string
	Description string `json:"description,omitempty" example:"Write detailed documentation for the project including setup, usage, and API endpoints."`
	AssigneeID  string `json:"assignee_id,omitempty" example:"123e4567-e89b-12d3-a456-426614174000"`
}

// TaskResponse represents the response structure for a task.
// It includes all fields of a task, formatted for API responses.
type TaskResponse struct {
    ID          string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
    Title       string    `json:"title" example:"Complete project documentation"`
    Description string    `json:"description" example:"Write detailed documentation for the project including setup, usage, and API endpoints."`
    DueDate     time.Time `json:"due_date" example:"2025-06-01T15:04:05Z"`
    Status      string    `json:"status" example:"pending"`
    Priority    string    `json:"priority" example:"high"`
}