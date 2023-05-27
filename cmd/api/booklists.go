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
	booklist.Username = user.Name
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
	err = app.encodeJSON(w, http.StatusOK, envelope{"booklist": booklist}, nil)
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
