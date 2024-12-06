package handlers

import (
	"bytes"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"umani-service/app/internal/config"
	"umani-service/app/internal/db"
	"umani-service/app/internal/models"

	"github.com/gin-gonic/gin"
)

func HandleNotification(cfg config.Config, db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {

		// Read request body
		if c.ContentType() != "application/x-www-form-urlencoded" {
			slog.Default().Error("Invalid Content-Type:", c.ContentType())
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Content-Type"})
			return
		}

		// Parse form data
		var notification models.Notification
		if err := c.ShouldBind(&notification); err != nil {
			slog.Default().Error("Failed to bind form data:", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse notification"})
			return
		}

		// Log parsed notification
		slog.Default().Info("Parsed notification:", notification)

		// Generate SHA-1 hash
		expectedHash := generateSHA1Hash(notification, cfg.SecretWord)
		if notification.Sha1Hash != expectedHash {
			slog.Default().Error("Invalid signature. Expected: " + expectedHash + ", Got: " + notification.Sha1Hash)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
			return
		}

		if notification.NotificationType == "p2p-incoming" || notification.NotificationType == "card-incoming" {
			if !notification.Unaccepted {
				completedOrder := models.CompletedOrder{
					OrderID:          notification.OperationId,
					OperationID:      notification.OperationId,
					Sender:           notification.Sender,
					Amount:           notification.Amount,
					Currency:         notification.Currency,
					Status:           true,
					Sha1Hash:         notification.Sha1Hash,
					TestNotification: notification.TestNotification,
					Label:            notification.Label,
					Handle:           "completed",
				}

				jsonBody, err := json.Marshal(completedOrder)
				if err != nil {
					slog.Default().Error("Error encoding completedOrder to JSON:", err)
					return
				}

				req, err := http.NewRequest("POST", cfg.SendURL, bytes.NewBuffer(jsonBody))
				if err != nil {
					slog.Default().Error("Error creating HTTP request:", err)
					saveFailedNotification(db, completedOrder)
					return
				}
				req.Header.Set("Content-Type", "application/json")

				client := &http.Client{}
				resp, err := client.Do(req)
				if err != nil || resp.StatusCode >= 400 {
					slog.Default().Error("Error sending HTTP request or bad response:", err, resp.Status)
					saveFailedNotification(db, completedOrder)
					c.JSON(http.StatusOK, completedOrder)
					return
				}
				defer resp.Body.Close()

				slog.Default().Info("Response status:", resp.Status)
				c.JSON(http.StatusOK, completedOrder)
				return
			}
		}

		ignoreModel := models.CompletedOrder{
			OrderID:          notification.OperationId,
			OperationID:      notification.OperationId,
			Sender:           notification.Sender,
			Amount:           notification.Amount,
			Currency:         notification.Currency,
			Status:           false,
			Sha1Hash:         notification.Sha1Hash,
			TestNotification: notification.TestNotification,
			Label:            notification.Label,
			Handle:           "ignored",
		}

		// Log the ignored notification
		slog.Default().Info("Notification ignored: Type="+notification.NotificationType, "Unaccepted=", notification.Unaccepted)

		// Respond with the ignoreModel
		c.JSON(http.StatusOK, ignoreModel)

		// Encode the ignoreModel to JSON
		jsonBody, err := json.Marshal(ignoreModel)
		if err != nil {
			slog.Default().Error("Error encoding ignoreModel to JSON:", err)
			return
		}

		// Create a new HTTP POST request
		req, err := http.NewRequest("POST", cfg.SendURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			slog.Default().Error("Error creating HTTP request:", err)
			return
		}
		req.Header.Set("Content-Type", "application/json")

		// Optionally, send the request using an HTTP client
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			slog.Default().Error("Error sending HTTP request:", err)
			return
		}
		defer resp.Body.Close()

		slog.Default().Info("Response status:", resp.Status)
	}
}

func generateSHA1Hash(n models.Notification, secret string) string {
	codepro, err := strconv.ParseBool(n.Codepro)
	if err != nil {
		slog.Default().Error("Invalid Codepro value:", n.Codepro)
		return ""
	}

	data := fmt.Sprintf(
		"%s&%s&%s&%s&%s&%s&%t&%s&%s",
		n.NotificationType, // notification_type
		n.OperationId,      // operation_id
		n.Amount,           // amount
		n.Currency,         // currency
		n.DateTime,         // datetime
		n.Sender,           // sender
		codepro,            // codepro
		secret,             // секретное слово
		n.Label,            // label
	)

	hash := sha1.New()
	hash.Write([]byte(data))
	return hex.EncodeToString(hash.Sum(nil))
}

func saveFailedNotification(database *sql.DB, order models.CompletedOrder) {
	err := db.SaveUnsentNotification(database, order)
	if err != nil {
		slog.Default().Error("Failed to save unsent notification:", err)
	}
}
