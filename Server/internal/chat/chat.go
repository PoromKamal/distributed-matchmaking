package chat

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"
)

type ChatManager struct {
	Port        string
	clients     map[string][]net.Conn // Room ID -> list of clients
	clientMutex sync.Mutex            // Mutex to protect access to the clients map
}

// NewChatManager initializes a new ChatManager with the specified port
func NewChatManager(port string) *ChatManager {
	return &ChatManager{
		Port:    port,
		clients: make(map[string][]net.Conn),
	}
}

// Start initializes the chat server
func (cm *ChatManager) Start() {
	listener, err := net.Listen("tcp", cm.Port)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		return
	}
	defer listener.Close()

	fmt.Printf("Chat server started on port %s...\n", cm.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Error accepting connection: %v\n", err)
			continue
		}

		fmt.Printf("New client connected: %s\n", conn.RemoteAddr().String())

		// Handle client in a separate goroutine
		go cm.handleClient(conn)
	}
}

func (cm *ChatManager) handleClient(conn net.Conn) {
	defer conn.Close()

	clientIp := conn.RemoteAddr().String()
	reader := bufio.NewReader(conn)

	// Read the initial message (username#roomId)
	input, err := reader.ReadString('\n')
	if err != nil {
		log.Printf("Error reading from client %s: %v\n", clientIp, err)
		return
	}

	// Trim the newline and parse the username and roomId
	input = strings.TrimSpace(input)
	parts := strings.SplitN(input, "#", 2)
	if len(parts) != 2 {
		log.Printf("Invalid input format from client %s: %s\n", clientIp, input)
		return
	}

	username, roomId := parts[0], parts[1]
	log.Printf("Client %s (%s) joined room %s\n", username, clientIp, roomId)

	// Add the client to the appropriate room
	cm.clientMutex.Lock()
	cm.clients[roomId] = append(cm.clients[roomId], conn)
	cm.clientMutex.Unlock()

	// Listen for messages from the client
	buffer := make([]byte, 1024) // Buffer for receiving messages
	for {
		n, err := conn.Read(buffer)
		if err != nil {
			fmt.Printf("Error reading from client: %v\n", err)
			break
		}

		message := string(buffer[:n])
		if message == "" {
			continue
		}

		timer := time.NewTimer(2 * time.Second)
		serverIp := conn.LocalAddr().String()
		go func() {
			<-timer.C
			cm.broadcastMessage("server", roomId, "Pong from server "+serverIp+"\n")
		}()

		// Broadcast the message to all clients in the same roomId
		cm.broadcastMessage(username, roomId, message)
	}
}

func (cm *ChatManager) broadcastMessage(username, roomId, message string) {
	cm.clientMutex.Lock()
	defer cm.clientMutex.Unlock()

	// Send the message to all clients in the specified roomId
	clientsInRoom, ok := cm.clients[roomId]
	if !ok {
		return // No clients in this room
	}

	for _, client := range clientsInRoom {
		message := fmt.Sprintf("%s: %s\n", username, message)
		clientIp := client.RemoteAddr().String()
		if _, err := client.Write([]byte(message)); err != nil {
			fmt.Printf("Error sending message to client %s: %v\n", clientIp, err)
		} else {
			fmt.Printf("Broadcasted '%s' to room %s '%s'\n", clientIp, message, roomId)
		}
	}
}
