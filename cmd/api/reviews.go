package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/validator"
)

func (app *application) createReviewHandler(w http.ResponseWriter, r *http.Request) {
	// First check whether a review from user already exists.
	// If it does, do not process further create request
	bookId, err := app.readIDParam(r, "bookId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	user := app.contextGetUser(r)
	exists := app.models.Reviews.RecordExistsForUser(bookId, user.ID)
	if exists {
		app.recordAlreadyExistsResponse(w, r)
		return
	}
	// From this point, create review as usual since user
	// does not have any review record
	var input struct {
		Rating  int8   `json:"rating"`
		Comment string `json:"comment"`
	}
	err = app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	book, err := app.models.Books.Get(bookId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
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
	// Get ratings and Update the popularity field of a book
	ratings, err := app.models.Reviews.GetRatings()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	book.Popularity = ratings.Average
	err = app.models.Books.Update(book)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Set location header for the newly created review and encode it to JSON
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
	// Get ratings and Update the popularity field of a book
	bookId, err := app.readIDParam(r, "bookId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	book, err := app.models.Books.Get(bookId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	rating, err := app.models.Reviews.GetRatings()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	book.Popularity = rating.Average
	err = app.models.Books.Update(book)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"review": review}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listReviewsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filters data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "vote", "-id", "-vote"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	ratings, reviews, metadata, err := app.models.Reviews.GetAll(input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"ratings": ratings, "reviews": reviews, "metadata": metadata}, nil)
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
	// Get ratings and update the popularity field of a book
	bookId, err := app.readIDParam(r, "bookId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	book, err := app.models.Books.Get(bookId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	rating, err := app.models.Reviews.GetRatings()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	book.Popularity = rating.Average
	err = app.models.Books.Update(book)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// encode delete success message to JSON
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "review deleted successfully"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) upvoteReviewHandler(w http.ResponseWriter, r *http.Request) {
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
	user := app.contextGetUser(r)
	if review.UserID != user.ID {
		review.Vote += 1
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

func (app *application) downvoteReviewHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
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
	if review.UserID != user.ID {
		review.Vote -= 1
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
