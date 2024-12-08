package chat

import (
	"fmt"
	"log"
	"net"
	"sync"
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

	// Get the roomId from the client (you can modify this to match the message format)
	var roomId string
	fmt.Fscanf(conn, "%s\n", &roomId)
	log.Printf("Client %s joined room %s\n", conn.RemoteAddr().String(), roomId)

	// Add the client to the appropriate room
	cm.clientMutex.Lock()
	cm.clients[roomId] = append(cm.clients[roomId], conn)
	cm.clientMutex.Unlock()

	// Send "pong" to the client every 5 seconds
	//pongCt := 0
	// go func() {
	// 	for {
	// 		message := fmt.Sprintf("pong%d\n", pongCt)
	// 		_, err := conn.Write([]byte(message))
	// 		if err != nil {
	// 			fmt.Printf("Error writing to client: %v\n", err)
	// 			return
	// 		}
	// 		pongCt += 1

	// 		fmt.Printf("Sent '%s' to client: %s\n", message, conn.RemoteAddr().String())
	// 		time.Sleep(1 * time.Second)
	// 	}
	// }()

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

		// Broadcast the message to all clients in the same roomId
		cm.broadcastMessage(roomId, message)
	}
}

func (cm *ChatManager) broadcastMessage(roomId, message string) {
	cm.clientMutex.Lock()
	defer cm.clientMutex.Unlock()

	// Send the message to all clients in the specified roomId
	clientsInRoom, ok := cm.clients[roomId]
	if !ok {
		return // No clients in this room
	}

	for _, client := range clientsInRoom {
		if _, err := client.Write([]byte(message)); err != nil {
			fmt.Printf("Error sending message to client: %v\n", err)
		} else {
			fmt.Printf("Broadcasted '%s' to room '%s'\n", message, roomId)
		}
	}
}
