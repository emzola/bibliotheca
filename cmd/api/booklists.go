package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/validator"
)

func (app *application) createBooklistHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Private     bool   `json:"private"`
	}
	err := app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	user := app.contextGetUser(r)
	booklist := &data.Booklist{}
	booklist.UserID = user.ID
	booklist.CreatorName = user.Name
	booklist.Name = input.Name
	booklist.Description = input.Description
	booklist.Private = input.Private
	v := validator.New()
	if data.ValidateBooklist(v, booklist); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Booklists.Insert(booklist)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/booklists/%d", booklist.ID))
	err = app.encodeJSON(w, http.StatusCreated, envelope{"booklist": booklist}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showBooklistHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Filters data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "-datetime")
	input.Filters.SortSafeList = []string{"datetime", "-datetime"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	booklistId, err := app.readIDParam(r, "booklistId")
	if err != nil || booklistId < 1 {
		app.notFoundResponse(w, r)
		return
	}
	booklist, err := app.models.Booklists.Get(booklistId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	booklist.Content.Books, booklist.Content.Metadata, err = app.models.Books.GetAllForBooklist(booklistId, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"booklist": booklist}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listBooklistsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Search  string
		Filters data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Search = app.readString(qs, "search", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "created_at", "updated_at", "-id", "-created_at", "-updated_at"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	booklists, metadata, err := app.models.Booklists.GetAll(input.Search, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	for _, booklist := range booklists {
		booklist.Content.Books, booklist.Content.Metadata, err = app.models.Books.GetAllForBooklist(booklist.ID, input.Filters)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"booklists": booklists, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) findBooksForBooklistHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Search  string
		Filters data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Search = app.readString(qs, "search", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "-id"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	books, metadata, err := app.models.Books.FindBooksInBooklist(input.Search, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"books": books, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateBooklistHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "booklistId")
	if err != nil || id < 1 {
		app.notFoundResponse(w, r)
		return
	}
	booklist, err := app.models.Booklists.Get(id)
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
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Private     *bool   `json:"private"`
	}
	err = app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if input.Name != nil {
		booklist.Name = *input.Name
	}
	if input.Description != nil {
		booklist.Description = *input.Description
	}
	if input.Private != nil {
		booklist.Private = *input.Private
	}
	v := validator.New()
	if data.ValidateBooklist(v, booklist); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Booklists.Update(booklist)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"booklist": booklist}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteBooklistHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "booklistId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	err = app.models.Booklists.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "booklist successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) addFavouriteBooklistHandler(w http.ResponseWriter, r *http.Request) {
	booklistId, err := app.readIDParam(r, "booklistId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	user := app.contextGetUser(r)
	err = app.models.Booklists.AddFavouriteForUser(user.ID, booklistId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateBooklistFavourite):
			app.recordAlreadyExistsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "booklist sucessfully added to favourites"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) removeFavouriteBooklistHandler(w http.ResponseWriter, r *http.Request) {
	booklistId, err := app.readIDParam(r, "booklistId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	user := app.contextGetUser(r)
	err = app.models.Booklists.RemoveFavouriteForUser(user.ID, booklistId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "booklist successfully removed from favourites"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listFavouriteBooklistsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	var input struct {
		Filters data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "-datetime")
	input.Filters.SortSafeList = []string{"datetime", "created_at", "updated_at", "-datetime", "-created_at", "-updated_at"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	booklists, metadata, err := app.models.Booklists.GetAllFavouritesForUser(user.ID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	for _, booklist := range booklists {
		booklist.Content.Books, booklist.Content.Metadata, err = app.models.Books.GetAllForBooklist(booklist.ID, input.Filters)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"booklists": booklists, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listUserBooklistsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	var input struct {
		Filters data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafeList = []string{"created_at", "updated_at", "-created_at", "-updated_at"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	booklists, metadata, err := app.models.Booklists.GetAllForUser(user.ID, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	for _, booklist := range booklists {
		booklist.Content.Books, booklist.Content.Metadata, err = app.models.Books.GetAllForBooklist(booklist.ID, input.Filters)
		if err != nil {
			app.serverErrorResponse(w, r, err)
			return
		}
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"booklists": booklists, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) addToBooklistHandler(w http.ResponseWriter, r *http.Request) {
	booklistId, err := app.readIDParam(r, "booklistId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	booklist, err := app.models.Booklists.Get(booklistId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	bookId, err := app.readIDParam(r, "bookId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	err = app.models.Books.AddToBooklist(booklist.ID, bookId)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.models.Booklists.Update(booklist)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully added to booklist"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteFromBooklistHandler(w http.ResponseWriter, r *http.Request) {
	booklistId, err := app.readIDParam(r, "booklistId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	bookId, err := app.readIDParam(r, "bookId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	err = app.models.Books.RemoveFromBooklist(booklistId, bookId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully removed from booklist"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showBookInBooklistHandler(w http.ResponseWriter, r *http.Request) {
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
	err = app.encodeJSON(w, http.StatusOK, envelope{"book": book}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
