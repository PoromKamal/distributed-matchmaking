package main

import (
	"chatserver/router_response"
	"fmt"
)

func main() {
	fmt.Println("ChatServer")
	router_response.StartRouterListener()
}
