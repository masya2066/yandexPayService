package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"strings"
	"umani-service/app/internal/config"
	"umani-service/app/internal/models"
	"umani-service/app/internal/utils"

	"github.com/gin-gonic/gin"
)

func CreateOrder(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreatePaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
			return
		}

		orderID := req.OrderID
		if orderID == "" {
			orderID = utils.GenerateUUID()
		}

		yoomoneyBaseURL := "https://yoomoney.ru/quickpay/confirm.xml"
		paymentLink := fmt.Sprintf("%s?receiver=%s&quickpay-form=shop&targets=%s&sum=%s&successURL=%s&failURL=%s&label=%s",
			yoomoneyBaseURL, cfg.Receiver, req.Description, req.Amount, cfg.SuccessURL, cfg.FailURL, orderID)

		c.JSON(http.StatusOK, models.CreatePaymentResponse{
			OrderID:     orderID,
			PaymentLink: paymentLink,
		})
	}
}

func CreateOrderCardLink(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Println("[CreateOrderCardLink] => START")

		// 1. Единая модель CreatePaymentRequest (currency обязателен для CardLink)
		var req models.CreatePaymentRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("[CreateOrderCardLink] Error binding order: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
			return
		}
		if strings.TrimSpace(req.Currency) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "currency is required for cardlink"})
			return
		}
		shopID := cfg.ShopIDCardLink

		log.Printf("[CreateOrderCardLink] Order bound successfully: %+v\n", req)

		// 2. Готовим multipart/form-data для CardLink
		var bodyBuffer bytes.Buffer
		writer := multipart.NewWriter(&bodyBuffer)

		log.Println("[CreateOrderCardLink] Writing multipart fields...")
		if err := writer.WriteField("amount", req.Amount); err != nil {
			log.Printf("[CreateOrderCardLink] Failed to write field 'amount': %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write field 'amount': " + err.Error()})
			return
		}
		if err := writer.WriteField("shop_id", shopID); err != nil {
			log.Printf("[CreateOrderCardLink] Failed to write field 'shop_id': %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write field 'shop_id': " + err.Error()})
			return
		}
		if err := writer.WriteField("currency_in", req.Currency); err != nil {
			log.Printf("[CreateOrderCardLink] Failed to write field 'currency_id': %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write field 'currency_id': " + err.Error()})
			return
		}

		if req.PaymentMethod != "" {
			if err := writer.WriteField("payment_method", req.PaymentMethod); err != nil {
				log.Printf("[CreateOrderCardLink] Failed to write field 'payment_methods': %v\n", err)
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write field 'payment_methods': " + err.Error()})
				return
			}
		}

		log.Println("[CreateOrderCardLink] All fields written to multipart form.")

		// Закрываем writer, чтобы финализировать формирование multipart
		if err := writer.Close(); err != nil {
			log.Printf("[CreateOrderCardLink] Failed to close writer: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to close writer: " + err.Error()})
			return
		}
		log.Println("[CreateOrderCardLink] Multipart writer closed successfully.")

		// 3. Формируем запрос к CardLink
		log.Println("[CreateOrderCardLink] Creating request to CardLink...")
		httpReq, err := http.NewRequest("POST", "https://cardlink.link/api/v1/bill/create", &bodyBuffer)
		if err != nil {
			log.Printf("[CreateOrderCardLink] Failed to create request: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create request: " + err.Error()})
			return
		}
		log.Println("[CreateOrderCardLink] Request created.")

		// Устанавливаем заголовок Authorization и Content-Type для multipart
		httpReq.Header.Set("Authorization", "Bearer "+cfg.AuthTokenCardLink)
		httpReq.Header.Set("Content-Type", writer.FormDataContentType())
		log.Println("[CreateOrderCardLink] Headers set (Authorization, Content-Type).")

		// 4. Отправляем запрос
		log.Println("[CreateOrderCardLink] Sending request to CardLink...")
		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			log.Printf("[CreateOrderCardLink] Failed to send request: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to send request: " + err.Error()})
			return
		}
		defer resp.Body.Close()
		log.Println("[CreateOrderCardLink] Request sent successfully, reading response...")

		// 5. Читаем ответ от CardLink
		responseBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("[CreateOrderCardLink] Failed to read response body: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response body: " + err.Error()})
			return
		}
		log.Printf("[CreateOrderCardLink] Raw response body: %s\n", string(responseBody))

		// Если CardLink возвращает JSON, разбираем в структуру
		var cardLinkResp models.OrderCardLinkResponse
		if err := json.Unmarshal(responseBody, &cardLinkResp); err != nil {
			log.Printf("[CreateOrderCardLink] Failed to unmarshal response: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to unmarshal cardlink response: " + err.Error()})
			return
		}
		log.Printf("[CreateOrderCardLink] Parsed CardLink response: %+v\n", cardLinkResp)

		c.JSON(http.StatusOK, models.CreatePaymentResponse{
			OrderID:     cardLinkResp.BillID,
			PaymentLink: cardLinkResp.LinkPageURL,
		})
		log.Println("[CreateOrderCardLink] => END")
	}
}
