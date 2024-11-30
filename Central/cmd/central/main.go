package main

import (
	ClientAPI "central/internal/client"
	ServiceAPI "central/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize stores and API
	clientStore := ClientAPI.NewInMemoryStore()
	clientAPI := ClientAPI.NewClientAPI(clientStore)
	serviceStore := ServiceAPI.NewInMemoryStore()
	serviceAPI := ServiceAPI.NewServiceAPI(serviceStore)

	// Create Gin router
	router := gin.Default()

	// Register Client API
	clientAPI.RegisterRoutes(router)
	serviceAPI.RegisterRoutes(router)

	// Start the HTTP server
	router.Run(":8080")
}
