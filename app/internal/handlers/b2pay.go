package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
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

// CreateOrderB2Pay создаёт счёт B2Pay и возвращает ссылку на оплату (metadata.auth_url). Регистрируется в main.go.
func CreateOrderB2Pay(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var in models.OrderB2PayInput
		if err := c.ShouldBindJSON(&in); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "некорректный JSON: " + err.Error()})
			return
		}
		if strings.TrimSpace(cfg.B2PayNotificationURL) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "в config не задан b2pay_notification_url (публичный URL коллбэка)"})
			return
		}
		if cfg.B2PayUserID == "" || cfg.B2PayEmail == "" || cfg.B2PayAPIKey == "" {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "B2Pay не настроен (user_id, email, api_key)"})
			return
		}

		amount, err := parseAmountB2Pay(in.Amount)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		orderID := utils.GenerateUUID()
		customerID := strings.TrimSpace(in.CustomerID)
		if customerID == "" {
			customerID = "cust-" + orderID
		}

		meta := map[string]any{
			"test_mode":        in.TestMode,
			"tracking_id":      orderID,
			"notification_url": strings.TrimSpace(cfg.B2PayNotificationURL),
		}
		if in.CustomerEmail != "" {
			meta["customer_email"] = in.CustomerEmail
		}
		ret := strings.TrimSpace(in.ReturnURL)
		if ret == "" {
			ret = strings.TrimSpace(cfg.B2PayReturnURL)
		}
		if ret != "" {
			meta["return_url"] = ret
		}

		invReq := b2pay.InvoiceCreateRequest{
			CustomerID:          customerID,
			Amount:              amount,
			Currency:            strings.TrimSpace(in.Currency),
			Description:         strings.TrimSpace(in.Description),
			IsReturningCustomer: in.IsReturningCustomer,
			Metadata:            meta,
		}

		resp, err := b2pay.CreateInvoice(cfg, invReq)
		if err != nil {
			slog.Default().Error("B2Pay CreateInvoice", "error", err)
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}

		authURL, _ := resp.Metadata["auth_url"].(string)
		if authURL == "" {
			c.JSON(http.StatusBadGateway, gin.H{"error": "B2Pay не вернул metadata.auth_url"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"order_id":     orderID,
			"invoice_id":   resp.ID,
			"payment_link": authURL,
			"status":       resp.Status,
		})
	}
}

// HandleB2PayNotification принимает JSON-коллбэк B2Pay. Регистрируется в main.go.
func HandleB2PayNotification(cfg config.Config, db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "не удалось прочитать тело"})
			return
		}

		sig := c.GetHeader("X-Callback-Signature")
		if !b2pay.VerifyCallbackSignature(raw, sig, cfg.B2PayAPIKey) {
			slog.Default().Error("B2Pay callback: неверная подпись")
			c.JSON(http.StatusOK, gin.H{"ok": false, "reason": "invalid_signature"})
			return
		}

		var inv models.B2PayInvoiceResponse
		if err := json.Unmarshal(raw, &inv); err != nil {
			slog.Default().Error("B2Pay callback: JSON", "error", err)
			c.JSON(http.StatusOK, gin.H{"ok": false, "reason": "bad_json"})
			return
		}

		tracking, _ := inv.Metadata["tracking_id"].(string)
		if tracking == "" {
			tracking = inv.CustomerID
		}
		testMode := metadataBool(inv.Metadata["test_mode"])

		amountStr := formatAmountB2Pay(inv.Amount)
		sigHex := callbackSigHex(sig)

		if inv.Status == "success" {
			completed := models.CompletedOrder{
				OrderID:          tracking,
				OperationID:      inv.ID,
				Sender:           "B2Pay",
				Amount:           amountStr,
				Currency:         inv.Currency,
				Status:           true,
				Sha1Hash:         sigHex,
				TestNotification: testMode,
				Label:            tracking,
				Handle:           "completed",
			}

			jsonBody, err := json.Marshal(completed)
			if err != nil {
				slog.Default().Error("B2Pay callback: marshal", "error", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
				return
			}

			req, err := http.NewRequest(http.MethodPost, cfg.SendURL, bytes.NewBuffer(jsonBody))
			if err != nil {
				slog.Default().Error("B2Pay callback: NewRequest", "error", err)
				saveFailedNotification(db, completed)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "request"})
				return
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil || resp == nil || resp.StatusCode >= 400 {
				status := ""
				if resp != nil {
					status = resp.Status
				}
				slog.Default().Error("B2Pay callback: SendURL", "error", err, "status", status)
				saveFailedNotification(db, completed)
				if resp != nil {
					resp.Body.Close()
				}
				c.JSON(http.StatusOK, completed)
				return
			}
			defer resp.Body.Close()

			slog.Default().Info("B2Pay callback: успешно отправлено в SendURL", "tracking", tracking)
			c.JSON(http.StatusOK, completed)
			return
		}

		slog.Default().Info("B2Pay callback: статус не success", "status", inv.Status)
		ignore := models.CompletedOrder{
			OrderID:          tracking,
			OperationID:      inv.ID,
			Sender:           "B2Pay",
			Amount:           amountStr,
			Currency:         inv.Currency,
			Status:           false,
			Sha1Hash:         sigHex,
			TestNotification: testMode,
			Label:            tracking,
			Handle:           "ignored",
		}
		c.JSON(http.StatusOK, ignore)

		jsonBody, err := json.Marshal(ignore)
		if err != nil {
			return
		}
		req, err := http.NewRequest(http.MethodPost, cfg.SendURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			slog.Default().Error("B2Pay callback: ignored SendURL", "error", err)
			return
		}
		defer resp.Body.Close()
		slog.Default().Info("B2Pay callback: ignored отправлен", "status", resp.Status)
	}
}

func parseAmountB2Pay(s string) (float64, error) {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		return 0, fmt.Errorf("пустая сумма")
	}
	return strconv.ParseFloat(s, 64)
}

func formatAmountB2Pay(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

func metadataBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case float64:
		return t != 0
	case string:
		b, err := strconv.ParseBool(t)
		return err == nil && b
	default:
		return false
	}
}

func callbackSigHex(sig string) string {
	sig = strings.TrimSpace(sig)
	if i := strings.Index(sig, "="); i >= 0 && i+1 < len(sig) {
		return sig[i+1:]
	}
	return sig
}
