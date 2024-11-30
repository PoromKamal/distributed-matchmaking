package router_response

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	serverPort    = 5500
	responseDelay = "CHAT_SERVER_RESPONSE" // Signature for responses
)

// StartServer starts the chat server on a specified port and listens for pings.
func StartRouterListener() {
	// Resolve the server address
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	// Create the UDP connection
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP port %d: %v", serverPort, err)
	}
	defer conn.Close()

	log.Printf("Chat server started on port %d. Waiting for pings...", serverPort)

	// Buffer for incoming data
	buffer := make([]byte, 1024)

	for {
		// Read incoming packet
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP packet: %v", err)
			continue
		}

		message := string(buffer[:n])
		parts := strings.Split(message, "|")

		if len(parts) == 2 && parts[0] == "CHAT_CONTROLLER" {
			// Parse the ping arrival time
			arrivalTimeStr := parts[1]
			arrivalTime, err := strconv.ParseInt(arrivalTimeStr, 10, 64)
			if err != nil {
				log.Printf("Invalid arrival time in message: %s", message)
				continue
			}

			// Calculate round-trip time in milliseconds
			now := time.Now().UnixNano()
			roundTripTime := (now - arrivalTime) / 1_000_000 // Convert nanoseconds to milliseconds

			// Prepare the response
			response := fmt.Sprintf("%s|%d|%d", responseDelay, roundTripTime, now)

			// Send the response back to the controller
			_, err = conn.WriteToUDP([]byte(response), remoteAddr)
			if err != nil {
				log.Printf("Failed to send response: %v", err)
			} else {
				log.Printf("Responded to controller at %s with round-trip time: %d ms", remoteAddr.String(), roundTripTime)
			}
		} else {
			log.Printf("Received invalid or unrecognized message from %s: %s", remoteAddr.String(), message)
		}
	}
}
