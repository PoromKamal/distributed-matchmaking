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
	ACK_CONN       = []byte("ACK\n")
	MSG_REQ_SENT   = []byte("REQ_SENT\n")
	AWAITING_REQ   = []byte("AWAITING_REQ\n")
	USER_NOT_FOUND = []byte("USER_NOT_FOUND\n")
	REQ_ACCEPTED   = []byte("REQ_ACCEPTED\n")
	ACCEPT_REQ     = []byte("ACCEPT_REQ")
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

func RequestAccepted(conn net.Conn) {
	conn.Write(REQ_ACCEPTED)
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

func (ms *MatchmakingServer) requestMatch(username string, conn net.Conn, requestChannel chan string) {
	// Send the request to the client, with the requesters username
	_, err := conn.Write([]byte(username + "\n"))
	if err != nil {
		requestChannel <- "Match declined"
		log.Printf("Failed to send request to client: %v\n", err)
		return
	}

	// Listen for a response from the client
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		requestChannel <- "Match declined"
		log.Printf("Failed to read response from client: %v\n", err)
		return
	}
	response_raw := string(buf[:n])
	response_raw = strings.TrimSpace(response_raw)
	if response_raw == string(ACCEPT_REQ) {
		// Match accepted
		requestChannel <- string(ACCEPT_REQ)
	} else {
		// Match declined
		requestChannel <- "Match declined"
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
	req_user_ip, err := ms.clientStore.ReadByUsername(req_user)
	if err != nil {
		UserNotFound(conn)
		return
	}

	requestChannel := make(chan string)
	// hack for local testing
	if req_user_ip == "::1" {
		req_user_ip = "localhost"
	}
	connRequest, err2 := net.Dial("tcp", req_user_ip+":3001")
	if err2 != nil {
		log.Printf("Failed to connect to client: %v\n", err2)
		return
	}
	go ms.requestMatch(username, connRequest, requestChannel)
	RequestSent(conn)
loop:
	for {
		select {
		case result := <-requestChannel:
			if result == string(ACCEPT_REQ) {
				RequestAccepted(conn)
				break loop
			} else {
				UserNotFound(conn)
				break loop
			}
		default:
			AwaitingRequest(conn)
		}
		// Debounce the loop by 50 ms
		time.Sleep(50 * time.Millisecond)
	}

	// Find a server match
	for {
		// Keep the connection open
	}
}
