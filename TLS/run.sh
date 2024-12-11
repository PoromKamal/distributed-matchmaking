#!/bin/bash

cleanup() {
  echo "Cleaning up..."
  kill "$server_pid" 2>/dev/null
  exit
}

trap cleanup SIGINT SIGTERM EXIT

go run server.go &
server_pid=$!

(
  wait "$server_pid"
  echo "server.go has terminated. Exiting."
  cleanup
) &

go run client.go

cleanup
