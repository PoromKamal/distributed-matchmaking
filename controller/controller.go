package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

const (
	broadcastPort = 9999
	message       = "CHAT_CONTROLLER"
	delayDuration = 2 * time.Second // Artificial delay (e.g., 2 seconds)
)

func main() {
	// Resolve UDP address for broadcast
	broadcastAddr := "255.255.255.255:9999"
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

	// Periodically broadcast the message with a timestamp and artificial delay
	for {
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		timestamp := time.Now().UnixNano() // Send current timestamp in nanoseconds
		payload := fmt.Sprintf("%s|%s|%d", message, localAddr.IP.String(), timestamp)

		// Introduce artificial delay before sending the message
		log.Printf("Introducing artificial delay of %v", delayDuration)
		time.Sleep(delayDuration)

		_, err := conn.Write([]byte(payload))
		if err != nil {
			log.Printf("Failed to send broadcast: %v", err)
		} else {
			log.Printf("Broadcasted: %s", payload)
		}
		time.Sleep(5 * time.Second) // Broadcast every 5 seconds
	}
}
