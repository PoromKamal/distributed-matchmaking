package client

import (
	"sync"
	"time"
)

type Client struct {
	UserName string
}

var lock = &sync.Mutex{}
var clientInstance *Client

func GetInstance() *Client {
	if clientInstance == nil {
		lock.Lock()
		defer lock.Unlock()
		if clientInstance == nil {
			clientInstance = &Client{}
		}
	}
	return clientInstance
}

func (c *Client) Register() <-chan bool {
	// TODO: Implement Central Registration
	result := make(chan bool)
	go func() {
		time.Sleep(2 * time.Second)
		result <- true
	}()
	return result
}
