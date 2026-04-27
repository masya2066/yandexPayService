package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"umani-service/app/internal/b2pay"
	"umani-service/app/internal/config"
	"umani-service/app/internal/consumer"
	"umani-service/app/internal/db"
	"umani-service/app/internal/handlers"
)

func main() {
	// .env optional; if missing, env vars come from the shell
	if err := godotenv.Load(); err != nil {
		log.Printf("godotenv: %v (continuing without .env)", err)
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
	cardlink := router.Group("/cardlink")
	{
		order := cardlink.Group("/order")
		{
			order.POST("/create", handlers.CreateOrderCardLink(cfg))
			order.POST("/notification", handlers.HandleCardlinkNotification(cfg, initDB))
		}
	}
	b2payClient := b2pay.NewClient()
	b2p := router.Group("/b2pay")
	{
		order := b2p.Group("/order")
		{
			order.POST("/create", handlers.CreateOrderB2Pay(cfg, b2payClient))
			order.POST("/notification", handlers.HandleB2PayNotification(cfg, initDB))
			order.GET("/:id/status", handlers.B2PayTransactionStatus(cfg, b2payClient))
		}
	}
	log.Printf("Starting server on port %s...", cfg.AppPort)
	if err := router.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
