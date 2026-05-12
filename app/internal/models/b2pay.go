package models

// OrderB2PayInput — тело POST /b2pay/order/create (объявлено здесь, используется в handlers/b2pay.go).
type OrderB2PayInput struct {
	CustomerID          string `json:"customer_id"`
	Amount              string `json:"amount" binding:"required"`
	Currency            string `json:"currency" binding:"required"`
	Description         string `json:"description" binding:"required"`
	TestMode            bool   `json:"test_mode"`
	IsReturningCustomer *bool  `json:"is_returning_customer"`
	ReturnURL           string `json:"return_url"`
	CustomerEmail       string `json:"customer_email"`
}

// B2PayInvoiceResponse — ответ POST /v1/invoices и тело callback (частично).
type B2PayInvoiceResponse struct {
	ID          string                 `json:"id"`
	CustomerID  string                 `json:"customer_id"`
	Amount      float64                `json:"amount"`
	Currency    string                 `json:"currency"`
	Status      string                 `json:"status"`
	Description string                 `json:"description"`
	Metadata    map[string]any         `json:"metadata"`
	CreatedAt   string                 `json:"created_at"`
	PaymentMethod string               `json:"payment_method"`
	ProviderName  string               `json:"provider_name"`
}
