package routes

import (
	"github.com/gorilla/mux"
	"github.com/kfeuerschvenger/task-manager-api/controllers"
	_ "github.com/kfeuerschvenger/task-manager-api/docs"
	"github.com/kfeuerschvenger/task-manager-api/middleware"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

// Initializes the router and defines the API routes for the application.
func SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// Public routes
	router.HandleFunc("/ping", controllers.Ping).Methods("GET")
	router.HandleFunc("/auth/register", controllers.Register).Methods("POST")
	router.HandleFunc("/auth/login", controllers.Login).Methods("POST")
	
	// Swagger documentation route
	router.PathPrefix("/documentation/").Handler(httpSwagger.Handler(
    httpSwagger.InstanceName("swagger"),
		httpSwagger.DocExpansion("none"),
    httpSwagger.DefaultModelsExpandDepth(-1),
	))

	// Protected routes
	protected := router.PathPrefix("/tasks").Subrouter()
	protected.Use(middleware.AuthMiddleware)
	protected.HandleFunc("", controllers.GetTasks).Methods("GET")
	protected.HandleFunc("", controllers.CreateTask).Methods("POST")
	protected.HandleFunc("/{id}", controllers.GetTaskByID).Methods("GET")
	protected.HandleFunc("/{id}", controllers.UpdateTask).Methods("PUT")
	protected.HandleFunc("/{id}", controllers.DeleteTask).Methods("DELETE")

	return router
}