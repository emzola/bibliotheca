package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/internal/jsonlog"
)

var (
	ErrInvalidMimeType = errors.New("content type is not supported")
)

func (h *Handler) logError(r *http.Request, err error) {
	var logger *jsonlog.Logger
	logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	})
}

func (h *Handler) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	env := envelope{"error": message}
	err := h.encodeJSON(w, status, env, nil)
	if err != nil {
		h.logError(r, err)
		w.WriteHeader(500)
	}
}

func (h *Handler) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.logError(r, err)
	message := "the server encountered a problem and could not process your request"
	h.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (h *Handler) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	h.errorResponse(w, r, http.StatusNotFound, message)
}

func (h *Handler) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	h.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

func (h *Handler) contentTooLargeResponse(w http.ResponseWriter, r *http.Request) {
	message := "the request body is too large"
	h.errorResponse(w, r, http.StatusRequestEntityTooLarge, message)
}

func (h *Handler) unsupportedMediaTypeResponse(w http.ResponseWriter, r *http.Request) {
	message := "the file type is not supported for this resource"
	h.errorResponse(w, r, http.StatusUnsupportedMediaType, message)
}

func (h *Handler) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (h *Handler) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	h.errorResponse(w, r, http.StatusConflict, message)
}

func (h *Handler) failedValidationResponse(w http.ResponseWriter, r *http.Request, err error) {
	h.errorResponse(w, r, http.StatusUnprocessableEntity, err.Error())
}

func (h *Handler) invalidCredentialsResponse(w http.ResponseWriter, r *http.Request) {
	message := "invalid authentication credentials"
	h.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (h *Handler) invalidAuthenticationTokenResponse(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	message := "invalid or missing authentication token"
	h.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (h *Handler) authenticationRequiredResponse(w http.ResponseWriter, r *http.Request) {
	message := "you must be authenticated to access this resource"
	h.errorResponse(w, r, http.StatusUnauthorized, message)
}

func (h *Handler) inactiveAccountResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account must be activated to access this resource"
	h.errorResponse(w, r, http.StatusForbidden, message)
}

func (h *Handler) notPermittedResponse(w http.ResponseWriter, r *http.Request) {
	message := "your user account doesn't have the necessary permissions to access this resource"
	h.errorResponse(w, r, http.StatusForbidden, message)
}

func (h *Handler) recordAlreadyExistsResponse(w http.ResponseWriter, r *http.Request) {
	message := "a record for this resource already exists for your user account"
	h.errorResponse(w, r, http.StatusUnprocessableEntity, message)
}

func (h *Handler) passwordMismatchResponse(w http.ResponseWriter, r *http.Request) {
	message := "passwords do not match"
	h.errorResponse(w, r, http.StatusUnprocessableEntity, message)
}

func (h *Handler) rateLimitExceededResponse(w http.ResponseWriter, r *http.Request) {
	message := "rate limit exceeded"
	h.errorResponse(w, r, http.StatusTooManyRequests, message)
}
