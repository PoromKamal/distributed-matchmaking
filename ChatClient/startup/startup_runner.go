/*
Responsible for starting up the client which includes:
1. Displaying startup message
2. Retrieving username from the user
3. Registering the client with Central
*/
package startup_runner

import (
	"fastchat/client"
	"fmt"
	"os"
)

func StartupClient() {
	clientInstance := client.GetInstance()
	fmt.Println("Welcome to FastChat!")
	fmt.Print("Enter your username: ")
	fmt.Scanln(&clientInstance.UserName)
	clearLastLine()
	clearLastLine()
	fmt.Printf("Hello, %s! Connecting to FastChat...\n", clientInstance.UserName)
	registrationResult := <-clientInstance.Register()
	clearLastLine()
	if registrationResult {
		fmt.Println("Connected to FastChat!")
	} else {
		fmt.Println("Failed to connect to FastChat!")
		os.Exit(1)
	}
}

func clearLastLine() {
	// fmt.Print("\033[1A")
	// fmt.Print("\033[2K")
}
