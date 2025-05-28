package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/kfeuerschvenger/task-manager-api/dto"
	"github.com/kfeuerschvenger/task-manager-api/services"
	"github.com/kfeuerschvenger/task-manager-api/utils"
	"github.com/kfeuerschvenger/task-manager-api/validators"
)

// Register godoc
// @Summary New User Registration
// @Description Creates a new user account with the provided registration details.
// @Router /auth/register [post]
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   input body dto.RegisterRequest true "Registration details"
// @Success 201 {object} map[string]string
// @Failure 400 {object} map[string]string
func Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Invalid input")
		return
	}

	if err := validators.ValidateRegisterInput(req); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := services.RegisterUser(req)
	if err != nil {
    if strings.Contains(err.Error(), "email already registered") {
        utils.Error(w, http.StatusConflict, err.Error())
    } else {
        utils.Error(w, http.StatusBadRequest, err.Error())
    }
    return
	}

	utils.JSON(w, http.StatusCreated, map[string]string{"token": token})
}

// Login godoc
// @Summary User Login
// @Description Authenticates a user and returns a JWT token.
// @Router /auth/login [post]
// @Tags auth
// @Accept  json
// @Produce  json
// @Param   input body dto.LoginRequest true "Login details"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
func Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error(w, http.StatusBadRequest, "Invalid input")
		return
	}

	if err := validators.ValidateLoginInput(req); err != nil {
		utils.Error(w, http.StatusBadRequest, err.Error())
		return
	}

	token, err := services.AuthenticateUser(req)
	if err != nil {
		utils.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	utils.JSON(w, http.StatusOK, map[string]string{"token": token})
}