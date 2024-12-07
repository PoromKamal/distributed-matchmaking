#!/bin/bash

# Function to display usage
display_usage() {
  echo "Usage: $0 [--run]"
  echo "  --run    Build and run the application"
  exit 1
}

# Check if go is installed
if ! command -v go &> /dev/null; then
  echo "Error: Go is not installed. Please install Go and try again."
  exit 1
fi

# Define variables for build output and main file
BIN_DIR="bin"
OUTPUT="$BIN_DIR/chat-client"
MAIN_FILE="main.go"

# Create bin directory if it does not exist
if [ ! -d "$BIN_DIR" ]; then
  mkdir -p "$BIN_DIR"
fi

# Build the application
echo "Building the application..."
go build -o "$OUTPUT" "$MAIN_FILE"
if [ $? -ne 0 ]; then
  echo "Build failed. Exiting."
  exit 1
fi

echo "Build succeeded. Output: $OUTPUT"

# Check for --run argument
if [ "$1" == "--run" ]; then
  echo "Running the application..."
  ./$OUTPUT
  if [ $? -ne 0 ]; then
    echo "Error: Failed to run the application."
    exit 1
  fi
fi
