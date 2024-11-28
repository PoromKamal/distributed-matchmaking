package main

import (
	"bufio"
	routers "fastchat/network"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	// Prompt for the username and overwrite the line each time
	go routers.StartListening()
	fmt.Print("Enter your username: ")
	reader := bufio.NewReader(os.Stdin)
	username, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read username: %v", err)
	}
	username = strings.TrimSpace(username)
	fmt.Print("\033[1A") // Move cursor up one line
	fmt.Print("\033[2K") // Clear the line
	// Overwrite the previous prompt line
	fmt.Printf("\rWelcome to FastChat, %s!\n", username)

	// Prompt for the recipient and overwrite the line each time
	fmt.Print("Enter the username of the person you want to chat with: ")
	recipient, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("Failed to read recipient: %v", err)
	}
	recipient = strings.TrimSpace(recipient)
	fmt.Print("\033[1A") // Move cursor up one line
	fmt.Print("\033[2K") // Clear the line

	// Start loading animation for "Starting chat with..."
	fmt.Printf("\rStarting chat with %s ", recipient)
	loading := []string{"|", "/", "-", "\\"}
	for { // Limit the number of iterations to avoid infinite loop
		for _, char := range loading {
			fmt.Print("\033[1D")               // Move the cursor one character back
			fmt.Print(char)                    // Print the next character in the animation
			time.Sleep(100 * time.Millisecond) // Delay for the animation effect
		}
	}

	// Final message after animation
	fmt.Print("\033[1A") // Move cursor up one line
	fmt.Print("\033[2K") // Clear the line
	fmt.Printf("\rChat with %s started!\n", recipient)
}
