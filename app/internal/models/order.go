package models

type Order struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Amount      string `json:"amount"`
	Description string `json:"description"`
	PaymentLink string `json:"payment_link"`
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
