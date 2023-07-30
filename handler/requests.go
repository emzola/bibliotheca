package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

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
