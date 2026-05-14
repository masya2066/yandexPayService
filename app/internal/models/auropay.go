package models

// AuropayInvoiceCreateResponse — ответ POST /invoice/create (используемые поля).
type AuropayInvoiceCreateResponse struct {
	ID          string `json:"id"`
	OrderID     string `json:"order_id"`
	Status      string `json:"status"`
	PaymentData struct {
		URL string `json:"url"`
	} `json:"payment_data"`
}
