package main

import (
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

	// Start the Heartbeat job
	heartbeat.Start()
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
