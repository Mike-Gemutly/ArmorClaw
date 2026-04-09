package db

import (
	"database/sql"
	"fmt"
	"log"
)

// RunMigrations executes database schema migrations for LIGHTHOUSE
func RunMigrations(db *sql.DB) error {
	log.Println("[LIGHTHOUSE] Starting database migrations...")

	// Create charts table
	chartsSQL := `
	CREATE TABLE IF NOT EXISTS charts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		domain TEXT NOT NULL,
		version TEXT NOT NULL,
		author TEXT NOT NULL,
		chart_data TEXT NOT NULL,
		signature TEXT,
		blessed BOOLEAN DEFAULT FALSE,
		downloads INTEGER DEFAULT 0,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(domain, version)
	);`

	_, err := db.Exec(chartsSQL)
	if err != nil {
		return fmt.Errorf("[LIGHTHOUSE] failed to create charts table: %w", err)
	}
	log.Println("[LIGHTHOUSE] Charts table created successfully")

	// Create users table
	usersSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		api_key TEXT NOT NULL,
		role TEXT DEFAULT 'community',
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	_, err = db.Exec(usersSQL)
	if err != nil {
		return fmt.Errorf("[LIGHTHOUSE] failed to create users table: %w", err)
	}
	log.Println("[LIGHTHOUSE] Users table created successfully")

	// Create indexes on charts table
	domainIndexSQL := `CREATE INDEX IF NOT EXISTS idx_domain ON charts(domain);`
	_, err = db.Exec(domainIndexSQL)
	if err != nil {
		return fmt.Errorf("[LIGHTHOUSE] failed to create domain index: %w", err)
	}
	log.Println("[LIGHTHOUSE] Domain index created successfully")

	blessedIndexSQL := `CREATE INDEX IF NOT EXISTS idx_blessed ON charts(blessed);`
	_, err = db.Exec(blessedIndexSQL)
	if err != nil {
		return fmt.Errorf("[LIGHTHOUSE] failed to create blessed index: %w", err)
	}
	log.Println("[LIGHTHOUSE] Blessed index created successfully")

	log.Println("[LIGHTHOUSE] Database migrations completed successfully")
	return nil
}
