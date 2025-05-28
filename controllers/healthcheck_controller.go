package controllers

import (
	"net/http"

	"github.com/kfeuerschvenger/task-manager-api/utils"
)

// Ping godoc
// @Summary Health Check
// @Description Checks the health of the API service.
// @Router /ping [get]
// @Tags healthcheck
// @Accept  plain
// @Produce  plain
// @Success 200 {string} string "Pong"
func Ping(w http.ResponseWriter, r *http.Request) {
    utils.PlainText(w, http.StatusOK, "Pong")
}
