package handlers

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"umani-service/app/internal/config"
	"umani-service/app/internal/models"

	"github.com/gin-gonic/gin"
)

func HandleNotification(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read request body
		if c.ContentType() != "application/x-www-form-urlencoded" {
			log.Printf("Invalid Content-Type: %s", c.ContentType())
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Content-Type"})
			return
		}

		// Parse form data
		var notification models.Notification
		if err := c.ShouldBind(&notification); err != nil {
			log.Printf("Failed to bind form data: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse notification"})
			return
		}

		// Log parsed notification
		log.Printf("Parsed notification: %+v", notification)

		// Generate SHA-1 hash
		expectedHash := generateSHA1Hash(notification, cfg.SecretWord)
		if notification.Sha1Hash != expectedHash {
			log.Printf("Invalid signature. Expected: %s, Got: %s", expectedHash, notification.Sha1Hash)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			return
		}

		// Handle success notification
		if notification.NotificationType == "p2p-incoming" || notification.NotificationType == "card-incoming" {
			if !notification.Unaccepted {
				log.Printf("Payment received: Label=%s, Amount=%s", notification.Label, notification.Amount)
				c.JSON(http.StatusOK, gin.H{"message": "Notification processed"})
				return
			}
		}

		// Ignore notification
		log.Printf("Notification ignored: Type=%s, Unaccepted=%v", notification.NotificationType, notification.Unaccepted)
		c.JSON(http.StatusOK, gin.H{"message": "Notification ignored"})
	}
}

func generateSHA1Hash(n models.Notification, secret string) string {
	data := fmt.Sprintf(
		"%s&%s&%s&%s&%s&%s&%t&%s&%s",
		n.NotificationType,
		n.OperationId,
		n.Amount,
		n.Currency,
		n.DateTime,
		n.Sender,
		n.Codepro,
		secret, // Secret word
		n.Label,
	)

	hash := sha1.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}
