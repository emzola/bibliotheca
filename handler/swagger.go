package handler

import "net/http"

func (h *Handler) handleSwaggerFile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./docs/swagger.json")
	}
}
