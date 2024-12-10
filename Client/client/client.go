package client

import (
	"bytes"
	"client/service"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Client struct {
	UserName              string
	CentralURL            string
	serverRegistryAPI     service.ServerRegistryAPI
	ServerRegistry        map[string]float32
	messageRequestChannel chan string
	ChatRequests          map[string]net.Conn
	currentChatConn       net.Conn
	CurrentChatServer     string
	currentRoomId         string
}

var lock = &sync.Mutex{}
var chatLock = &sync.Mutex{}
var clientInstance *Client

var (
	ACK_CONN       = "ACK"
	MSG_REQ_SENT   = "REQ_SENT"
	AWAITING_REQ   = "AWAITING_REQ"
	USER_NOT_FOUND = "USER_NOT_FOUND"
	SERVER_ERROR   = "SERVER_ERROR"
	ACCEPT_REQ     = "ACCEPT_REQ"
)

func (c *Client) SendMessage(message string) {
	if c.currentChatConn == nil {
		fmt.Println("No chat connection established")
		return
	}
	chatLock.Lock()
	_, err := c.currentChatConn.Write([]byte(message + "\n"))
	chatLock.Unlock()
	if err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
	}
}

// StartChat connects to the server and handles sending and receiving messages.
func (c *Client) StartChat(messages chan string, serverAddress string, roomId string) {
	// Connect to the server
	conn, err := net.Dial("tcp", serverAddress+":3002")
	if err != nil {
		fmt.Printf("Failed to connect to server at %s: %v\n", serverAddress, err)
		return
	}

	chatLock.Lock()
	c.currentChatConn = conn
	c.CurrentChatServer = serverAddress
	c.currentRoomId = roomId
	chatLock.Unlock()
	// Send the room ID to the server
	_, err = conn.Write([]byte(fmt.Sprintf("%s#%s\n", c.UserName, roomId)))
	if err != nil {
		fmt.Printf("Failed to send room ID: %v\n", err)
		return
	}
	messages <- "START_CHAT"
	// Listen for messages from the server
	go func() {
		buffer := make([]byte, 1024)
		incomplete := ""
		for {
			n, err := c.currentChatConn.Read(buffer)
			//c.currentChatConn.Write([]byte("ACK\n"))
			if err != nil {
				// Just eat the error
				continue
			}
			//fmt.Printf("Got message: %s\n", string(buffer[:n]))
			//time.Sleep(1 * time.Second)
			// Append to incomplete data
			incomplete += string(buffer[:n])

			// Split messages based on newline
			messagesArr := splitMessages(&incomplete, '\n')
			for _, msg := range messagesArr {
				messages <- msg
			}
		}
	}()

	go func() {
		// listen for server relocations
		conn, err := net.Listen("tcp", ":3003")
		if err != nil {
			fmt.Println("Failed to start chat switch listener")
			return
		}
		defer conn.Close()
		for {
			serverConn, err := conn.Accept()
			if err != nil {
				fmt.Println("Failed to accept connection")
				continue
			}

			// Read the new server address
			buf := make([]byte, 1024)
			n, err := serverConn.Read(buf)
			if err != nil {
				fmt.Printf("Failed to read from server: %v\n", err)
				continue
			}
			newServerAddress := string(buf[:n])
			newServerAddress = strings.TrimSuffix(newServerAddress, "\n")
			newConn, err := net.Dial("tcp", newServerAddress+":3002")
			if err != nil {
				fmt.Printf("Failed to connect to new server: %v\n", err)
				continue
			}

			//chatLock.Lock()
			c.CurrentChatServer = newServerAddress
			c.currentChatConn = newConn

			// Register the client and send roomId to the new server
			// Send the room ID to the server
			_, err = clientInstance.currentChatConn.Write([]byte(fmt.Sprintf("%s#%s\n",
				clientInstance.UserName,
				clientInstance.currentRoomId)))
			if err != nil {
				fmt.Printf("Failed to send room ID: %v\n", err)
				continue
			}
		}
	}()
}

// Utility to split messages based on a delimiter and handle leftover data
func splitMessages(data *string, delimiter rune) []string {
	parts := []string{}
	lastIndex := 0

	for i, char := range *data {
		if char == delimiter {
			parts = append(parts, (*data)[lastIndex:i])
			lastIndex = i + 1
		}
	}

	// Keep any incomplete message
	*data = (*data)[lastIndex:]
	return parts
}

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

func (c *Client) reportDelaysToCentral() {
	// Prepare the request payload
	payload := map[string]interface{}{
		"username": c.UserName,
		"delays":   c.ServerRegistry,
	}

	// Serialize the payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error serializing payload: %v", err)
		return
	}

	// Send the HTTP POST request to the central server
	url := c.CentralURL + "/clients/delays"
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the HTTP request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error sending request to central server: %v", err)
		return
	}
	defer resp.Body.Close()
}

// updateServerDelays updates the delay values in the ServerRegistry.
func (c *Client) updateServerDelays() {
	// Fetch all the servers again
	servers, err := c.serverRegistryAPI.GetServers()
	if err != nil {
		return
	}

	newServerMap := make(map[string]float32)
	for _, server := range servers {
		newServerMap[server] = math.MaxFloat32
	}
	c.ServerRegistry = newServerMap

	for serverIP := range c.ServerRegistry {
		delay, err := pingServer(serverIP)
		if err != nil {
			// the server is likely down, just set it to max float32
			delay = math.MaxFloat32
		}

		// Update the delay in the ServerRegistry
		lock.Lock()
		c.ServerRegistry[serverIP] = delay
		lock.Unlock()

		//fmt.Printf("Updated delay for server %s: %.2f ms\n", serverIP, delay)

		// Putting this inside the loop so we can provide updated ping lists earlier
		c.reportDelaysToCentral()
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
				CentralURL:            url,
				ServerRegistry:        make(map[string]float32), // Empty for now
				serverRegistryAPI:     service.NewCentralServerRegistry(url),
				messageRequestChannel: make(chan string),
				ChatRequests:          make(map[string]net.Conn),
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

func (c *Client) AcceptMessageRequest(username string, statusChannel chan string) {
	conn, exists := c.ChatRequests[username]
	if !exists {
		fmt.Println("FATAL: Could not find user in chat requests")
		os.Exit(1) // blow up for now, get better error handling later.
		return
	}

	// send a ACCEPT_REQ message to Central
	_, err := conn.Write([]byte(ACCEPT_REQ + "\n"))
	if err != nil {
		fmt.Printf("Failed to send ACCEPT_REQ message: %v\n", err)
		os.Exit(1) // let's just blow up for now
	}
	statusChannel <- ACCEPT_REQ

	// Wait for server to send you an chat server to connect to
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("Failed to read server: %v\n", err)
		os.Exit(1) // let's just blow up for now
	}
	serverResponse := string(buf[:n])
	if !strings.HasPrefix(serverResponse, "IP:") {
		statusChannel <- SERVER_ERROR
		return
	}
	// Split on newline
	responseTokens := strings.Split(serverResponse, "\n")
	serverAddress := responseTokens[0]
	roomId := responseTokens[1]

	serverAddress = strings.TrimPrefix(serverAddress, "IP:")
	serverAddress = strings.TrimSuffix(serverAddress, "\n")

	if !strings.HasPrefix(roomId, "RoomID:") {
		statusChannel <- SERVER_ERROR
		return
	}
	roomId = strings.TrimPrefix(roomId, "RoomID:")
	roomId = strings.TrimSuffix(roomId, "\n")
	// remove new line

	statusChannel <- serverAddress
	statusChannel <- roomId
}

func handleChatRequest(conn net.Conn) {
	// Read the username from the connection
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		fmt.Printf("Failed to read from connection: %v\n", err)
		os.Exit(1) // blow up for now
	}
	username := string(buf[:n])
	clientInstance.ChatRequests[username] = conn
}

// Listen for message requests on port 3001
func messageRequestListener() {
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
		go handleChatRequest(conn)
	}
}

func (c *Client) Initialize() <-chan error {
	// Create a channel to communicate the result
	resultChan := make(chan error)
	go messageRequestListener()
	//go StartChatSwitchListener()
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
			c.startPingJob(3 * time.Second)
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
