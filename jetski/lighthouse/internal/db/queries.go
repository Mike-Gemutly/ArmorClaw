package db

import (
	"database/sql"
	"github.com/armorclaw/lighthouse/internal/models"
)

// ListChartsByDomain retrieves up to 50 charts for a domain, ordered by created_at DESC
func ListChartsByDomain(db *sql.DB, domain string) ([]models.Chart, error) {
	stmt, err := db.Prepare(`
		SELECT id, domain, version, author, chart_data, signature, blessed, downloads, created_at
		FROM charts
		WHERE domain = ?
		ORDER BY created_at DESC
		LIMIT 50
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var charts []models.Chart
	for rows.Next() {
		var chart models.Chart
		err := rows.Scan(
			&chart.ID,
			&chart.Domain,
			&chart.Version,
			&chart.Author,
			&chart.ChartData,
			&chart.Signature,
			&chart.Blessed,
			&chart.Downloads,
			&chart.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		charts = append(charts, chart)
	}

	return charts, nil
}

// GetChartByVersion retrieves a specific chart by domain and version
func GetChartByVersion(db *sql.DB, domain, version string) (*models.Chart, error) {
	stmt, err := db.Prepare(`
		SELECT id, domain, version, author, chart_data, signature, blessed, downloads, created_at
		FROM charts
		WHERE domain = ? AND version = ?
		LIMIT 1
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var chart models.Chart
	err = stmt.QueryRow(domain, version).Scan(
		&chart.ID,
		&chart.Domain,
		&chart.Version,
		&chart.Author,
		&chart.ChartData,
		&chart.Signature,
		&chart.Blessed,
		&chart.Downloads,
		&chart.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &chart, nil
}

// InsertChart inserts a new chart into the database
func InsertChart(db *sql.DB, chart *models.Chart) error {
	stmt, err := db.Prepare(`
		INSERT INTO charts (domain, version, author, chart_data, signature, blessed, downloads, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	result, err := stmt.Exec(
		chart.Domain,
		chart.Version,
		chart.Author,
		chart.ChartData,
		chart.Signature,
		chart.Blessed,
		chart.Downloads,
		chart.CreatedAt,
	)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	chart.ID = int(id)
	return nil
}

// GetLatestBlessedChart retrieves the most recent blessed chart for a domain
func GetLatestBlessedChart(db *sql.DB, domain string) (*models.Chart, error) {
	stmt, err := db.Prepare(`
		SELECT id, domain, version, author, chart_data, signature, blessed, downloads, created_at
		FROM charts
		WHERE domain = ? AND blessed = 1
		ORDER BY created_at DESC
		LIMIT 1
	`)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var chart models.Chart
	err = stmt.QueryRow(domain).Scan(
		&chart.ID,
		&chart.Domain,
		&chart.Version,
		&chart.Author,
		&chart.ChartData,
		&chart.Signature,
		&chart.Blessed,
		&chart.Downloads,
		&chart.CreatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &chart, nil
}
