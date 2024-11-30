package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CentralAPI represents the REST API for the Central service.
type CentralAPI struct {
	store Store // Store interface for flexible backend storage
}

// NewCentralAPI initializes a new CentralAPI instance.
func NewCentralAPI(store Store) *CentralAPI {
	return &CentralAPI{store: store}
}

// ClientRegistrationRequest represents the payload for client registration.
type ClientRegistrationRequest struct {
	Username string `json:"username" binding:"required"`
}

// RegisterClient handles client registration (POST).
func (api *CentralAPI) RegisterClient(c *gin.Context) {
	var req ClientRegistrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON payload"})
		return
	}

	clientIP := c.ClientIP() // Gin automatically extracts the client IP
	if err := api.store.Create(clientIP, req.Username); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Client registered", "ip": clientIP, "username": req.Username})
}

// GetClient handles retrieving a client by IP (GET).
func (api *CentralAPI) GetClient(c *gin.Context) {
	clientIP := c.ClientIP()

	username, err := api.store.Read(clientIP)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ip": clientIP, "username": username})
}

func (api *CentralAPI) GetClientByUsername(c *gin.Context) {
	username := c.Param("username")

	clientIP, err := api.store.ReadByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ip": clientIP, "username": username})
}

// DeleteClient handles deleting a client by IP (DELETE).
func (api *CentralAPI) DeleteClient(c *gin.Context) {
	clientIP := c.ClientIP()

	if err := api.store.Delete(clientIP); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client deleted", "ip": clientIP})
}

func main() {
	// Initialize the in-memory store
	store := NewInMemoryStore()

	// Initialize the Central API
	api := NewCentralAPI(store)

	// Create a Gin router
	router := gin.Default()

	// Define routes and methods
	router.POST("/clients", api.RegisterClient) // POST
	router.GET("/clients", api.GetClient)       // GET
	router.GET("/clients/:username", api.GetClientByUsername)
	router.DELETE("/clients", api.DeleteClient) // DELETE

	// Start the HTTP server
	router.Run(":8080") // Default listen on :8080
}
