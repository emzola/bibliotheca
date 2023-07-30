package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/service"
)

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
