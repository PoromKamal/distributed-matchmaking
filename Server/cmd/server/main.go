package main

import (
	"chatserver/internal/chat"
	"chatserver/jobs"
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create the Heartbeat job
	heartbeat, err := jobs.NewHeartbeatJob(3 * time.Second)
	if err != nil {
		log.Fatalf("Error initializing Heartbeat job: %v", err)
	}
	chatManager := chat.NewChatManager(":3002")

	// Start the Heartbeat job
	heartbeat.Start()
	go chatManager.Start()
	// Initialize Gin router
	r := gin.Default()

	// Start the Gin server on port 3000
	go func() {
		if err := r.Run(":3000"); err != nil {
			log.Fatalf("Error starting server: %v", err)
		}
	}()
	// Keep the application running
	select {}
}
