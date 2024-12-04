package models

type Notification struct {
	NotificationType string `json:"notification_type"`
	OperationId      string `json:"operation_id"`
	Amount           string `json:"amount"`
	Currency         string `json:"currency"`
	DateTime         string `json:"datetime"`
	Sender           string `json:"sender"`
	Codepro          bool   `json:"codepro"`
	Label            string `json:"label"`
	Sha1Hash         string `json:"sha1_hash"`
	Unaccepted       bool   `json:"unaccepted"`
}
