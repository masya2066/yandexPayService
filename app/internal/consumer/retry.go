package consumer

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
	"umani-service/app/internal/config"
	"umani-service/app/internal/db"
)

func RetryFailedNotifications(cfg config.Config, database *sql.DB) {
	for {
		unsentNotifications, err := db.GetUnsentNotifications(database)
		if err != nil {
			slog.Default().Error("Failed to retrieve unsent notifications:", err)
			time.Sleep(1 * time.Minute)
			continue
		}

		for _, notification := range unsentNotifications {
			jsonBody, err := json.Marshal(notification)
			if err != nil {
				slog.Default().Error("Error encoding notification to JSON:", err)
				continue
			}

			req, err := http.NewRequest("POST", cfg.SendURL, bytes.NewBuffer(jsonBody))
			if err != nil {
				slog.Default().Error("Error creating HTTP request:", err)
				continue
			}
			req.Header.Set("Content-Type", "application/json")

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil || resp.StatusCode >= 400 {
				slog.Default().Error("Error sending HTTP request or bad response:", err, resp.Status)
				continue
			}
			defer resp.Body.Close()

			deleteQuery := "DELETE FROM failed_notifications WHERE order_id = ?;"
			_, err = database.Exec(deleteQuery, notification.OrderID)
			if err != nil {
				slog.Default().Error("Failed to delete notification:", err)
			} else {
				slog.Default().Info("Successfully retried and deleted notification:", notification.OrderID)
			}
		}

		time.Sleep(1 * time.Minute)
	}
}
