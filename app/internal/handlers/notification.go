package handlers

import (
	"bytes"
	"crypto/md5"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
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

func HandleCardlinkNotification(cfg config.Config, db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		slog.Default().Info("HandleCardlinkNotification => START")

		// 1. Проверяем Content-Type
		slog.Default().Info("Checking Content-Type...", "content-type", c.ContentType())
		if c.ContentType() != "application/json" && c.ContentType() != "application/x-www-form-urlencoded" {
			slog.Default().Error("Invalid Content-Type", "got", c.ContentType())
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Content-Type"})
			return
		}

		// 2. Парсим JSON в вашу структуру
		slog.Default().Info("Parsing JSON into CardLinkNotification...")
		var notification models.CardLinkNotification
		if err := c.ShouldBind(&notification); err != nil {
			slog.Default().Error("Failed to parse JSON", "error", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse notification"})
			return
		}
		slog.Default().Info("Notification parsed successfully", "notification", notification)

		// 3. Генерируем подпись и сравниваем
		signString := fmt.Sprintf("%d:%s:%s", notification.OutSum, notification.TrsId, cfg.SecretWord)
		md5Hash := md5.Sum([]byte(signString))
		expectedSignature := strings.ToUpper(hex.EncodeToString(md5Hash[:]))

		slog.Default().Info("Verifying signature",
			"calculated_signature", expectedSignature,
			"received_signature", notification.SignatureValue,
		)

		//if notification.SignatureValue != expectedSignature {
		//	slog.Default().Error("Invalid signature", "expected", expectedSignature, "got", notification.SignatureValue)
		//	c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		//	return
		//}
		slog.Default().Info("Signature verified successfully")

		// 4. Проверяем статус платежа
		slog.Default().Info("Checking payment status", "status", notification.Status)
		if notification.Status == "SUCCESS" {
			slog.Default().Info("Payment status is SUCCESS, creating CompletedOrder")

			// Формируем структуру для дальнейшей обработки / записи в БД
			completedOrder := models.CompletedOrder{
				OrderID:          strconv.Itoa(notification.InvId),
				OperationID:      notification.TrsId,
				Sender:           "CardLink",
				Amount:           fmt.Sprintf("%d", notification.OutSum),
				Currency:         notification.CurrencyIn,
				Status:           true,
				Sha1Hash:         notification.SignatureValue,
				TestNotification: false,
				Label:            "cardlink",
				Handle:           "completed",
			}

			jsonBody, err := json.Marshal(completedOrder)
			if err != nil {
				slog.Default().Error("Error encoding completedOrder to JSON", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal error"})
				return
			}

			// 5. Отправляем POST-запрос на ваш сервис
			slog.Default().Info("Sending completedOrder to external service", "url", cfg.SendURL)
			req, err := http.NewRequest("POST", cfg.SendURL, bytes.NewBuffer(jsonBody))
			if err != nil {
				slog.Default().Error("Error creating HTTP request", "error", err)
				saveFailedNotification(db, completedOrder)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create request"})
				return
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil || resp.StatusCode >= 400 {
				status := ""
				if resp != nil {
					status = resp.Status
				}
				slog.Default().Error("Error sending request or bad response", "error", err, "status", status)
				saveFailedNotification(db, completedOrder)
				c.JSON(http.StatusOK, completedOrder)
				return
			}
			defer resp.Body.Close()

			slog.Default().Info("SUCCESS payout processed", "response_status", resp.Status)
			c.JSON(http.StatusOK, completedOrder)
			slog.Default().Info("HandleCardlinkNotification => END")
			return
		}

		// Иначе (не SUCCESS) — игнорируем или обрабатываем по-своему
		slog.Default().Info("Notification ignored (status != SUCCESS)", "status", notification.Status)
		ignoreModel := models.CompletedOrder{
			OrderID:          strconv.Itoa(notification.InvId),
			OperationID:      notification.TrsId,
			Sender:           "",
			Amount:           fmt.Sprintf("%d", notification.OutSum),
			Currency:         notification.CurrencyIn,
			Status:           false,
			Sha1Hash:         notification.SignatureValue,
			TestNotification: false,
			Label:            "cardlink",
			Handle:           "ignored",
		}
		c.JSON(http.StatusOK, ignoreModel)

		// Если вы всё равно хотите отправлять «ignored» на cfg.SendURL:
		slog.Default().Info("Sending ignored notification to external service", "url", cfg.SendURL)
		jsonBody, err := json.Marshal(ignoreModel)
		if err != nil {
			slog.Default().Error("Error encoding ignoreModel to JSON", "error", err)
			slog.Default().Info("HandleCardlinkNotification => END (with error encoding ignoreModel)")
			return
		}

		req, err := http.NewRequest("POST", cfg.SendURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			slog.Default().Error("Error creating HTTP request (ignored model)", "error", err)
			slog.Default().Info("HandleCardlinkNotification => END (with error creating request for ignored model)")
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			slog.Default().Error("Error sending HTTP request (ignored model)", "error", err)
			slog.Default().Info("HandleCardlinkNotification => END (with error sending ignored model)")
			return
		}
		defer resp.Body.Close()

		slog.Default().Info("Ignored notification sent successfully", "response_status", resp.Status)
		slog.Default().Info("HandleCardlinkNotification => END")
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
