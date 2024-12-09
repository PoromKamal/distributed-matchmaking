package main

import (
	ClientAPI "central/internal/client"
	"central/internal/matchmaking"
	ServiceAPI "central/internal/service"

	"github.com/gin-gonic/gin"
)

func main() {
	// Initialize stores and API
	clientStore := ClientAPI.GetInMemoryStore()
	clientAPI := ClientAPI.NewClientAPI(clientStore)
	serviceStore := ServiceAPI.GetInMemoryStore()
	serviceAPI := ServiceAPI.NewServiceAPI(serviceStore)
	matchmakingService := matchmaking.NewMatchmakingServer(clientStore, serviceStore)

	// Create Gin router
	router := gin.Default()

	// Register Client API
	clientAPI.RegisterRoutes(router)
	serviceAPI.RegisterRoutes(router)

	// Start the HTTP server
	go matchmakingService.Start(":8081")
	router.Run(":8080")
}
