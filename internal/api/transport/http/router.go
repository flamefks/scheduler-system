package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(h http.Handler, access_secret string) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	return r
}
