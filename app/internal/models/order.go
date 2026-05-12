package models

// CreatePaymentRequest is the unified JSON body for /yandex/order/create, /cardlink/order/create, and /b2pay/order/create.
// Extra fields (currency, payment_method, customer_id, …) are ignored by methods that do not use them.
type CreatePaymentRequest struct {
	Amount              string `json:"amount" binding:"required"`
	Description         string `json:"description"` // optional — legacy CardLink clients often omit it
	Email               string `json:"email"`
	Currency            string `json:"currency"`
	OrderID             string `json:"order_id"`
	PaymentMethod       string `json:"payment_method"`
	CustomerID          string `json:"customer_id"`
	IsReturningCustomer *bool  `json:"is_returning_customer"`
	TestMode            *bool  `json:"test_mode"`
}

// CreatePaymentResponse is the unified successful JSON for all payment create endpoints.
type CreatePaymentResponse struct {
	OrderID     string `json:"order_id"`
	PaymentLink string `json:"payment_link"`
	InvoiceID   string `json:"invoice_id,omitempty"`
	Status      string `json:"status,omitempty"`
}

type CompletedOrder struct {
	OrderID          string `json:"order_id"`
	OperationID      string `json:"operation_id"`
	Sender           string `json:"sender"`
	Amount           string `json:"amount"`
	Currency         string `json:"currency"`
	Status           bool   `json:"status"`
	Sha1Hash         string `json:"sha1_hash"`
	TestNotification bool   `json:"test_notification"`
	Label            string `json:"label"`
	Handle           string `json:"handle"`
}

type OrderCardLinkResponse struct {
	Success     string `json:"success"`
	LinkURL     string `json:"link_url"`
	LinkPageURL string `json:"link_page_url"`
	BillID      string `json:"bill_id"`
}
