package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

// CreateReview godoc
// @Summary Create a new book review
// @Description This endpoint creates a new book request
// @Tags reviews
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book for review"
// @Param body body dto.CreateReviewRequestBody true "JSON payload required to create a book review"
// @Success 201 {object} data.Review
// @Failure 400
// @Failure 404
// @Failure 422
// @Failure 500
// @Router /v1/books/{bookId}/reviews [post]
func (h *Handler) createReviewHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.CreateReviewRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	user := h.contextGetUser(r)
	review, err := h.service.CreateReview(user.ID, bookID, user.Name, requestBody.Rating, requestBody.Comment)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		case errors.Is(err, service.ErrDuplicateRecord):
			h.recordAlreadyExistsResponse(w, r)
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/books/%d/reviews/%d", bookID, review.ID))
	err = h.encodeJSON(w, http.StatusCreated, envelope{"review": review}, headers)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ShowReview godoc
// @Summary Show details of a book review
// @Description This endpoint shows the details of a specific book review
// @Tags reviews
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book for review"
// @Param reviewId path int true "ID of review to show"
// @Success 200 {object} data.Review
// @Failure 404
// @Failure 500
// @Router /v1/books/{bookId}/reviews/{reviewId} [get]
func (h *Handler) showReviewHandler(w http.ResponseWriter, r *http.Request) {
	reviewID, err := h.readIDParam(r, "reviewId")
	if err != nil || reviewID < 1 {
		h.notFoundResponse(w, r)
		return
	}
	review, err := h.service.GetReview(reviewID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"review": review}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ShowReview godoc
// @Summary Show details of a book review
// @Description This endpoint shows the details of a specific book review
// @Tags reviews
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book for review"
// @Param reviewId path int true "ID of review to update"
// @Success 200 {object} data.Review
// @Failure 400
// @Failure 404
// @Failure 409
// @Failure 422
// @Failure 500
// @Router /v1/books/{bookId}/reviews/{reviewId} [patch]
func (h *Handler) updateReviewHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.UpdateReviewRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	reviewID, err := h.readIDParam(r, "reviewId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	review, err := h.service.UpdateReview(reviewID, bookID, requestBody.Rating, requestBody.Comment)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"review": review}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DeleteReview godoc
// @Summary Delete a book review
// @Description This endpoint deletes a book review
// @Tags reviews
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book for review"
// @Param reviewId path int true "ID of review to delete"
// @Success 200
// @Failure 404
// @Failure 500
// @Router /v1/books/{bookId}/reviews/{reviewId} [delete]
func (h *Handler) deleteReviewHandler(w http.ResponseWriter, r *http.Request) {
	reviewID, err := h.readIDParam(r, "reviewId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	err = h.service.DeleteReview(reviewID, bookID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "review deleted successfully"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ListReviews godoc
// @Summary List all reviews
// @Description This endpoint lists all reviews
// @Tags reviews
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book for review"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param sort query string false "Sort by ascending or descending order. Asc: id, vote. Desc: -id, -vote"
// @Success 200 {array} data.Review
// @Failure 422
// @Failure 500
// @Router /v1/books/{bookId}/reviews [get]
func (h *Handler) listReviewsHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListReviews
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "id")
	qsInput.Filters.SortSafeList = []string{"id", "vote", "-id", "-vote"}
	ratings, reviews, metadata, err := h.service.ListReviews(qsInput.Filters)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"ratings": ratings, "reviews": reviews, "metadata": metadata}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
