package main

import (
	"chatserver/jobs"
	"log"
	"time"
)

func main() {
	// Create the Heartbeat job
	heartbeat, err := jobs.NewHeartbeatJob(3 * time.Second)
	if err != nil {
		log.Fatalf("Error initializing Heartbeat job: %v", err)
	}

	// Start the Heartbeat job
	heartbeat.Start()

	// Keep the application running
	select {}
}
