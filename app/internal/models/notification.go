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
