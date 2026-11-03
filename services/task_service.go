package services

import (
	stderrors "errors"
	"time"

	"github.com/google/uuid"
	"github.com/kfeuerschvenger/task-manager-api/database"
	"github.com/kfeuerschvenger/task-manager-api/dto"
	"github.com/kfeuerschvenger/task-manager-api/errors"
	"github.com/kfeuerschvenger/task-manager-api/models"
	"gorm.io/gorm"
)

var (
	validPriorities = map[string]bool{"low": true, "medium": true, "high": true}
	validStatuses   = map[string]bool{"pending": true, "in_progress": true, "complete": true}
)

// validateTaskEnums rejects priority and status values the database CHECK constraints would refuse.
// Empty values are allowed and mean "not provided".
func validateTaskEnums(priority, status string) error {
	if priority != "" && !validPriorities[priority] {
		return errors.ErrInvalidField("priority")
	}
	if status != "" && !validStatuses[status] {
		return errors.ErrInvalidField("status")
	}
	return nil
}

// ensureUserExists returns a validation error when no user has the given ID.
func ensureUserExists(id uuid.UUID, label string) error {
	var count int64
	if err := database.DB.Model(&models.User{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return errors.WrapInternalServerError("error checking "+label, err)
	}
	if count == 0 {
		return errors.NewValidationError(label + " does not exist")
	}
	return nil
}

// parseTaskID parses a task ID, returning a validation error when it is not a UUID.
func parseTaskID(taskID string) (uuid.UUID, error) {
	id, err := uuid.Parse(taskID)
	if err != nil {
		return uuid.Nil, errors.ErrInvalidID("task")
	}
	return id, nil
}

func CreateTask(input dto.CreateTaskInput, creatorID string) (models.Task, error) {
	creatorUUID, err := uuid.Parse(creatorID)
	if err != nil {
		return models.Task{}, errors.ErrInvalidID("user")
	}

	assigneeUUID, err := uuid.Parse(input.AssigneeID)
	if err != nil {
		return models.Task{}, errors.ErrInvalidID("assignee")
	}

	if err := validateTaskEnums(input.Priority, input.Status); err != nil {
		return models.Task{}, err
	}

	if err := ensureUserExists(assigneeUUID, "assignee"); err != nil {
		return models.Task{}, err
	}

	task := models.Task{
		ID:          uuid.New(),
		Title:       input.Title,
		Description: input.Description,
		DueDate:     input.DueDate,
		Priority:    input.Priority,
		Status:      input.Status,
		CreatorID:   creatorUUID,
		AssigneeID:  assigneeUUID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := database.DB.Create(&task).Error; err != nil {
		return models.Task{}, errors.WrapInternalServerError("error creating task", err)
	}
	return task, nil
}

func GetTasks(userID, status, priority string) ([]models.Task, error) {
	var tasks []models.Task

	query := database.DB.
		Where("creator_id = ? OR assignee_id = ?", userID, userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if priority != "" {
		query = query.Where("priority = ?", priority)
	}

	err := query.Order("due_date ASC").Find(&tasks).Error
	return tasks, err
}

func GetTaskByID(taskID string, userID string) (*models.Task, error) {
	var task models.Task

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.ErrInvalidID("user")
	}

	taskUUID, err := parseTaskID(taskID)
	if err != nil {
		return nil, err
	}

	if err := database.DB.Where("id = ? AND (creator_id = ? OR assignee_id = ?)", taskUUID, userUUID, userUUID).First(&task).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrNotFound("task")
		}
		return nil, errors.WrapInternalServerError("error retrieving task", err)
	}

	return &task, nil
}

func UpdateTask(taskID string, userID string, dto dto.UpdateTaskDTO) (*models.Task, error) {
	var task models.Task

	taskUUID, err := parseTaskID(taskID)
	if err != nil {
		return nil, err
	}

	// Find the task by ID
	if err := database.DB.First(&task, "id = ?", taskUUID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.ErrNotFound("task")
		}
		return nil, errors.WrapInternalServerError("error retrieving task", err)
	}

	// Validate if the user is authorized to update the task
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.ErrInvalidID("user")
	}

	if task.CreatorID != userUUID {
		return nil, errors.ErrUnauthorizedAction("update", "task")
	}

	if err := validateTaskEnums(dto.Priority, dto.Status); err != nil {
		return nil, err
	}

	// Apply updates from the DTO
	if dto.Status != "" {
		task.Status = dto.Status
	}
	if dto.Priority != "" {
		task.Priority = dto.Priority
	}
	if dto.DueDate != "" {
		parsedDate, err := time.Parse(time.RFC3339, dto.DueDate)
		if err != nil {
			return nil, errors.ErrInvalidField("due_date")
		}
		task.DueDate = parsedDate
	}
	if dto.Description != "" {
		task.Description = dto.Description
	}
	if dto.AssigneeID != "" {
		assigneeUUID, err := uuid.Parse(dto.AssigneeID)
		if err != nil {
			return nil, errors.ErrInvalidID("assignee")
		}
		if err := ensureUserExists(assigneeUUID, "assignee"); err != nil {
			return nil, err
		}
		task.AssigneeID = assigneeUUID
	}

	task.UpdatedAt = time.Now()

	if err := database.DB.Save(&task).Error; err != nil {
		return nil, errors.WrapInternalServerError("error updating task", err)
	}

	return &task, nil
}

func DeleteTask(taskID string, userID string) error {
	var task models.Task

	taskUUID, err := parseTaskID(taskID)
	if err != nil {
		return err
	}

	// Find the task by ID
	if err := database.DB.First(&task, "id = ?", taskUUID).Error; err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return errors.ErrNotFound("task")
		}
		return errors.WrapInternalServerError("error retrieving task", err)
	}

	// Validate if the user is authorized to delete the task
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return errors.ErrInvalidID("user")
	}

	if task.CreatorID != userUUID {
		return errors.ErrUnauthorizedAction("delete", "task")
	}

	// Delete the task
	if err := database.DB.Delete(&task).Error; err != nil {
		return errors.WrapInternalServerError("error deleting task", err)
	}

	return nil
}
