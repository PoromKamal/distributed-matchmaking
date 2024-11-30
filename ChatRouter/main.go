package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	broadcastPort   = 9999
	message         = "CHAT_CONTROLLER"
	delayDuration   = 2 * time.Second // Artificial delay (e.g., 2 seconds)
	defaultChatPort = "5500"          // Default chat server port
)

type ChatServer struct {
	IP            string
	Port          string
	OneWayDelayMS int64
}

func main() {
	// Parse CLI arguments
	env := flag.String("env", "", "Specify environment (mn for Mininet)")
	serverList := flag.String("servers", "", "Comma-separated list of chat server IPs (e.g., 192.168.1.1,192.168.1.2)")
	flag.Parse()

	// Resolve broadcast address
	broadcastAddr := ""
	if *env == "mn" {
		broadcastAddr = "10.255.255.255:9999"
	} else {
		broadcastAddr = "255.255.255.255:9999"
	}
	log.Printf("Broadcasting to %s...", broadcastAddr)
	addr, err := net.ResolveUDPAddr("udp", broadcastAddr)
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	// Create UDP connection for broadcasting
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		log.Fatalf("Failed to create UDP connection: %v", err)
	}
	defer conn.Close()

	// Parse chat servers from CLI or use default
	chatServers := parseChatServers(*serverList)
	log.Printf("Monitoring chat servers: %+v", chatServers)

	// Periodically ping chat servers and broadcast
	for {
		// Maintain delay list by pinging chat servers
		updateDelays(chatServers)

		// Broadcast information
		broadcastPayload(conn, chatServers)

		time.Sleep(5 * time.Second) // Broadcast every 5 seconds
	}
}

// parseChatServers parses a list of server IPs from CLI or defaults to localhost
func parseChatServers(serverList string) []ChatServer {
	if serverList == "" {
		// Default to a single chat server on localhost
		return []ChatServer{{IP: "127.0.0.1", Port: defaultChatPort}}
	}

	servers := strings.Split(serverList, ",")
	chatServers := make([]ChatServer, len(servers))
	for i, server := range servers {
		chatServers[i] = ChatServer{IP: strings.TrimSpace(server), Port: defaultChatPort}
	}
	return chatServers
}

// updateDelays pings each server and updates the one-way delay
func updateDelays(chatServers []ChatServer) {
	for i, server := range chatServers {
		startTime := time.Now()
		serverAddr := fmt.Sprintf("%s:%s", server.IP, server.Port)

		// Resolve server address
		addr, err := net.ResolveUDPAddr("udp", serverAddr)
		if err != nil {
			log.Printf("Failed to resolve chat server address %s: %v", serverAddr, err)
			continue
		}

		// Create UDP connection
		conn, err := net.DialUDP("udp", nil, addr)
		if err != nil {
			log.Printf("Failed to connect to chat server %s: %v", serverAddr, err)
			continue
		}

		// Send ping
		payload := fmt.Sprintf("%s|%d", message, startTime.UnixNano())
		_, err = conn.Write([]byte(payload))
		if err != nil {
			log.Printf("Failed to send ping to %s: %v", serverAddr, err)
			conn.Close()
			continue
		}

		// Wait for response
		buffer := make([]byte, 1024)
		conn.SetReadDeadline(time.Now().Add(2 * time.Second)) // 2-second timeout
		n, _, err := conn.ReadFromUDP(buffer)
		conn.Close()
		if err != nil {
			log.Printf("Failed to receive response from %s: %v", serverAddr, err)
			continue
		}

		// Parse response and calculate delay
		response := string(buffer[:n])
		parts := strings.Split(response, "|")
		if len(parts) == 3 && parts[0] == "CHAT_SERVER_RESPONSE" {
			sentTime, _ := strconv.ParseInt(parts[2], 10, 64)
			roundTripTime := time.Now().UnixNano() - sentTime
			chatServers[i].OneWayDelayMS = roundTripTime / 2 / 1_000_000 // Convert to milliseconds
			log.Printf("Updated delay for server %s: %d ms", server.IP, chatServers[i].OneWayDelayMS)
		}
	}
}

// broadcastPayload broadcasts the chat controller message with delay information
func broadcastPayload(conn *net.UDPConn, chatServers []ChatServer) {
	// Combine server information into a single message
	serverInfo := ""
	for _, server := range chatServers {
		serverInfo += fmt.Sprintf("%s:%s|%d;", server.IP, server.Port, server.OneWayDelayMS)
	}
	payload := fmt.Sprintf("%s|%s", message, serverInfo)

	// Introduce artificial delay before broadcasting
	log.Printf("Introducing artificial delay of %v", delayDuration)
	time.Sleep(delayDuration)

	_, err := conn.Write([]byte(payload))
	if err != nil {
		log.Printf("Failed to send broadcast: %v", err)
	} else {
		log.Printf("Broadcasted: %s", payload)
	}
}
