package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

// Broadcast UDP port
const broadcastPort = 9999

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Prompt for username
	fmt.Print("Enter your username: ")
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}
	username = strings.TrimSpace(username)

	// Clear the previous line and display the welcome message
	fmt.Print("\033[1A") // Move cursor up one line
	fmt.Print("\033[2K") // Clear the line
	fmt.Printf("Welcome to FastChat, %s!\n", username)

	// Start broadcaster and listener in goroutines
	go startBroadcaster()
	go startListener()

	// Keep the program running
	select {}
}

// Broadcast IP address over the network
func startBroadcaster() {
	broadcastAddr := fmt.Sprintf("10.255.255.255:%d", broadcastPort)
	conn, err := net.Dial("udp", broadcastAddr)
	if err != nil {
		log.Fatalf("Broadcast connection failed: %v", err)
	}
	defer conn.Close()

	for {
		localAddr := conn.LocalAddr().(*net.UDPAddr)
		message := fmt.Sprintf("User:%s IP:%s", "FastChat", localAddr.IP.String())
		_, err := conn.Write([]byte(message))
		if err != nil {
			log.Printf("Broadcast error: %v", err)
		}
		log.Printf("Broadcasting: %s", message)
		time.Sleep(5 * time.Second)
	}
}

// Listen for broadcasts
func startListener() {
	addr := net.UDPAddr{
		Port: broadcastPort,
		IP:   net.IPv4zero,
	}

	conn, err := net.ListenUDP("udp", &addr)
	if err != nil {
		log.Fatalf("Listener setup failed: %v", err)
	}
	defer conn.Close()

	buf := make([]byte, 1024)
	for {
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Read error: %v", err)
			continue
		}
		log.Printf("Received: %s from %s", string(buf[:n]), remoteAddr)
	}
}
