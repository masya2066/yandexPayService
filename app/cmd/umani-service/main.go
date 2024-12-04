package main

import (
	"log"
	"umani-service/app/internal/handlers"

	"github.com/gin-gonic/gin"
	"umani-service/app/internal/config"
)

func main() {
	cfg := config.LoadConfig()

	router := gin.Default()

	yandex := router.Group("/yandex")
	{
		order := yandex.Group("/order")
		{
			order.POST("/create", handlers.CreateOrder(cfg))
			order.POST("/notification", handlers.HandleNotification(cfg))
		}
	}
	log.Printf("Starting server on port %s...", cfg.AppPort)
	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
