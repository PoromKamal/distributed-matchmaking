package main

import (
	"fmt"
	"log"
	"net"

	tls "tls_poc/tls"
)

func main() {
	listener, err := net.Listen("tcp", ":3137")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	fmt.Println("Server listening on port 3137")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println(err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	config := tls.TLSConnectionConfig{
		Conn:     conn,
		IsServer: true,
		CertPath: "server.crt",
		KeyPath:  "server.key",
	}
	tlsConn, _ := tls.NewTLSConn(&config)

	if err := tlsConn.HandshakeServerController(); err != nil {
		log.Println("Code is done..:", err)
		return
	}
	fmt.Println("Execution Done")

	for {
		buf := make([]byte, 1024)
		n, err := tlsConn.Read(buf)
		if err != nil {
			log.Println("Error reading data:", err)
			break
		}
		message := string(buf[:n])
		fmt.Println("Received from client:", message)

		_, err = tlsConn.Write([]byte(message))
		if err != nil {
			log.Println("Error writing data:", err)
			break
		}
		fmt.Println("Echoed message back to client")
	}
}
