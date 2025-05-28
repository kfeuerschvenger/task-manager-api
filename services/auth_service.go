package services

import (
	"strings"

	"github.com/kfeuerschvenger/task-manager-api/database"
	"github.com/kfeuerschvenger/task-manager-api/dto"
	"github.com/kfeuerschvenger/task-manager-api/errors"
	"github.com/kfeuerschvenger/task-manager-api/models"
	"github.com/kfeuerschvenger/task-manager-api/utils"
	"gorm.io/gorm"
)

func RegisterUser(req dto.RegisterRequest) (string, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	var existingUser models.User
	result := database.DB.Where("email = ?", email).First(&existingUser)
	if result.Error == nil {
		return "", errors.NewConflictError("email already registered")
	} else if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
		return "", errors.NewInternalServerError("database error")
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return "", errors.NewInternalServerError("error hashing password")
	}

	user := models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     email,
		Password:  hashedPassword,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return "", errors.NewInternalServerError("error creating user")
	}

	token, err := utils.GenerateJWT(user.ID.String())
	if err != nil {
		return "", errors.NewInternalServerError("error generating token")
	}

	return token, nil
}

func AuthenticateUser(req dto.LoginRequest) (string, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	result := database.DB.Where("email = ?", email).First(&user)
	if result.Error != nil {
		return "", errors.NewAuthError("invalid credentials")
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return "", errors.NewAuthError("invalid credentials")
	}

	token, err := utils.GenerateJWT(user.ID.String())
	if err != nil {
		return "", errors.NewInternalServerError("error generating token")
	}

	return token, nil
}