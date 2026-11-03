package services

import (
	stderrors "errors"
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
	} else if !stderrors.Is(result.Error, gorm.ErrRecordNotFound) {
		return "", errors.WrapInternalServerError("database error", result.Error)
	}

	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return "", errors.WrapInternalServerError("error hashing password", err)
	}

	user := models.User{
		FirstName: req.FirstName,
		LastName:  req.LastName,
		Email:     email,
		Password:  hashedPassword,
	}

	if err := database.DB.Create(&user).Error; err != nil {
		return "", errors.WrapInternalServerError("error creating user", err)
	}

	token, err := utils.GenerateJWT(user.ID.String())
	if err != nil {
		return "", errors.WrapInternalServerError("error generating token", err)
	}

	return token, nil
}

func AuthenticateUser(req dto.LoginRequest) (string, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	result := database.DB.Where("email = ?", email).First(&user)
	if result.Error != nil {
		if stderrors.Is(result.Error, gorm.ErrRecordNotFound) {
			// Spend the same bcrypt time as a real comparison so response timing does not reveal registered emails.
			utils.BurnPasswordCheck(req.Password)
			return "", errors.NewAuthError("invalid credentials")
		}
		return "", errors.WrapInternalServerError("database error", result.Error)
	}

	if !utils.CheckPasswordHash(req.Password, user.Password) {
		return "", errors.NewAuthError("invalid credentials")
	}

	token, err := utils.GenerateJWT(user.ID.String())
	if err != nil {
		return "", errors.WrapInternalServerError("error generating token", err)
	}

	return token, nil
}
