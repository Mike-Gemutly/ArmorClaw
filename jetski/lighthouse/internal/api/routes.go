package api

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"

	"github.com/armorclaw/lighthouse/internal/config"
	"github.com/armorclaw/lighthouse/internal/db"
	"github.com/armorclaw/lighthouse/internal/models"
	"github.com/go-chi/chi/v5"
)

var sqlDB *sql.DB

func NewRouter(cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	r.Use(LoggingMiddleware)

	r.Get("/charts", ListChartsHandler)
	r.Get("/charts/{domain}/{version}", GetChartHandler)
	r.With(AuthMiddleware(cfg)).Post("/charts", PostChartHandler)
	r.Get("/charts/blessed", GetBlessedChartsHandler)

	return r
}

func SetDatabase(db *sql.DB) {
	sqlDB = db
}

func ListChartsHandler(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")

	if domain != "" {
		charts, err := db.ListChartsByDomain(sqlDB, domain)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(charts); err != nil {
			log.Printf("[LIGHTHOUSE] Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Missing domain parameter"))
}

func GetChartHandler(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	version := chi.URLParam(r, "version")

	chart, err := db.GetChartByVersion(sqlDB, domain, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if chart == nil {
		http.Error(w, "Chart not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(chart); err != nil {
		log.Printf("[LIGHTHOUSE] Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func PostChartHandler(w http.ResponseWriter, r *http.Request) {
	var chart models.Chart
	if err := json.NewDecoder(r.Body).Decode(&chart); err != nil {
		log.Printf("[LIGHTHOUSE] Error decoding chart: %v", err)
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := db.InsertChart(sqlDB, &chart); err != nil {
		log.Printf("[LIGHTHOUSE] Error inserting chart: %v", err)
		http.Error(w, "Failed to insert chart", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(chart); err != nil {
		log.Printf("[LIGHTHOUSE] Failed to encode response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func GetBlessedChartsHandler(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")

	if domain != "" {
		chart, err := db.GetLatestBlessedChart(sqlDB, domain)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if chart == nil {
			http.Error(w, "Blessed chart not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(chart); err != nil {
			log.Printf("[LIGHTHOUSE] Failed to encode response: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
			return
		}
		return
	}

	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte("Missing domain parameter"))
}
