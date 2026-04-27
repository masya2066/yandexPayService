package models

// B2PayInvoice is the B2Pay invoice / callback document (subset used by this service).
type B2PayInvoice struct {
	ID            string  `json:"id"`
	CustomerID    string  `json:"customer_id"`
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	Status        string  `json:"status"`
	Description   string  `json:"description"`
	PaymentMethod string  `json:"payment_method"`
	ProviderName  string  `json:"provider_name"`
	Metadata      struct {
		AuthURL       string `json:"auth_url"`
		TrackingID    string `json:"tracking_id"`
		TestMode      bool   `json:"test_mode"`
		CustomerEmail string `json:"customer_email"`
	} `json:"metadata"`
}
