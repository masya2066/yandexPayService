package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"umani-service/app/internal/b2pay"
	"umani-service/app/internal/config"
	"umani-service/app/internal/models"
	"umani-service/app/internal/utils"

	"github.com/gin-gonic/gin"
)

// CreateOrderB2Pay creates a B2Pay invoice and returns the redirect URL (metadata.auth_url).
func CreateOrderB2Pay(_ config.Config, client *b2pay.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := config.GetConfig()
		if strings.TrimSpace(cfg.B2PayNotificationURL) == "" {
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"error": "b2pay_notification_url must be set in config, or set env B2PAY_NOTIFICATION_URL (public URL for POST /b2pay/order/notification). Check CONFIG_PATH in .env if the wrong file is loaded",
			})
			return
		}

		var req models.CreatePaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
			return
		}
		if strings.TrimSpace(req.Currency) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "currency is required for b2pay"})
			return
		}

		amount, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(req.Amount), ",", "."), 64)
		if err != nil || amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be a positive number"})
			return
		}

		orderID := strings.TrimSpace(req.OrderID)
		if orderID == "" {
			orderID = utils.GenerateUUID()
		}
		customerID := strings.TrimSpace(req.CustomerID)
		if customerID == "" {
			customerID = orderID
		}
		customerEmail := strings.TrimSpace(req.Email)

		testMode := cfg.B2PayTestMode
		if req.TestMode != nil {
			testMode = *req.TestMode
		}

		returnURL := strings.TrimSpace(cfg.B2PayReturnURL)
		if returnURL == "" {
			returnURL = cfg.SuccessURL
		}

		payload := struct {
			CustomerID          string  `json:"customer_id"`
			Amount              float64 `json:"amount"`
			Currency            string  `json:"currency"`
			Description         string  `json:"description"`
			IsReturningCustomer *bool   `json:"is_returning_customer,omitempty"`
			Metadata            meta    `json:"metadata"`
		}{
			CustomerID:  customerID,
			Amount:      amount,
			Currency:    req.Currency,
			Description: paymentDescription(req.Description),
			Metadata: meta{
				TestMode:        testMode,
				CustomerEmail:   customerEmail,
				TrackingID:      orderID,
				ReturnURL:       returnURL,
				NotificationURL: cfg.B2PayNotificationURL,
			},
		}
		if req.IsReturningCustomer != nil {
			payload.IsReturningCustomer = req.IsReturningCustomer
		}

		body, err := json.Marshal(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		respBody, code, err := client.CreateInvoice(body)
		if err != nil {
			slog.Default().Error("B2Pay CreateInvoice", "err", err, "code", code, "body", string(respBody))
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "upstream": string(respBody)})
			return
		}
		if code < 200 || code >= 300 {
			c.JSON(code, json.RawMessage(respBody))
			return
		}

		var inv models.B2PayInvoice
		if err := json.Unmarshal(respBody, &inv); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "b2pay response: " + err.Error(), "raw": string(respBody)})
			return
		}
		authURL := inv.Metadata.AuthURL
		if authURL == "" {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":  "b2pay response missing metadata.auth_url",
				"raw":    string(respBody),
				"parsed": inv,
			})
			return
		}

		c.JSON(http.StatusOK, models.CreatePaymentResponse{
			OrderID:     orderID,
			InvoiceID:   inv.ID,
			PaymentLink: authURL,
			Status:      inv.Status,
		})
	}
}

type meta struct {
	TestMode        bool   `json:"test_mode"`
	CustomerEmail   string `json:"customer_email,omitempty"`
	TrackingID      string `json:"tracking_id"`
	ReturnURL       string `json:"return_url,omitempty"`
	NotificationURL string `json:"notification_url"`
}

// HandleB2PayNotification receives B2Pay POST callbacks on metadata.notification_url.
// Verifies X-Callback-Signature (HMAC-SHA256 of raw body with API key). Responds 200 on success of handler.
func HandleB2PayNotification(cfg config.Config, database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != "" && c.ContentType() != "application/json" {
			slog.Default().Error("B2Pay notification: bad Content-Type", "got", c.ContentType())
		}

		body, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
			return
		}

		sig := c.GetHeader("X-Callback-Signature")
		cfgNow := config.GetConfig()
		if !b2pay.VerifyCallbackSignature(cfgNow.B2PayAPIKey, body, sig) {
			slog.Default().Error("B2Pay notification: invalid X-Callback-Signature")
			c.Status(http.StatusOK)
			return
		}

		var inv models.B2PayInvoice
		if err := json.Unmarshal(body, &inv); err != nil {
			slog.Default().Error("B2Pay notification: JSON", "err", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		slog.Default().Info("B2Pay notification OK",
			"invoice_id", inv.ID, "status", inv.Status, "tracking_id", inv.Metadata.TrackingID)

		orderID := inv.Metadata.TrackingID
		if orderID == "" {
			orderID = inv.ID
		}
		amountStr := formatMoney(inv.Amount)
		if inv.Status == "success" {
			completed := models.CompletedOrder{
				OrderID:          orderID,
				OperationID:      inv.ID,
				Sender:           "B2Pay",
				Amount:           amountStr,
				Currency:         inv.Currency,
				Status:           true,
				Sha1Hash:         sig,
				TestNotification: inv.Metadata.TestMode,
				Label:            orderID,
				Handle:           "completed",
			}
			if err := postCompletedToSendURL(cfg, database, completed); err != nil {
				slog.Default().Error("B2Pay notification: post send_url failed", "send_url", cfg.SendURL, "err", err)
			} else {
				slog.Default().Info("B2Pay notification: posted CompletedOrder to send_url", "send_url", cfg.SendURL)
			}
			c.JSON(http.StatusOK, completed)
			return
		}

		ignore := models.CompletedOrder{
			OrderID:          orderID,
			OperationID:      inv.ID,
			Sender:           "B2Pay",
			Amount:           amountStr,
			Currency:         inv.Currency,
			Status:           false,
			Sha1Hash:         sig,
			TestNotification: inv.Metadata.TestMode,
			Label:            orderID,
			Handle:           "ignored",
		}
		c.JSON(http.StatusOK, ignore)
		_ = postIgnoreToSendURL(cfg, ignore)
	}
}

func postCompletedToSendURL(cfg config.Config, database *sql.DB, completed models.CompletedOrder) error {
	jsonBody, err := json.Marshal(completed)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, cfg.SendURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		saveFailedNotification(database, completed)
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Default().Error("send_url request failed", "send_url", cfg.SendURL, "err", err)
		saveFailedNotification(database, completed)
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		slog.Default().Error("send_url bad status", "send_url", cfg.SendURL, "status", resp.Status)
		saveFailedNotification(database, completed)
		return fmt.Errorf("send_url HTTP %s", resp.Status)
	}
	return nil
}

func postIgnoreToSendURL(cfg config.Config, ignore models.CompletedOrder) error {
	jsonBody, err := json.Marshal(ignore)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, cfg.SendURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err == nil && resp != nil {
		_ = resp.Body.Close()
	}
	return err
}

func formatMoney(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

// B2PayTransactionStatus proxies GET /v1/transactions/{id}/status (rate limit: 10/min on B2Pay side).
func B2PayTransactionStatus(_ config.Config, client *b2pay.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.Param("id"))
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
			return
		}
		body, code, err := client.TransactionStatus(id)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "upstream": string(body)})
			return
		}
		c.Data(code, "application/json", body)
	}
}
