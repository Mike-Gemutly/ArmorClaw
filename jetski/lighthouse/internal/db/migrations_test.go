package db

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestMigrations(t *testing.T) {
	// Create temporary database file
	dbPath := "test_migrations.db"
	defer os.Remove(dbPath)

	// Open database connection
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Run migrations
	err = RunMigrations(db)
	if err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify charts table structure
	t.Run("charts_table_exists", func(t *testing.T) {
		var tableName string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='charts';").Scan(&tableName)
		if err != nil {
			t.Errorf("Charts table does not exist: %v", err)
		}
	})

	// Verify users table structure
	t.Run("users_table_exists", func(t *testing.T) {
		var tableName string
		err := db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='users';").Scan(&tableName)
		if err != nil {
			t.Errorf("Users table does not exist: %v", err)
		}
	})

	// Verify charts table columns
	t.Run("charts_table_columns", func(t *testing.T) {
		expectedColumns := []string{
			"id", "domain", "version", "author", "chart_data",
			"signature", "blessed", "downloads", "created_at",
		}

		rows, err := db.Query("PRAGMA table_info(charts)")
		if err != nil {
			t.Fatalf("Failed to get chart table info: %v", err)
		}
		defer rows.Close()

		foundColumns := make(map[string]bool)
		for rows.Next() {
			var cid int
			var name string
			var dataType string
			var notNull int
			var dfltValue interface{}
			var pk int

			err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
			if err != nil {
				t.Fatalf("Failed to scan column info: %v", err)
			}
			foundColumns[name] = true
		}

		for _, col := range expectedColumns {
			if !foundColumns[col] {
				t.Errorf("Expected column '%s' not found in charts table", col)
			}
		}
	})

	// Verify users table columns
	t.Run("users_table_columns", func(t *testing.T) {
		expectedColumns := []string{
			"id", "username", "api_key", "role", "created_at",
		}

		rows, err := db.Query("PRAGMA table_info(users)")
		if err != nil {
			t.Fatalf("Failed to get users table info: %v", err)
		}
		defer rows.Close()

		foundColumns := make(map[string]bool)
		for rows.Next() {
			var cid int
			var name string
			var dataType string
			var notNull int
			var dfltValue interface{}
			var pk int

			err := rows.Scan(&cid, &name, &dataType, &notNull, &dfltValue, &pk)
			if err != nil {
				t.Fatalf("Failed to scan column info: %v", err)
			}
			foundColumns[name] = true
		}

		for _, col := range expectedColumns {
			if !foundColumns[col] {
				t.Errorf("Expected column '%s' not found in users table", col)
			}
		}
	})

	// Verify UNIQUE constraint on charts (domain, version)
	t.Run("charts_unique_constraint", func(t *testing.T) {
		// Insert a chart
		_, err := db.Exec(`
			INSERT INTO charts (domain, version, author, chart_data)
			VALUES ('test.com', '1.0.0', 'testuser', '{}')
		`)
		if err != nil {
			t.Fatalf("Failed to insert test chart: %v", err)
		}

		// Try to insert duplicate (should fail due to UNIQUE constraint)
		_, err = db.Exec(`
			INSERT INTO charts (domain, version, author, chart_data)
			VALUES ('test.com', '1.0.0', 'otheruser', '{}')
		`)
		if err == nil {
			t.Error("Expected UNIQUE constraint violation for duplicate (domain, version)")
		}
	})

	// Verify indexes exist
	t.Run("indexes_exist", func(t *testing.T) {
		expectedIndexes := []string{"idx_domain", "idx_blessed"}

		rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='charts';")
		if err != nil {
			t.Fatalf("Failed to get indexes: %v", err)
		}
		defer rows.Close()

		foundIndexes := make(map[string]bool)
		for rows.Next() {
			var name string
			err := rows.Scan(&name)
			if err != nil {
				t.Fatalf("Failed to scan index name: %v", err)
			}
			foundIndexes[name] = true
		}

		for _, idx := range expectedIndexes {
			if !foundIndexes[idx] {
				t.Errorf("Expected index '%s' not found", idx)
			}
		}
	})
}
