package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/armorclaw/lighthouse/internal/api"
	"github.com/armorclaw/lighthouse/internal/config"
	"github.com/armorclaw/lighthouse/internal/db"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("[LIGHTHOUSE] Failed to load config: %v", err)
	}

	sqlDB, err := sql.Open("sqlite3", cfg.DatabasePath)
	if err != nil {
		log.Fatalf("[LIGHTHOUSE] Failed to open database: %v", err)
	}
	defer sqlDB.Close()

	if err := db.RunMigrations(sqlDB); err != nil {
		log.Fatalf("[LIGHTHOUSE] Failed to run migrations: %v", err)
	}

	r := api.NewRouter(cfg)
	api.SetDatabase(sqlDB)

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("[LIGHTHOUSE] Starting server on %s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[LIGHTHOUSE] Server error: %v", err)
		}
	}()

	sig := <-shutdownChan
	log.Printf("[LIGHTHOUSE] Received signal %v, initiating graceful shutdown...", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[LIGHTHOUSE] Error during server shutdown: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		log.Printf("[LIGHTHOUSE] Error closing database: %v", err)
	}

	log.Println("[LIGHTHOUSE] Server shutdown complete")
}
