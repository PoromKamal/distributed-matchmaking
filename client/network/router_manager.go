package routers

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Controller represents the controller's information
type Controller struct {
	IP            string
	OneWayDelayMS int64
}

const (
	listenPort = 9999
	signature  = "CHAT_CONTROLLER"
)

// ControllerManager holds the list of controllers and manages concurrency
type ControllerManager struct {
	mu              sync.Mutex
	controllers     []Controller
	controllersChan chan<- Controller
}

var (
	instance *ControllerManager
	once     sync.Once
)

// NewControllerManager ensures that only one instance of ControllerManager is created
func NewControllerManager(controllersChan chan<- Controller) *ControllerManager {
	once.Do(func() {
		instance = &ControllerManager{
			controllersChan: controllersChan,
		}
	})
	return instance
}

// AddController adds a new controller to the list if it's not already present
func (cm *ControllerManager) AddController(controller Controller) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Check if the controller already exists in the list
	for _, existingController := range cm.controllers {
		if existingController.IP == controller.IP {
			// Update latency if the controller is already in the list
			existingController.OneWayDelayMS = controller.OneWayDelayMS
			return
		}
	}

	// Add new controller to the list
	cm.controllers = append(cm.controllers, controller)
}

// GetLowestLatencyController retrieves the controller with the lowest latency
func (cm *ControllerManager) GetLowestLatencyController() *Controller {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.controllers) == 0 {
		return nil
	}

	// Find the controller with the minimum latency
	lowestLatencyController := cm.controllers[0]
	for _, controller := range cm.controllers {
		if controller.OneWayDelayMS < lowestLatencyController.OneWayDelayMS {
			lowestLatencyController = controller
		}
	}

	return &lowestLatencyController
}

// StartListening starts the UDP listener in the background to listen for broadcasts
func StartListening() {
	// Get the singleton instance of ControllerManager
	cm := NewControllerManager(nil)

	// Resolve UDP address to listen for broadcasts
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%d", listenPort))
	if err != nil {
		log.Fatalf("Failed to resolve address: %v", err)
	}

	// Create UDP socket for listening
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Failed to listen on UDP port: %v", err)
	}
	defer conn.Close()

	log.Printf("Listening for broadcasts on port %d...", listenPort)

	// Store discovered controllers
	buffer := make([]byte, 1024)

	for {
		// Read incoming UDP packet
		n, remoteAddr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("Error reading UDP packet: %v", err)
			continue
		}

		// Process the packet
		message := string(buffer[:n])
		parts := strings.Split(message, "|")
		if len(parts) == 3 && parts[0] == signature {
			controllerIP := parts[1]
			timestampStr := parts[2]

			// Parse timestamp from the message
			sentTime, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				log.Printf("Invalid timestamp in message: %s", message)
				continue
			}

			// Calculate one-way delay (assuming relatively synchronized clocks)
			now := time.Now().UnixNano()
			oneWayDelay := (now - sentTime) / 1_000_000 // Convert nanoseconds to milliseconds

			// Create the controller object
			controller := Controller{
				IP:            controllerIP,
				OneWayDelayMS: oneWayDelay,
			}

			fmt.Println("Received controller:", controller)
			// Get the singleton instance of ControllerManager
			cm.AddController(controller)

			// Optionally, send the controller information to the main thread
			if cm.controllersChan != nil {
				cm.controllersChan <- controller
			}
		} else {
			log.Printf("Received non-controller message from %s: %s", remoteAddr, message)
		}
	}
}
