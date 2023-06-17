package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/validator"
)

func (app *application) createCommentHandler(w http.ResponseWriter, r *http.Request) {
	booklistId, err := app.readIDParam(r, "booklistId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	var input struct {
		Content string `json:"content"`
	}
	err = app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	user := app.contextGetUser(r)
	comment := &data.Comment{}
	comment.BooklistID = booklistId
	comment.UserID = user.ID
	comment.UserName = user.Name
	comment.Content = input.Content
	v := validator.New()
	if data.ValidateComment(v, comment); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Comments.Insert(comment)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/booklists/%d/comments/%d", booklistId, comment.ID))
	err = app.encodeJSON(w, http.StatusCreated, envelope{"comment": comment}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateCommentHandler(w http.ResponseWriter, r *http.Request) {
	commentId, err := app.readIDParam(r, "commentId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	comment, err := app.models.Comments.Get(commentId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	var input struct {
		Content *string `json:"content"`
	}
	err = app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	v := validator.New()
	if data.ValidateComment(v, comment); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	if input.Content != nil {
		comment.Content = *input.Content
	}
	err = app.models.Comments.Update(comment)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"comment": comment}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteCommentHandler(w http.ResponseWriter, r *http.Request) {
	commentId, err := app.readIDParam(r, "commentId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	err = app.models.Comments.Delete(commentId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "comment deleted successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listCommentsHandler(w http.ResponseWriter, r *http.Request) {
	booklistId, err := app.readIDParam(r, "booklistId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	comments, err := app.models.Comments.GetAll(booklistId)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"comments": comments}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) createCommentReplyHandler(w http.ResponseWriter, r *http.Request) {
	booklistId, err := app.readIDParam(r, "booklistId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	commentId, err := app.readIDParam(r, "commentId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	var input struct {
		Content string `json:"content"`
	}
	err = app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	user := app.contextGetUser(r)
	comment := &data.Comment{}
	comment.ParentID = commentId
	comment.BooklistID = booklistId
	comment.UserID = user.ID
	comment.Content = input.Content
	v := validator.New()
	if data.ValidateComment(v, comment); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Comments.InsertReply(comment)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
	err = app.encodeJSON(w, http.StatusCreated, envelope{"comment": comment}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
