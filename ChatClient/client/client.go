package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
)

type Client struct {
	UserName   string
	CentralURL string
}

var lock = &sync.Mutex{}
var clientInstance *Client

func GetInstance() *Client {
	if clientInstance == nil {
		lock.Lock()
		defer lock.Unlock()
		if clientInstance == nil {
			url, err := readConfig("client/config.txt")
			if err != nil {
				fmt.Printf("Error reading config: %v\n", err)
				return nil
			}
			clientInstance = &Client{
				CentralURL: url,
			}
		}
	}
	return clientInstance
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

		// Handle the response
		if resp.StatusCode != http.StatusCreated {
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
