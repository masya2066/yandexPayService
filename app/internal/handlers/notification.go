package handlers

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"net/http"
	"umani-service/app/internal/config"
	"umani-service/app/internal/models"

	"github.com/gin-gonic/gin"
)

func HandleNotification(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Failed to read request body: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}
		log.Printf("Request body: %s", body)

		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		var notification models.Notification

		// Parse request body
		if err := c.BindJSON(&notification); err != nil {
			log.Printf("Invalid notification payload: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification payload"})
			return
		}

		// Generate expected hash
		expectedHash := generateSHA1Hash(notification, cfg.SecretWord) // Use secret word
		if notification.Sha1Hash != expectedHash {
			log.Printf("Invalid signature. Expected: %s, Got: %s", expectedHash, notification.Sha1Hash)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			return
		}

		// Handler successful payment
		if notification.NotificationType == "p2p-incoming" && !notification.Unaccepted {
			log.Printf("Payment received: Label=%s, Amount=%s", notification.Label, notification.Amount)
			c.JSON(http.StatusOK, gin.H{"message": "Notification processed"})
			return
		}

		// Logging ignored notifications
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
