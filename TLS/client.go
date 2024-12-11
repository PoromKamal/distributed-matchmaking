package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"

	tls "tls_poc/tls"
)

func main() {
	conn, err := net.Dial("tcp", "localhost:3137")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	config := tls.TLSConnectionConfig{
		Conn:     conn,
		IsServer: false,
		CertPath: "client.crt",
		KeyPath:  "client.key",
	}
	tlsConn, err := tls.NewTLSConn(&config)

	if err := tlsConn.HandshakeClientController(); err != nil {
		log.Println("Handshake failed:", err)
		return
	}

	fmt.Println("Done: Enjoy your private echo chat!.")
	fmt.Println("\n\n\n\n\n")
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter messages (type 'exit' to quit):")
	for {
		fmt.Print("> ")
		message, _ := reader.ReadString('\n')
		message = message[:len(message)-1]

		if message == "exit" {
			break
		}

		_, err = tlsConn.Write([]byte(message))
		if err != nil {
			log.Println("Error writing data:", err)
			return
		}
		fmt.Println("Sent to server:", message)

		buf := make([]byte, 1024)
		n, err := tlsConn.Read(buf)
		if err != nil {
			log.Println("Error reading data:", err)
			return
		}
		response := string(buf[:n])
		fmt.Println("Received from server:", response)
	}
}
