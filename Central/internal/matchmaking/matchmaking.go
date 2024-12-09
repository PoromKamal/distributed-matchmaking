package matchmaking

import (
	client "central/internal/client"
	service "central/internal/service"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	mathrand "math/rand"
	"net"
	"strings"
	"time"
)

type MatchmakingServer struct {
	clientStore  client.Store
	serviceStore service.Store
}

var (
	ACK_CONN       = []byte("ACK\n")
	MSG_REQ_SENT   = []byte("REQ_SENT\n")
	AWAITING_REQ   = []byte("AWAITING_REQ\n")
	USER_NOT_FOUND = []byte("USER_NOT_FOUND\n")
	REQ_ACCEPTED   = []byte("REQ_ACCEPTED\n")
	ACCEPT_REQ     = []byte("ACCEPT_REQ")
	SERVER_ERROR   = []byte("SERVER_ERROR\n")
)

// NewMatchmakingServer initializes a new matchmaking server
func NewMatchmakingServer(store client.Store, serviceStore service.Store) *MatchmakingServer {
	return &MatchmakingServer{clientStore: store, serviceStore: serviceStore}
}

// Start starts the TCP matchmaking server
func (ms *MatchmakingServer) Start(address string) error {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	go ms.backgroundAnalysis()
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

func ServerError(conn net.Conn) {
	conn.Write(SERVER_ERROR)
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

func generateRoomId() string {
	// Format current time without spaces or special characters
	currentTime := time.Now().Format("20060102150405") // YYYYMMDDHHMMSS

	// Generate a 16-byte cryptographically random string
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		// Handle error (e.g., fall back to another random source or log)
		panic("Failed to generate cryptographically secure random string")
	}
	randomString := hex.EncodeToString(randomBytes) // Convert to hexadecimal string

	// Combine the time and random string
	roomId := fmt.Sprintf("%s-%s", currentTime, randomString)
	return roomId
}

/*
Find the best server to route the clients to, which will provide the best overall experience for
both clients.
We want to minimize the maximum latency experience by either clients.
i.e.
minimize max(latency(client1, server), latency(client2, server))

if multiple servers have the same minimum latency, we choose one at random.
*/
func compute_optimal_server(client1 map[string]float32, client2 map[string]float32) (string, error) {
	minimized_latency_servers := []string{}   // Array of servers
	serverLatency := make(map[string]float64) // Map of server to maximum latency experienced by either client
	for server, latency := range client1 {
		serverLatency[server] = float64(latency)
	}

	for server, latency := range client2 {
		serverLatency[server] = math.Max(serverLatency[server], float64(latency))
	}

	// Find the minimums
	minimum_latency := math.MaxFloat64
	minimum_combined_latency := math.MaxFloat64
	for server, latency := range serverLatency {
		// check if both clients have this server
		if _, ok := client1[server]; !ok {
			continue
		}
		if _, ok := client2[server]; !ok {
			continue
		}

		combined_latency := float64(client1[server] + client2[server])
		// TODO: Fix DRY
		if latency < minimum_latency {
			minimum_latency = latency
			minimized_latency_servers = []string{} // Reset the list
			minimized_latency_servers = append(minimized_latency_servers, server)
			minimum_combined_latency = combined_latency
		} else if latency == minimum_latency && combined_latency < minimum_combined_latency {
			minimized_latency_servers = []string{} // Reset the list
			minimized_latency_servers = append(minimized_latency_servers, server)
			minimum_combined_latency = combined_latency
		} else if latency == minimum_latency && combined_latency == minimum_combined_latency {
			minimized_latency_servers = append(minimized_latency_servers, server)
		}
	}

	if len(minimized_latency_servers) == 0 {
		return "", fmt.Errorf("no server found")
	}

	if len(minimized_latency_servers) == 1 {
		return minimized_latency_servers[0], nil
	}

	// Choose one at random
	r := mathrand.New(mathrand.NewSource(time.Now().UnixNano()))
	randomIndex := r.Intn(len(minimized_latency_servers))
	return minimized_latency_servers[randomIndex], nil
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

	// Send the server IP to both clients
	client1Delay, err := ms.clientStore.GetDelayList(username)
	client2Delay, err2 := ms.clientStore.GetDelayList(req_user)
	if err != nil || err2 != nil {
		ServerError(conn)
		ServerError(connRequest)
		return
	}
	serverIP, err := compute_optimal_server(client1Delay, client2Delay)
	if err != nil {
		ServerError(conn)
		ServerError(connRequest)
		return
	}
	roomId := generateRoomId()

	response := fmt.Sprintf("IP:%s\nRoomID:%s\n", serverIP, roomId)
	conn.Write([]byte(response))
	connRequest.Write([]byte(response))
	// Close both connections after sending the IP
	ms.clientStore.InsertChatInstance(roomId, serverIP, []string{username, req_user})
	conn.Close()
	connRequest.Close()
}

// runs every 10 seconds
// Reroutes clients to the best server
func (ms *MatchmakingServer) backgroundAnalysis() {
	ticker := time.NewTicker(4 * time.Second)
	defer ticker.Stop() // Ensure ticker is stopped when function exits

	for {
		select {
		case <-ticker.C:
			// Iterate over each server, and get the chat instances, if the server
			// hasn't sent a heartbeat in the last 10 seconds, reroute the clients
			// to the best server
			allChatInstances, err := ms.clientStore.GetAllChatInstances()
			if err != nil {
				log.Printf("Error getting chat instances: %v\n", err)
				continue
			}

			for _, instance := range allChatInstances {
				client1 := instance.Users[0]
				client2 := instance.Users[1]
				client1Delay, err := ms.clientStore.GetDelayList(client1)
				client2Delay, err2 := ms.clientStore.GetDelayList(client2)
				if err != nil || err2 != nil {
					log.Printf("Error getting delay list for clients: %v\n", err)
					continue
				}
				serverIP, err := compute_optimal_server(client1Delay, client2Delay)
				if err != nil {
					log.Printf("Error computing optimal server: %v\n", err)
					continue
				}
				if serverIP != instance.ChatServer {
					// Reroute the clients
					ms.clientStore.RemoveChatInstancesForUser(client1)
					ms.clientStore.RemoveChatInstancesForUser(client2)
					ms.clientStore.InsertChatInstance(instance.RoomId, serverIP, []string{client1, client2})
					fmt.Printf("Rerouting clients %s and %s to server %s\n", client1, client2, serverIP)

					client1IP, err := ms.clientStore.ReadByUsername(client1)
					client2IP, err2 := ms.clientStore.ReadByUsername(client2)
					if err != nil || err2 != nil {
						log.Printf("Error getting client IPs: %v\n", err)
						continue
					}

					// Redirect them to server 2
					connRedirect1, err2 := net.Dial("tcp", client1IP+":3003")
					connRedirect2, err3 := net.Dial("tcp", client2IP+":3003")
					if err2 != nil || err3 != nil {
						continue
					}
					connRedirect1.Write([]byte(serverIP))
					connRedirect2.Write([]byte(serverIP))

					connRedirect1.Close()
					connRedirect2.Close()
				}
			}
		}
	}
}
