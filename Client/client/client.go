package client

import (
	"bytes"
	"client/service"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
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

var (
	ACK_CONN       = "ACK"
	MSG_REQ_SENT   = "REQ_SENT"
	AWAITING_REQ   = "AWAITING_REQ"
	USER_NOT_FOUND = "USER_NOT_FOUND"
	SERVER_ERROR   = "SERVER_ERROR"
)

func (c *Client) StartMatchmaking(username string, statusChannel chan string) error {
	matchMakingPort := "8081"
	matchMakingAddress := c.CentralURL[:len(c.CentralURL)-4] + matchMakingPort
	matchMakingAddress = strings.Replace(matchMakingAddress, "http://", "", 1)
	fmt.Println("Matchmaking address: ", matchMakingAddress)
	conn, err := net.Dial("tcp", matchMakingAddress) // Establish a connection to the matchmaking server
	if err != nil {
		statusChannel <- SERVER_ERROR
		return fmt.Errorf("failed to connect to matchmaking server: %w", err)
	}
	defer conn.Close()

	// Send initial message to the server
	_, err = conn.Write([]byte(username))
	if err != nil {
		statusChannel <- SERVER_ERROR
		return fmt.Errorf("failed to send initial message: %w", err)
	}

	// Continuously read messages from the server
	// go func() {
	for {
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				fmt.Println("Connection closed by server.")
			} else {
				fmt.Printf("Error reading from server: %v\n", err)
			}
			break
		}

		message := string(buf[:n])
		// break up the messages by delimiter (newline)
		messages := strings.Split(message, "\n")
		for _, msg := range messages {
			if msg == "" {
				continue
			}
			statusChannel <- msg
		}
	}
	//}()
	return nil
}

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

func GetClient() *Client {
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
			registrationResult := <-clientInstance.Register()
			if registrationResult != nil {
				fmt.Println("Failed to register with Central!")
				os.Exit(1)
			}
			clientInstance.Initialize()
		}
	}
	return clientInstance
}

func handleConnection(conn net.Conn){

}

// Listen for message requests on port 3001
func messageRequestListener(){
	listener, err := net.Listen("tcp", ":3001")
	if err != nil {
		fmt.Println("failed to start message request listener: %w", err)
		os.Exit(0)
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			// just eat it for now
			//fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}
		go handleConnection(conn)
	}
}

func (c *Client) Initialize() <-chan error {
	// Create a channel to communicate the result
	resultChan := make(chan error)

	// Start the initialization in a goroutine
	go func() {
		// Create channels for server fetching
		serverChan := make(chan []string)
		errorChan := make(chan error)

		// Fetch servers asynchronously
		go func() {
			servers, err := c.serverRegistryAPI.GetServers()
			if err != nil {
				errorChan <- fmt.Errorf("failed to initialize client: %w", err)
				return
			}
			serverChan <- servers
		}()

		// Logic a bit sketchy
		select {
		case servers := <-serverChan:
			for _, server := range servers {
				c.ServerRegistry[server] = math.MaxFloat32
			}
			c.startPingJob(10 * time.Second)
			resultChan <- nil
		case err := <-errorChan:
			resultChan <- err
		}

		// Close the result channel
		close(resultChan)
	}()

	return resultChan
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
func (c *Client) Register() <-chan error {
	result := make(chan error)
	go func() {
		defer close(result)

		// Read the central server URL from the configuration file
		url := c.CentralURL

		// Prepare the request payload
		payload := map[string]string{"username": c.UserName}
		jsonPayload, err := json.Marshal(payload)
		if err != nil {
			//fmt.Printf("Error marshaling payload: %v\n", err)
			result <- err
			return
		}

		// Make the POST request to the central server
		resp, err := http.Post(fmt.Sprintf("%s/clients", url), "application/json", bytes.NewBuffer(jsonPayload))
		if err != nil {
			//fmt.Printf("Error sending POST request: %v\n", err)
			result <- err
			return
		}
		defer resp.Body.Close()

		// Handle the response, we are ok with conflict.
		if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
			//body, _ := io.ReadAll(resp.Body)
			//fmt.Printf("Server returned error: %s\n", string(body))
			result <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return
		}

		//body, _ := io.ReadAll(resp.Body)

		//fmt.Println("Successfully registered with Central!")
		//fmt.Println(string(body))
		result <- nil
	}()
	return result
}
