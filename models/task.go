package models

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	ID          uuid.UUID `gorm:"type:uuid;default:uuid_generate_v4();primaryKey"`
	Title       string    `gorm:"not null"`
	Description string    `gorm:"not null"`
	DueDate     time.Time `gorm:"not null"`

	Priority   string    `gorm:"not null;default:'medium'"`
	Status     string    `gorm:"not null;default:'pending'"`
	CreatorID  uuid.UUID `gorm:"type:uuid;not null"`
	AssigneeID uuid.UUID `gorm:"type:uuid;not null"`

	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}