package main

import clientrunner "client/runner"

func main() {
	clientrunner.NewClientRunner().Start()
	go func() {
		for {
			// Do nothing, just loop forever
		}
	}()
	select {}
}
