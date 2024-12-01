package matchmaking

import (
	client "central/internal/client"
	"fmt"
	"log"
	"net"
	"strings"
)

type MatchmakingServer struct {
	clientStore client.Store
}

// NewMatchmakingServer initializes a new matchmaking server
func NewMatchmakingServer(store client.Store) *MatchmakingServer {
	return &MatchmakingServer{clientStore: store}
}

// Start starts the TCP matchmaking server
func (ms *MatchmakingServer) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	defer listener.Close()

	fmt.Printf("Matchmaking server listening on %s...\n", address)

	for {
		fmt.Println("POROMKAMAL")
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		go ms.handleConnection(conn)
	}
}

// handleConnection processes an individual client connection
func (ms *MatchmakingServer) handleConnection(conn net.Conn) {
	defer conn.Close()

	// Extract the IP address from the connection
	clientAddr := conn.RemoteAddr().String()
	clientIP := clientAddr
	if strings.Contains(clientAddr, "[") {
		clientIP = strings.Split(clientAddr, "]")[0]
		clientIP = strings.Trim(clientIP, "[")
	} else {
		clientIP = strings.Split(clientAddr, ":")[0]
	}

	fmt.Println("THE CLIENT IP IS: " + clientIP)

	// Check if the client IP is registered
	username, err := ms.clientStore.Read(clientIP)
	if err != nil {
		log.Printf("Unregistered client attempted to connect: %s\n", clientIP)
		conn.Write([]byte("Unauthorized\n"))
		return
	}

	if username == "" {
		log.Printf("Unregistered client attempted to connect: %s\n", clientIP)
		conn.Write([]byte("Unauthorized\n"))
		return
	}

	// Acknowledge connection
	log.Printf("Client %s connected successfully\n", clientIP)
	conn.Write([]byte("Welcome to matchmaking!\n"))

	// Keep the connection open for further communication
	for {
		// Handle incoming messages from the client
		buf := make([]byte, 1024)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Connection with client %s closed: %v\n", clientIP, err)
			break
		}

		message := strings.TrimSpace(string(buf[:n]))
		log.Printf("Received message from %s: %s\n", clientIP, message)

		if message == "exit" {
			log.Printf("Client %s disconnected\n", clientIP)
			break
		}
		conn.Write([]byte("Message received\n"))
	}
}
