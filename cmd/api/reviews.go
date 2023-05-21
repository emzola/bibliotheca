package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/validator"
)

func (app *application) createReviewHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Rating  int8   `json:"rating"`
		Comment string `json:"comment"`
	}
	err := app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	id, err := app.readIDParam(r, "bookId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	book, err := app.models.Books.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	user := app.contextGetUser(r)
	review := &data.Review{}
	review.BookID = book.ID
	review.UserID = user.ID
	review.Rating = input.Rating
	review.Comment = input.Comment
	v := validator.New()
	if data.ValidateReview(v, review); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Reviews.Insert(review)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/books/%d/reviews/%d", book.ID, review.ID))
	err = app.encodeJSON(w, http.StatusCreated, envelope{"Review": review}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showReviewHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "reviewId")
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}
	review, err := app.models.Reviews.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"review": review}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateReviewHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "reviewId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	review, err := app.models.Reviews.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// The input struct holds expected data from decoded JSON input. This is done to limit the input fields
	// that can be supplied by the client as JSON (e.g so that a client doesn't supply an ID field).
	// The fields are set to a pointer type to allow partial updates based on nil value.
	var input struct {
		Rating  *int8   `json:"rating"`
		Comment *string `json:"comment"`
	}
	err = app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if input.Rating != nil {
		review.Rating = *input.Rating
	}
	if input.Comment != nil {
		review.Comment = *input.Comment
	}
	v := validator.New()
	if data.ValidateReview(v, review); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Reviews.Update(review)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"review": review}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteReviewHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "reviewId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	err = app.models.Reviews.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "review deleted successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
