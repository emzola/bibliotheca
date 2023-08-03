package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

// CreateRequest godoc
// @Summary Create a new book request
// @Description This endpoint creates a new book request
// @Tags requests
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param body body dto.CreateRequestRequestBody true "JSON payload required to create a book request"
// @Success 201 {object} data.Request
// @Failure 400
// @Failure 404
// @Failure 422
// @Failure 500
// @Router /v1/requests [post]
func (h *Handler) createRequestHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.CreateRequestRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	user := h.contextGetUser(r)
	request, err := h.service.CreateRequest(user.ID, requestBody.Isbn)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		case errors.Is(err, service.ErrDuplicateRecord):
			h.recordAlreadyExistsResponse(w, r)
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/requests/%d", request.ID))
	err = h.encodeJSON(w, http.StatusCreated, envelope{"request": request}, headers)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ShowRequest godoc
// @Summary Show details of a book request
// @Description This endpoint shows the details of a specific book request
// @Tags requests
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param requestId path int true "ID of request to show"
// @Success 200 {object} data.Request
// @Failure 404
// @Failure 500
// @Router /v1/requests/{requestId} [get]
func (h *Handler) showRequestHandler(w http.ResponseWriter, r *http.Request) {
	requestID, err := h.readIDParam(r, "requestId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	request, err := h.service.GetRequest(requestID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"request": request}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ListRequests godoc
// @Summary List all book requests
// @Description This endpoint lists all book requests
// @Tags requests
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param search query string false "Query string param for search"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param status query int false "Query string param for book status (options: active, expired, completed)"
// @Param sort query string false "Sort by ascending or descending order. Asc: id. Desc: -id"
// @Success 200 {array} data.Request
// @Failure 422
// @Failure 500
// @Router /v1/requests [get]
func (h *Handler) listRequestsHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListRequest
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Search = h.readString(qs, "search", "")
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Status = h.readString(qs, "status", "active")
	qsInput.Filters.Sort = h.readString(qs, "sort", "id")
	qsInput.Filters.SortSafeList = []string{"id", "-id"}
	requests, metadata, err := h.service.ListRequests(qsInput.Search, qsInput.Status, qsInput.Filters)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"requests": requests, "metadata": metadata}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// SubscribeRequest godoc
// @Summary Subscribe to a book request
// @Description This endpoint subscribes to a book request
// @Tags requests
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of request to subscribe to"
// @Success 200
// @Failure 400
// @Failure 404
// @Failure 409
// @Failure 422
// @Failure 500
// @Router /v1/requests/{requestId}/subscribe [post]
func (h *Handler) subscribeRequestHandler(w http.ResponseWriter, r *http.Request) {
	requestID, err := h.readIDParam(r, "requestId")
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	user := h.contextGetUser(r)
	err = h.service.SubscribeRequest(user.ID, requestID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		case errors.Is(err, service.ErrDuplicateRecord):
			h.recordAlreadyExistsResponse(w, r)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "you've successfully subscribed to this book request"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// UnsubscribeRequest godoc
// @Summary Unsubscribe from a book request
// @Description This endpoint unsubscribes from a book request
// @Tags requests
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of request to unsubscribe from"
// @Success 200
// @Failure 400
// @Failure 404
// @Failure 422
// @Failure 500
// @Router /v1/requests/{requestId}/unsubscribe [delete]
func (h *Handler) unsubscribeRequestHandler(w http.ResponseWriter, r *http.Request) {
	requestID, err := h.readIDParam(r, "requestId")
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	user := h.contextGetUser(r)
	err = h.service.UnsubscribeRequest(user.ID, requestID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "you've successfully unsubscribed from this book request"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
