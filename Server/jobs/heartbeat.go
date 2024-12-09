package jobs

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// HeartbeatJob periodically sends a heartbeat to the Central server.
type HeartbeatJob struct {
	serverURL  string
	interval   time.Duration
	registered bool // Tracks whether the service is registered
}

// readConfig reads the central server URL from a configuration file.
func readConfig(configFile string) (string, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return "", fmt.Errorf("failed to read config file: %w", err)
	}

	url := string(bytes.TrimSpace(data))
	if url == "" {
		return "", fmt.Errorf("config file is empty")
	}

	return url, nil
}

// NewHeartbeatJob creates a new HeartbeatJob instance.
func NewHeartbeatJob(interval time.Duration) (*HeartbeatJob, error) {
	url, err := readConfig("config.txt") // Assuming the config file is in the parent directory
	if err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return &HeartbeatJob{
		serverURL:  url,
		interval:   interval,
		registered: false,
	}, nil
}

// Start begins the heartbeat process in a goroutine.
func (h *HeartbeatJob) Start() {
	ticker := time.NewTicker(h.interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if !h.registered {
					h.registerService()
				} else {
					h.sendHeartbeat()
				}
			}
		}
	}()
}

// registerService sends a POST request to register the service.
func (h *HeartbeatJob) registerService() {
	resp, err := http.Post(h.serverURL+"/services", "application/json", nil) // Assumin no payload for now
	if err != nil {
		log.Printf("Failed to register service: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		h.registered = true
		log.Printf("Service successfully registered. Server responded with: %s", resp.Status)
	} else {
		log.Printf("Failed to register service. Server responded with: %s", resp.Status)
	}
}

// sendHeartbeat sends a PATCH request to the server to indicate the service is alive.
func (h *HeartbeatJob) sendHeartbeat() {
	req, err := http.NewRequest(http.MethodPatch, h.serverURL+"/services", nil) // Assuming no payload for now
	if err != nil {
		log.Printf("Failed to create heartbeat request: %v", err)
		return
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("Failed to send heartbeat: %v", err)
		return
	}
	defer resp.Body.Close()
}
