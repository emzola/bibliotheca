package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/service"
)

// CreateComment godoc
// @Summary Create a new booklist comment
// @Description This endpoint creates a new booklist comment
// @Tags comments
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param booklistId path int true "ID of booklist for comment"
// @Param body body dto.CreateCommentRequestBody true "JSON payload required to create a booklist comment"
// @Success 201 {object} data.Comment
// @Failure 400
// @Failure 404
// @Failure 422
// @Failure 500
// @Router /v1/booklists/{booklistId}/comments [post]
func (h *Handler) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.CreateCommentRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	user := h.contextGetUser(r)
	comment, err := h.service.CreateComment(user.ID, booklistID, user.Name, requestBody.Content)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/booklists/%d/comments/%d", booklistID, comment.ID))
	err = h.encodeJSON(w, http.StatusCreated, envelope{"comment": comment}, headers)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// UpdateComment godoc
// @Summary Update the details of a booklist comment
// @Description This endpoint updates the details of a specific booklist comment
// @Tags comments
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param comment body dto.UpdateCommentRequestBody true "JSON Payload required to update a booklist comment"
// @Param commentId path int true "ID of comment to update"
// @Success 200 {object} data.Comment
// @Failure 400
// @Failure 404
// @Failure 409
// @Failure 500
// @Router /v1/booklists/{booklistId}/comments [patch]
func (h *Handler) updateCommentHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.UpdateCommentRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	commentID, err := h.readIDParam(r, "commentId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	comment, err := h.service.UpdateComment(commentID, requestBody.Content)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"comment": comment}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DeleteComment godoc
// @Summary Delete a booklist comment
// @Description This endpoint deletes a specific booklist comment
// @Tags comments
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param commentId path int true "ID of booklist comment to delete"
// @Success 200
// @Failure 404
// @Failure 500
// @Router /v1/booklists/{booklistId}/comments [delete]
func (h *Handler) deleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	commentID, err := h.readIDParam(r, "commentId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	err = h.service.DeleteComment(commentID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "comment deleted successfully"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ListComments godoc
// @Summary List all comments
// @Description This endpoint lists all comments
// @Tags comments
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Success 200 {array} data.Comment
// @Failure 404
// @Failure 500
// @Router /v1/booklists/{booklistId}/comments [get]
func (h *Handler) listCommentsHandler(w http.ResponseWriter, r *http.Request) {
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	comments, err := h.service.ListComments(booklistID)
	if err != nil {
		h.serverErrorResponse(w, r, err)
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"comments": comments}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// CreateCommentReply godoc
// @Summary Reply a booklist comment
// @Description This endpoint creates a new comment child
// @Tags comments
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param booklistId path int true "ID of booklist for comment"
// @Param commentId path int true "ID of parent comment"
// @Param body body dto.CreateCommentReplyRequestBody true "JSON payload required to create a comment child"
// @Success 201 {object} data.Comment
// @Failure 400
// @Failure 404
// @Failure 422
// @Failure 500
// @Router /v1/booklists/{booklistId}/comments/{commentId} [post]
func (h *Handler) createCommentReplyHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.CreateCommentReplyRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	commentID, err := h.readIDParam(r, "commentId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	user := h.contextGetUser(r)
	comment, err := h.service.CreateCommentReply(user.ID, booklistID, commentID, requestBody.Content)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusCreated, envelope{"comment": comment}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
