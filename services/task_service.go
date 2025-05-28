package services

import (
	"time"

	"github.com/google/uuid"
	"github.com/kfeuerschvenger/task-manager-api/database"
	"github.com/kfeuerschvenger/task-manager-api/dto"
	"github.com/kfeuerschvenger/task-manager-api/errors"
	"github.com/kfeuerschvenger/task-manager-api/models"
)

func CreateTask(input dto.CreateTaskInput, creatorID string) (models.Task, error) {
	creatorUUID, err := uuid.Parse(creatorID)
	if err != nil {
		return models.Task{}, errors.ErrInvalidID("user")
	}

	assigneeUUID, err := uuid.Parse(input.AssigneeID)
	if err != nil {
		return models.Task{}, errors.ErrInvalidID("assignee")
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

	err = database.DB.Create(&task).Error
	return task, err
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

	if err := database.DB.Where("id = ? AND (creator_id = ? OR assignee_id = ?)", taskID, userUUID, userUUID).First(&task).Error; err != nil {
		return nil, errors.ErrNotFound("task")
	}

	return &task, nil
}

func UpdateTask(taskID string, userID string, dto dto.UpdateTaskDTO) (*models.Task, error) {
	var task models.Task

	// Find the task by ID
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, errors.ErrNotFound("task")
	}

	// Validate if the user is authorized to update the task
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.ErrInvalidID("user")
	}

	if task.CreatorID != userUUID {
		return nil, errors.ErrUnauthorizedAction("update", "task")
	}

	// Aply updates from the DTO
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
		task.AssigneeID = assigneeUUID
	}

	task.UpdatedAt = time.Now()

	if err := database.DB.Save(&task).Error; err != nil {
		return nil, err
	}

	return &task, nil
}

func DeleteTask(taskID string, userID string) error {
	var task models.Task

	// Find the task by ID
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return errors.ErrNotFound("task")
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
		return err
	}

	return nil
}