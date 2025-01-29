package models

type Notification struct {
	NotificationType string `form:"notification_type" binding:"required"`
	OperationId      string `form:"operation_id" binding:"required"`
	Amount           string `form:"amount" binding:"required"`
	Currency         string `form:"currency" binding:"required"`
	DateTime         string `form:"datetime" binding:"required"`
	Sender           string `form:"sender"`
	Codepro          string `form:"codepro" binding:"required"`
	Label            string `form:"label"`
	Sha1Hash         string `form:"sha1_hash" binding:"required"`
	Unaccepted       bool   `form:"unaccepted"`
	WithdrawAmount   string `form:"withdraw_amount"`
	TestNotification bool   `form:"test_notification"`
	OperationLabel   string `form:"operation_label"`
}

type CardLinkSuccess struct {
	InvId          int    `json:"InvId"`
	OutSum         int    `json:"OutSum"`
	Custom         string `json:"custom"`
	CurrencyIn     string `json:"CurrencyIn"`
	SignatureValue string `json:"SignatureValue"`
}

type CardLinkFail struct {
	InvId          int    `json:"InvId"`
	OutSum         int    `json:"OutSum"`
	Custom         string `json:"custom"`
	CurrencyIn     string `json:"CurrencyIn"`
	SignatureValue string `json:"SignatureValue"`
}

type CardLinkNotification struct {
	InvId          int    `form:"InvId"`
	OutSum         string `form:"OutSum"`
	CurrencyIn     string `form:"CurrencyIn"`
	Commission     string `form:"Commission"`
	TrsId          string `form:"TrsId"`
	Status         string `form:"Status"`
	Custom         string `form:"custom"`
	SignatureValue string `form:"SignatureValue"`
}
