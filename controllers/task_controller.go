package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/kfeuerschvenger/task-manager-api/dto"
	"github.com/kfeuerschvenger/task-manager-api/middleware"
	"github.com/kfeuerschvenger/task-manager-api/services"
	"github.com/kfeuerschvenger/task-manager-api/utils"
)

// CreateTask godoc
// @Summary Create a new task
// @Description Creates a new task with the provided details.
// @Router /tasks [post]
// @Tags tasks
// @Accept  json
// @Produce  json
// @Param   input body dto.CreateTaskInput true "Task details"
// @Success 201 {object} dto.TaskResponse
// @Failure 400 {object} dto.ErrorResponse "Invalid input or missing required fields"
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
func CreateTask(w http.ResponseWriter, r *http.Request) {
	var input dto.CreateTaskInput

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		utils.Error(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if input.Title == "" || input.Description == "" || input.DueDate.IsZero() {
		utils.Error(w, http.StatusBadRequest, "Missing required fields")
		return
	}

	if input.Priority == "" {
		input.Priority = "medium"
	}
	if input.Status == "" {
		input.Status = "pending"
	}

	creatorID := r.Context().Value(middleware.UserIDKey).(string)
	if input.AssigneeID == "" {
		input.AssigneeID = creatorID
	}

	task, err := services.CreateTask(input, creatorID)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Failed to create task")
		return
	}

	resp := dto.TaskResponse{
    ID:          task.ID.String(),
    Title:       task.Title,
    Description: task.Description,
    DueDate:     task.DueDate,
    Status:      task.Status,
    Priority:    task.Priority,
	}
	utils.JSON(w, http.StatusCreated, resp)
}

// GetTasks godoc
// @Summary Get all tasks
// @Description Retrieves all tasks for the authenticated user, with optional filtering by status and priority.
// @Router /tasks [get]
// @Tags tasks
// @Accept  json
// @Produce  json
// @Param   status query string false "Filter by task status (pending, in_progress, complete)"
// @Param   priority query string false "Filter by task priority (low, medium, high)"
// @Success 200 {array} dto.TaskResponse
// @Failure 500 {object} dto.ErrorResponse "Internal server error"
// @Security BearerAuth
func GetTasks(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)

	// Optional query parameters for filtering
	status := r.URL.Query().Get("status")
	priority := r.URL.Query().Get("priority")

	tasks, err := services.GetTasks(userID, status, priority)
	if err != nil {
		utils.Error(w, http.StatusInternalServerError, "Failed to retrieve tasks")
		return
	}

	// Map models to response DTOs
	var resp []dto.TaskResponse
	for _, t := range tasks {
			resp = append(resp, dto.TaskResponse{
					ID:          t.ID.String(),
					Title:       t.Title,
					Description: t.Description,
					DueDate:     t.DueDate,
					Status:      t.Status,
					Priority:    t.Priority,
			})
	}
	utils.JSON(w, http.StatusOK, resp)
}

// GetTaskByID godoc
// @Summary Obtiene una tarea por ID
// @Description Obtiene una tarea específica por su ID
// @Router /tasks/{id} [get]
// @Tags tasks
// @Param id path string true "ID de la tarea"
// @Success 200 {object} dto.TaskResponse
// @Failure 404 {object} dto.ErrorResponse
// @Security BearerAuth
func GetTaskByID(w http.ResponseWriter, r *http.Request) {
    userID := r.Context().Value(middleware.UserIDKey).(string)
    taskID := mux.Vars(r)["id"]

    task, err := services.GetTaskByID(taskID, userID)
    if err != nil {
        if err.Error() == "task not found" {
            utils.Error(w, http.StatusNotFound, "Task not found")
        } else {
            utils.Error(w, http.StatusInternalServerError, "Failed to retrieve task")
        }
        return
    }

    resp := dto.TaskResponse{
        ID:          task.ID.String(),
        Title:       task.Title,
        Description: task.Description,
        DueDate:     task.DueDate,
        Status:      task.Status,
        Priority:    task.Priority,
    }
    utils.JSON(w, http.StatusOK, resp)
}

// UpdateTask godoc
// @Summary Update an existing task
// @Description Updates the details of an existing task.
// @Router /tasks/{id} [put]
// @Tags tasks
// @Accept  json
// @Produce  json
// @Param   id path string true "Task ID"
// @Param   input body dto.UpdateTaskDTO true "Updated task details"
// @Success 200 {object} dto.TaskResponse
// @Failure 400 {object} dto.ErrorResponse "Invalid input or missing required fields"
// @Failure 404 {object} dto.ErrorResponse "Task not found"
// @Failure 403 {object} dto.ErrorResponse "Unauthorized to update this task"
// @Security BearerAuth
func UpdateTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	taskID := mux.Vars(r)["id"]

	var updateData dto.UpdateTaskDTO
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		utils.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	task, err := services.UpdateTask(taskID, userID, updateData)
	if err != nil {
		if err.Error() == "unauthorized" {
			utils.Error(w, http.StatusForbidden, "You are not the creator of this task")
			return
		}
		if err.Error() == "task not found" {
			utils.Error(w, http.StatusNotFound, "Task not found")
			return
		}
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	if task == nil {
    utils.Error(w, http.StatusInternalServerError, "Task update failed")
    return
	}

	// Map to response DTO
	resp := dto.TaskResponse{
		ID:          task.ID.String(),
		Title:       task.Title,
		Description: task.Description,
		DueDate:     task.DueDate,
		Status:      task.Status,
		Priority:    task.Priority,
	}
	utils.JSON(w, http.StatusOK, resp)
}

// DeleteTask godoc
// @Summary Delete a task
// @Description Deletes a task by its ID.
// @Router /tasks/{id} [delete]
// @Tags tasks
// @Param   id path string true "Task ID"
// @Success 204 {object} nil
// @Failure 404 {object} dto.ErrorResponse "Task not found"
// @Failure 403 {object} dto.ErrorResponse "Unauthorized to delete this task"
// @Security BearerAuth
func DeleteTask(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(middleware.UserIDKey).(string)
	taskID := mux.Vars(r)["id"]

	err := services.DeleteTask(taskID, userID)
	if err != nil {
		switch err.Error() {
		case "task not found":
			utils.Error(w, http.StatusNotFound, "Task not found")
		case "unauthorized":
			utils.Error(w, http.StatusForbidden, "You are not authorized to delete this task")
		default:
			utils.Error(w, http.StatusInternalServerError, "Error deleting task")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
