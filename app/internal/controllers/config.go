package controllers

import (
	"net/http"
	"umani-service/app/internal/config"

	"github.com/gin-gonic/gin"
)

type ConfigController struct{}

func NewConfigController() *ConfigController {
	return &ConfigController{}
}

func (cc *ConfigController) UpdateConfig(c *gin.Context) {
	var newConfig config.Config
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.SaveConfig(newConfig)
	c.JSON(http.StatusOK, gin.H{"message": "Configuration updated successfully"})
}

func (cc *ConfigController) GetConfig(c *gin.Context) {
	currentConfig := config.GetConfig()
	c.JSON(http.StatusOK, currentConfig)
}
