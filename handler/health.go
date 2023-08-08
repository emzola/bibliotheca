package handler

import "net/http"

func (h *Handler) healthcheckHandler(w http.ResponseWriter, r *http.Request) {
	health := envelope{
		"status": "available",
		"system_info": map[string]string{
			"environment": h.config.Env,
			"version":     "1.0.0",
		},
	}
	err := h.encodeJSON(w, http.StatusOK, health, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
