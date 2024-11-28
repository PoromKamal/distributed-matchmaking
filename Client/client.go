package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	listenPort = 9999
	signature  = "CHAT_CONTROLLER"
)

type Controller struct {
	IP            string
	OneWayDelayMS int64
}

func main() {
	// Prompt the user for their username
	fmt.Print("Enter your username: ")
	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	if err != nil {
		log.Fatalf("Failed to read username: %v", err)
	}
	username = strings.TrimSpace(username) // Remove trailing newline or spaces

	fmt.Printf("\rWelcome to FastChat, %s!\n", username)

	// Resolve UDP address to listen for broadcasts
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	// Create UDP socket for listening
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP port: %v", err)
	}
	defer conn.Close()

	log.Printf("Listening for broadcasts on port %d...", listenPort)

	// Store discovered controllers
	var controllers []Controller
	buffer := make([]byte, 1024)

	for {
		// Read incoming UDP packet
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP packet: %v", err)
			continue
		}

		// Process the packet
		message := string(buffer[:n])
		parts := strings.Split(message, "|")
		if len(parts) == 3 && parts[0] == signature {
			controllerIP := parts[1]
			timestampStr := parts[2]

			// Parse timestamp from the message
			sentTime, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				log.Printf("Invalid timestamp in message: %s", message)
				continue
			}

			// Calculate one-way delay (assuming relatively synchronized clocks)
			now := time.Now().UnixNano()
			oneWayDelay := (now - sentTime) / 1_000_000 // Convert nanoseconds to milliseconds

			// Add to controllers list if not already present
			found := false
			for _, c := range controllers {
				if c.IP == controllerIP {
					found = true
					break
				}
			}

			if !found {
				controller := Controller{
					IP:            controllerIP,
					OneWayDelayMS: oneWayDelay,
				}
				controllers = append(controllers, controller)

				log.Printf("Discovered controller: %+v", controller)
			} else {
				log.Printf("Controller %s already in list", controllerIP)
			}
		} else {
			log.Printf("Received non-controller message from %s: %s", remoteAddr, message)
		}
	}
}
