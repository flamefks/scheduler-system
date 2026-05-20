package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter(h *ApiHandler) http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/jobs", h.CreateJob)
		r.Get("/jobs/{id}", h.GetJob)
		r.Patch("/jobs/{id}", h.UpdateJob)
		r.Post("/jobs/{id}/activate", h.ActivateJob)
		r.Post("/jobs/{id}/deactivate", h.DeactivateJob)

		r.Delete("/jobs/{id}", h.DeleteJob)

	})
	return r
}
