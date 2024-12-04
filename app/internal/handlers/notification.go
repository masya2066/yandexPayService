package handlers

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"umani-service/app/internal/config"
	"umani-service/app/internal/models"

	"github.com/gin-gonic/gin"
)

func HandleNotification(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var notification models.Notification

		if err := c.BindJSON(&notification); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid notification"})
			return
		}

		expectedHash := generateSHA1Hash(notification, cfg.Receiver)
		if notification.Sha1Hash != expectedHash {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			return
		}

		// Обработка успешного платежа
		if notification.NotificationType == "p2p-incoming" && !notification.Unaccepted {
			slog.Default().Error("Payment received for label: %s, amount: %s\n", notification.Label, notification.Amount)
			c.JSON(http.StatusOK, gin.H{"message": "Notification processed"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"message": "Notification ignored"})
	}
}

// Генерация SHA-1 подписи
func generateSHA1Hash(n models.Notification, secret string) string {
	data := fmt.Sprintf("%s&%s&%s&%s&%s&%s&%t&%s&%s",
		n.NotificationType, n.OperationId, n.Amount, n.Currency, n.DateTime,
		n.Sender, n.Codepro, secret, n.Label)
	hash := sha1.New()
	io.WriteString(hash, data)
	return hex.EncodeToString(hash.Sum(nil))
}
