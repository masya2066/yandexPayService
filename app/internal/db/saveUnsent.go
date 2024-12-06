package db

import (
	"database/sql"
	"fmt"
	"umani-service/app/internal/models"
)

func SaveUnsentNotification(db *sql.DB, order models.CompletedOrder) error {
	insertQuery := `
	INSERT INTO failed_notifications (
		order_id,
		operation_id,
		sender,
		amount,
		currency,
		status,
		sha1_hash,
		test_notification,
		label,
		handle
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	_, err := db.Exec(insertQuery,
		order.OrderID,
		order.OperationID,
		order.Sender,
		order.Amount,
		order.Currency,
		order.Status,
		order.Sha1Hash,
		order.TestNotification,
		order.Label,
		order.Handle,
	)
	if err != nil {
		return fmt.Errorf("failed to save completed order: %w", err)
	}

	return nil
}

func GetUnsentNotifications(db *sql.DB) ([]models.CompletedOrder, error) {
	selectQuery := `
	SELECT
		order_id,
		operation_id,
		sender,
		amount,
		currency,
		status,
		sha1_hash,
		test_notification,
		label,
		handle
	FROM failed_notifications;`

	rows, err := db.Query(selectQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get unsent notifications: %w", err)
	}
	defer rows.Close()

	var unsentNotifications []models.CompletedOrder
	for rows.Next() {
		var notification models.CompletedOrder
		if err := rows.Scan(
			&notification.OrderID,
			&notification.OperationID,
			&notification.Sender,
			&notification.Amount,
			&notification.Currency,
			&notification.Status,
			&notification.Sha1Hash,
			&notification.TestNotification,
			&notification.Label,
			&notification.Handle,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		unsentNotifications = append(unsentNotifications, notification)
	}

	return unsentNotifications, nil
}
