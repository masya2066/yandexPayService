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

type OrderCardLink struct {
	Amount        string `json:"amount"`
	ShopID        string `json:"shop_id"`
	CurrencyIn    string `json:"currency_in"`
	PaymentMethod string `json:"payment_methods"`
}

type OrderCardLinkResponse struct {
	Success     string `json:"success"`
	LinkURL     string `json:"link_url"`
	LinkPageURL string `json:"link_page_url"`
	BillID      string `json:"bill_id"`
}
