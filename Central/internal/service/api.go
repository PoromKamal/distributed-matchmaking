package serviceapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

/*
API for handling service discovery for chat servers
*/
type ServiceAPI struct {
	store Store
}

func NewServiceAPI(store Store) *ServiceAPI {
	return &ServiceAPI{store: store}
}

func (api *ServiceAPI) RegisterRoutes(router *gin.Engine) {
	group := router.Group("/services")
	{
		group.POST("", api.RegisterService)
		group.GET("", api.GetServices)
		group.PATCH("", api.PatchService)
		group.DELETE("", api.DeleteService)
	}
}

func (api *ServiceAPI) RegisterService(c *gin.Context) {
	clientIP := c.ClientIP()
	if err := api.store.Create(clientIP); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Service registered", "ip": clientIP})
}

func (api *ServiceAPI) GetServices(c *gin.Context) {
	ips, err := api.store.Read()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"services": ips})
}

func (api *ServiceAPI) PatchService(c *gin.Context) {
	clientIP := c.ClientIP()
	if _, err := api.store.Patch(clientIP); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service patched", "ip": clientIP})
}

func (api *ServiceAPI) DeleteService(c *gin.Context) {
	clientIP := c.ClientIP()
	if err := api.store.Delete(clientIP); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Service deleted", "ip": clientIP})
}
