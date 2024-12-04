package models

type Order struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	Amount      string `json:"amount"`
	Description string `json:"description"`
	PaymentLink string `json:"payment_link"`
}
