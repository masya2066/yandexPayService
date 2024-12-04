package handlers

import (
	"fmt"
	"net/http"
	"umani-service/app/internal/config"
	"umani-service/app/internal/models"
	"umani-service/app/internal/utils"

	"github.com/gin-gonic/gin"
)

func CreateOrder(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var order models.Order
		if err := c.Bind(&order); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
			return
		}

		// Генерация уникального ID заказа
		order.ID = utils.GenerateUUID()

		// Формирование ссылки на оплату
		yoomoneyBaseURL := "https://yoomoney.ru/quickpay/confirm.xml"
		order.PaymentLink = fmt.Sprintf("%s?receiver=%s&quickpay-form=shop&targets=%s&sum=%s&successURL=%s&failURL=%s&label=%s",
			yoomoneyBaseURL, cfg.Receiver, order.Description, order.Amount, cfg.SuccessURL, cfg.FailURL, order.ID)

		// Возвращение ссылки клиенту
		c.JSON(http.StatusOK, gin.H{
			"order_id":     order.ID,
			"payment_link": order.PaymentLink,
		})
	}
}
