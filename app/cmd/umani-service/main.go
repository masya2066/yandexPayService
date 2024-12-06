package main

import (
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"log"
	"os"
	"umani-service/app/internal/config"
	"umani-service/app/internal/consumer"
	"umani-service/app/internal/db"
	"umani-service/app/internal/handlers"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	cfg := config.LoadConfig()

	initDB, err := db.InitDatabase(os.Getenv("SQLITE_PATH"))
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer initDB.Close()

	go consumer.RetryFailedNotifications(cfg, initDB)

	router := gin.Default()

	yandex := router.Group("/yandex")
	{
		order := yandex.Group("/order")
		{
			order.POST("/create", handlers.CreateOrder(cfg))
			order.POST("/notification", handlers.HandleNotification(cfg, initDB))
		}
	}
	log.Printf("Starting server on port %s...", cfg.AppPort)
	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
