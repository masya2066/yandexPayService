package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"umani-service/app/internal/auropay"
	"umani-service/app/internal/config"
	"umani-service/app/internal/models"
	"umani-service/app/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CreateOrderAuropay создаёт инвойс Aurapay и возвращает payment_data.url как payment_link.
func CreateOrderAuropay(_ config.Config, client *auropay.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		readBodyLogRestore(c, "CreateOrderAuropay")
		cfg := config.GetConfig()
		if strings.TrimSpace(cfg.AuropayNotificationURL) == "" {
			c.JSON(http.StatusPreconditionFailed, gin.H{
				"error": "auropay_notification_url должен быть в config или в env AUROPAY_NOTIFICATION_URL (публичный URL POST /auropay/order/notification)",
			})
			return
		}

		var req models.CreatePaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input: " + err.Error()})
			return
		}

		amount, err := strconv.ParseFloat(strings.ReplaceAll(strings.TrimSpace(req.Amount), ",", "."), 64)
		if err != nil || amount < 0.01 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be a number >= 0.01 (RUB)"})
			return
		}

		orderID := strings.TrimSpace(req.OrderID)
		if orderID == "" {
			orderID = utils.GenerateUUID()
		}

		payload := map[string]any{
			"amount":       amount,
			"order_id":     orderID,
			"success_url":  strings.TrimSpace(cfg.SuccessURL),
			"fail_url":     strings.TrimSpace(cfg.FailURL),
			"callback_url": strings.TrimSpace(cfg.AuropayNotificationURL),
			"comment":      paymentDescription(req.Description),
		}
		pm := strings.ToLower(strings.TrimSpace(req.PaymentMethod))
		if pm == "card" || pm == "sbp" {
			payload["service"] = pm
		}

		body, err := json.Marshal(payload)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		log.Printf("[CreateOrderAuropay] исходящее тело в Aurapay (JSON): %s", string(body))

		respBody, code, err := client.CreateInvoice(body)
		if err != nil {
			slog.Default().Error("Auropay CreateInvoice", "err", err, "code", code, "body", string(respBody))
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "upstream": string(respBody)})
			return
		}
		if code < 200 || code >= 300 {
			c.JSON(code, json.RawMessage(respBody))
			return
		}

		var inv models.AuropayInvoiceCreateResponse
		if err := json.Unmarshal(respBody, &inv); err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": "auropay response: " + err.Error(), "raw": string(respBody)})
			return
		}
		payURL := strings.TrimSpace(inv.PaymentData.URL)
		if payURL == "" {
			c.JSON(http.StatusBadGateway, gin.H{
				"error":  "auropay response missing payment_data.url",
				"raw":    string(respBody),
				"parsed": inv,
			})
			return
		}

		out := models.CreatePaymentResponse{
			OrderID:     orderID,
			InvoiceID:   strings.TrimSpace(inv.ID),
			PaymentLink: payURL,
			Status:      strings.TrimSpace(inv.Status),
		}
		if ob, err := json.Marshal(out); err == nil {
			log.Printf("[CreateOrderAuropay] исходящий ответ клиенту: %s", string(ob))
		}
		c.JSON(http.StatusOK, out)
	}
}

func decodeJSONObjectUseNumber(body []byte) (map[string]any, error) {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	return m, nil
}

// HandleAuropayNotification — POST webhook инвойса (или выплаты: подпись проверяем, в send_url не шлём).
func HandleAuropayNotification(cfg config.Config, database *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		body, err := c.GetRawData()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "read body"})
			return
		}
		slog.Default().Info("Auropay notification входящее тело", "raw", string(body))

		params, err := decodeJSONObjectUseNumber(body)
		if err != nil {
			slog.Default().Error("Auropay notification: JSON", "err", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}

		sig := strings.TrimSpace(c.GetHeader("X-SIGNATURE"))
		cfgNow := config.GetConfig()
		secret := strings.TrimSpace(cfgNow.AuropayWebhookSecret)
		if !auropay.VerifyWebhookSignature(secret, params, sig) {
			slog.Default().Error("Auropay notification: invalid X-SIGNATURE")
			c.Status(http.StatusOK)
			return
		}

		if auropay.IsPayoutWebhook(params) {
			slog.Default().Info("Auropay notification: payout webhook, пропуск бизнес-логики оплаты")
			c.Status(http.StatusOK)
			return
		}

		status, _ := params["status"].(string)
		status = strings.TrimSpace(strings.ToUpper(status))
		orderID, _ := params["order_id"].(string)
		orderID = strings.TrimSpace(orderID)
		invID, _ := params["id"].(string)
		invID = strings.TrimSpace(invID)
		if orderID == "" {
			orderID = invID
		}

		amountStr := auropayAmountString(params["amount"])

		if status == "PAID" {
			completed := models.CompletedOrder{
				OrderID:          orderID,
				OperationID:      invID,
				Sender:           "Aurapay",
				Amount:           amountStr,
				Currency:         "RUB",
				Status:           true,
				Sha1Hash:         sig,
				TestNotification: false,
				Label:            orderID,
				Handle:           "completed",
			}
			if err := postCompletedToSendURL(cfg, database, completed); err != nil {
				slog.Default().Error("Auropay notification: post send_url failed", "send_url", cfg.SendURL, "err", err)
			} else {
				slog.Default().Info("Auropay notification: posted CompletedOrder to send_url", "send_url", cfg.SendURL)
			}
			c.JSON(http.StatusOK, completed)
			return
		}

		ignore := models.CompletedOrder{
			OrderID:          orderID,
			OperationID:      invID,
			Sender:           "Aurapay",
			Amount:           amountStr,
			Currency:         "RUB",
			Status:           false,
			Sha1Hash:         sig,
			TestNotification: false,
			Label:            orderID,
			Handle:           "ignored",
		}
		c.JSON(http.StatusOK, ignore)
		_ = postIgnoreToSendURL(cfg, ignore)
	}
}

func auropayAmountString(v any) string {
	switch t := v.(type) {
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return t.String()
	case float64:
		return formatMoney(t)
	default:
		return fmt.Sprint(t)
	}
}

// AuropayInvoiceStatus — прокси POST /invoice/status по :id (UUID инвойса или order_id).
func AuropayInvoiceStatus(_ config.Config, client *auropay.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := strings.TrimSpace(c.Param("id"))
		if id == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing id"})
			return
		}

		tryBodies := [][]byte{}
		if _, err := uuid.Parse(id); err == nil {
			b, _ := json.Marshal(map[string]string{"id": id})
			tryBodies = append(tryBodies, b)
		}
		b2, _ := json.Marshal(map[string]string{"order_id": id})
		tryBodies = append(tryBodies, b2)

		var lastBody []byte
		var lastCode int
		var lastErr error
		for i, tb := range tryBodies {
			if i > 0 && string(tb) == string(tryBodies[i-1]) {
				continue
			}
			lastBody, lastCode, lastErr = client.InvoiceStatus(tb)
			if lastErr != nil {
				c.JSON(http.StatusBadGateway, gin.H{"error": lastErr.Error(), "upstream": string(lastBody)})
				return
			}
			if lastCode != http.StatusNotFound {
				c.Data(lastCode, "application/json", lastBody)
				return
			}
		}
		c.Data(lastCode, "application/json", lastBody)
	}
}
