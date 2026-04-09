package http

import (
	"github.com/go-chi/chi/v5"
	_ "github.com/mattn/go-sqlite3"
)

func NewRouter() *chi.Mux {
	return chi.NewRouter()
}
