package main

import startupRunner "fastchat/startup"

func main() {
	startupRunner.StartupClient()
	select {}
}
