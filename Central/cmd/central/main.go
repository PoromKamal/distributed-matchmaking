package main

import (
	ClientAPI "central/internal/client"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize stores
	clientStore := ClientAPI.NewInMemoryStore()
	clientAPI := ClientAPI.NewClientAPI(clientStore)

	// Create Gin router
	router := gin.Default()

	// Register Client API
	clientAPI.RegisterRoutes(router)

	// Start the HTTP server
	router.Run(":8080")
}
