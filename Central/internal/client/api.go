package clientapi

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ClientAPI represents the REST API for the Client service.
type ClientAPI struct {
	store Store
}

func NewClientAPI(store Store) *ClientAPI {
	return &ClientAPI{store: store}
}

// RegisterRoutes sets up client-related routes.
func (api *ClientAPI) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/clients")
	{
		group.POST("", api.RegisterClient)
		group.GET("", api.GetClient)
		group.GET("/:username", api.GetClientByUsername)
		group.DELETE("", api.DeleteClient)
	}
}

// ClientRegistrationRequest represents the payload for client registration.
type ClientRegistrationRequest struct {
	Username string `json:"username" binding:"required"`
}

// RegisterClient handles client registration (POST).
func (api *ClientAPI) RegisterClient(c *gin.Context) {
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

	fmt.Println("REGISTERED CLIENT WITH IP: ", clientIP)

	c.JSON(http.StatusCreated, gin.H{"message": "Client registered", "ip": clientIP, "username": req.Username})
}

// GetClient handles retrieving a client by IP (GET).
func (api *ClientAPI) GetClient(c *gin.Context) {
	clientIP := c.ClientIP()
	username, err := api.store.Read(clientIP)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ip": clientIP, "username": username})
}

func (api *ClientAPI) GetClientByUsername(c *gin.Context) {
	username := c.Param("username")

	clientIP, err := api.store.ReadByUsername(username)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ip": clientIP, "username": username})
}

// DeleteClient handles deleting a client by IP (DELETE).
func (api *ClientAPI) DeleteClient(c *gin.Context) {
	clientIP := c.ClientIP()

	if err := api.store.Delete(clientIP); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Client deleted", "ip": clientIP})
}
