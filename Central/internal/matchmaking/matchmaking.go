package matchmaking

import (
	client "central/internal/client"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

type MatchmakingServer struct {
	clientStore client.Store
}

var (
	ACK_CONN       = []byte("ACK")
	MSG_REQ_SENT   = []byte("REQ_SENT")
	AWAITING_REQ   = []byte("AWAITING_REQ")
	USER_NOT_FOUND = []byte("USER_NOT_FOUND")
)

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
		conn, err := listener.Accept()
		if err != nil {
			fmt.Printf("Failed to accept connection: %v\n", err)
			continue
		}

		go ms.handleConnection(conn)
	}
}

func AcknowledgeConnection(conn net.Conn) {
	conn.Write(ACK_CONN)
}

func RequestSent(conn net.Conn) {
	conn.Write(MSG_REQ_SENT)
}

func AwaitingRequest(conn net.Conn) {
	conn.Write(AWAITING_REQ)
}

func UserNotFound(conn net.Conn) {
	conn.Write(USER_NOT_FOUND)
}

func GetRequestedUsername(conn net.Conn) string {
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		log.Printf("Failed to read from connection: %v\n", err)
		return ""
	}
	return strings.TrimSpace(string(buf[:n]))
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

	fmt.Println("Client with IP: " + clientIP + " connected")

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

	// Requested username from client
	req_user := GetRequestedUsername(conn)
	fmt.Println("Requested username: " + req_user)
	if req_user == "" {
		UserNotFound(conn)
		return
	}
	AcknowledgeConnection(conn)
	// Simulate it for now
	time.Sleep(2 * time.Second)
	RequestSent(conn)
	for {
		time.Sleep(3 * time.Second)
		AwaitingRequest(conn)
	}
	// Locate the peer for the client
	// // Keep the connection open for further communication
	// for {
	// 	// Handle incoming messages from the client
	// 	buf := make([]byte, 1024)
	// 	n, err := conn.Read(buf)
	// 	if err != nil {
	// 		log.Printf("Connection with client %s closed: %v\n", clientIP, err)
	// 		break
	// 	}

	// 	message := strings.TrimSpace(string(buf[:n]))
	// 	log.Printf("Received message from %s: %s\n", clientIP, message)

	// 	if message == "exit" {
	// 		log.Printf("Client %s disconnected\n", clientIP)
	// 		break
	// 	}
	// 	conn.Write([]byte("Message received\n"))
	// }
}
