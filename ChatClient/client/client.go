package client

import (
	"bytes"
	"encoding/json"
	"fastchat/service"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"sync"
	"time"
)

type Client struct {
	UserName          string
	CentralURL        string
	serverRegistryAPI service.ServerRegistryAPI
	ServerRegistry    map[string]float32
}

var lock = &sync.Mutex{}
var clientInstance *Client

// PingServer pings a server and calculates the two-way delay.
func pingServer(serverIP string) (float32, error) {
	fmt.Println("Pinging server %s...\n", serverIP)
	if serverIP == "::1" {
		serverIP = "localhost"
	}
	start := time.Now()
	conn, err := net.DialTimeout("tcp", serverIP+":3000", 2*time.Second) // Adjust the port and timeout as needed.
	if err != nil {
		return 0, fmt.Errorf("failed to ping server %s: %w", serverIP, err)
	}
	defer conn.Close()

	delay := time.Since(start).Seconds() * 1000 // Convert to milliseconds
	return float32(delay), nil
}

// StartPingJob starts a background job to ping servers and update the registry.
func (c *Client) startPingJob(interval time.Duration) {
	fmt.Println("POROMK MALA")
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				c.updateServerDelays()
			}
		}
	}()
}

// updateServerDelays updates the delay values in the ServerRegistry.
func (c *Client) updateServerDelays() {
	for serverIP := range c.ServerRegistry {
		delay, err := pingServer(serverIP)
		if err != nil {
			fmt.Printf("Error pinging server %s: %v\n", serverIP, err)
			continue
		}

		// Update the delay in the ServerRegistry
		lock.Lock()
		c.ServerRegistry[serverIP] = delay
		lock.Unlock()

		fmt.Printf("Updated delay for server %s: %.2f ms\n", serverIP, delay)
	}
}

func GetInstance() *Client {
	if clientInstance == nil {
		lock.Lock()
		defer lock.Unlock()
		if clientInstance == nil {
			// Basic initialization without network calls
			url, err := readConfig("client/config.txt")
			if err != nil {
				fmt.Printf("Error reading config: %v\n", err)
				return nil
			}

			clientInstance = &Client{
				CentralURL:        url,
				ServerRegistry:    make(map[string]float32), // Empty for now
				serverRegistryAPI: service.NewCentralServerRegistry(url),
			}
		}
	}
	return clientInstance
}

func (c *Client) Initialize() error {
	// Create a channel to receive the servers or error
	serverChan := make(chan []string)
	errorChan := make(chan error)

	// Fetch servers asynchronously in a goroutine
	go func() {
		servers, err := c.serverRegistryAPI.GetServers()
		fmt.Print(servers)
		if err != nil {
			errorChan <- fmt.Errorf("failed to initialize client: %w", err)
			return
		}
		serverChan <- servers
	}()

	// Wait for either servers or an error
	select {
	case servers := <-serverChan:
		// Populate ServerRegistry with initial values
		for _, server := range servers {
			c.ServerRegistry[server] = math.MaxFloat32
		}
		// Start the ping job
		c.startPingJob(10 * time.Second)
		return nil
	case err := <-errorChan:
		// Handle any error that occurred during the async GetServers call
		return err
	}
}

// ReadConfig reads the central server URL from a configuration file
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

// Register sends the username to the central server's register endpoint
func (c *Client) Register() <-chan bool {
	result := make(chan bool)
	go func() {
		defer close(result)

		// Read the central server URL from the configuration file
		url := c.CentralURL

		// Prepare the request payload
		payload := map[string]string{"username": c.UserName}
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			fmt.Printf("Error marshaling payload: %v\n", err)
			result <- false
			return
		}

		// Make the POST request to the central server
		resp, err := http.Post(fmt.Sprintf("%s/clients", url), "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			fmt.Printf("Error sending POST request: %v\n", err)
			result <- false
			return
		}
		defer resp.Body.Close()

		// Handle the response, we are ok with conflict.
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
			body, _ := io.ReadAll(resp.Body)
			fmt.Printf("Server returned error: %s\n", string(body))
			result <- false
			return
		}

		fmt.Println("Successfully registered with Central!")
		result <- true
	}()
	return result
}
