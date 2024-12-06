package db

import (
	"database/sql"
	"fmt"
	_ "modernc.org/sqlite" // SQLite драйвер
	"os"
)

// Initialize SQLite database
func InitDatabase(dbPath string) (*sql.DB, error) {
	// Check if the directory for the database exists
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		file, err := os.Create(dbPath) // Create the file explicitly
		if err != nil {
			return nil, fmt.Errorf("failed to create database file: %w", err)
		}
		file.Close()
	}

	// Open the SQLite database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to SQLite: %w", err)
	}

	// Create table for failed notifications if not exists
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS failed_notifications (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		order_id TEXT NOT NULL,
		operation_id TEXT,
		sender TEXT,
		amount TEXT NOT NULL,
		currency TEXT,
		status BOOLEAN NOT NULL,
		sha1_hash TEXT,
		test_notification BOOLEAN DEFAULT FALSE,
		label TEXT,
		handle TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err = db.Exec(createTableQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return db, nil
}
