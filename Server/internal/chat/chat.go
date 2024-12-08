package chat

import (
	"fmt"
	"net"
	"time"
)

type ChatManager struct {
	Port string
}

// NewChatManager initializes a new ChatManager with the specified port
func NewChatManager(port string) *ChatManager {
	return &ChatManager{
		Port: port,
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

	// Send "pong" to the client every 5 seconds
	for {
		_, err := conn.Write([]byte("pong\n"))
		if err != nil {
			fmt.Printf("Error writing to client: %v\n", err)
			return
		}

		fmt.Printf("Sent 'pong' to client: %s\n", conn.RemoteAddr().String())
		time.Sleep(5 * time.Second)
	}
}
