package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type ServerRegistryAPI interface {
	GetServers() ([]string, error)
}

type CentralServerRegistry struct {
	serverURL string
}

// NewCentralServerRegistry creates a new instance of CentralServerRegistry.
func NewCentralServerRegistry(serverURL string) *CentralServerRegistry {
	return &CentralServerRegistry{
		serverURL: serverURL,
	}
}

// Define a struct that matches the structure of the response JSON
type ServicesResponse struct {
	Services []string `json:"services"` // Match the "services" key
}

// GetServers fetches the list of servers from the Central server.
func (c *CentralServerRegistry) GetServers() ([]string, error) {
	resp, err := http.Get(c.serverURL + "/services")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch servers: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Debugging output: printing the raw JSON
	fmt.Println("Raw response body:", string(body))

	// Parse the JSON response into the struct
	var response ServicesResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Return the services (list of servers)
	return response.Services, nil
}
