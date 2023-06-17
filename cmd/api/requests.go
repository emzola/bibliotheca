package main

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/validator"
)

func (app *application) createRequestHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Isbn string `json:"isbn"`
	}
	err := app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	v := validator.New()
	if v.Check(input.Isbn != "", "isbn", "must be provided"); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	// Fetch JSON data for a book from openlibrary api
	bookJSON := &data.BookJSONData{}
	url := "https://openlibrary.org/isbn/" + input.Isbn + ".json"
	err = app.fetchRemoteResource(app.client(), url, &bookJSON)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	user := app.contextGetUser(r)
	request := &data.Request{}
	request.UserID = user.ID
	request.Title = bookJSON.Title
	request.Publisher = bookJSON.Publisher[0]
	request.Isbn = input.Isbn
	dateString := strings.Split(bookJSON.Date, ",")
	year, err := strconv.Atoi(strings.TrimSpace(dateString[1]))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	request.Year = int32(year)
	request.Expiry = time.Now().Add(time.Hour * 24 * 182)
	request.Status = "active"
	err = app.models.Requests.Insert(request)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Add the new request to the users_requests table
	err = app.models.Requests.AddForUser(user.ID, request.ID, request.Expiry)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateRequest):
			app.recordAlreadyExistsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Update request waitlist
	request.Waitlist++
	err = app.models.Requests.Update(request)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Encode to JSON as usual
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/requests/%d", request.ID))
	err = app.encodeJSON(w, http.StatusCreated, envelope{"request": request}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showRequestHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r, "requestId")
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	request, err := app.models.Requests.Get(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"request": request}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listRequestsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Search  string
		Status  string
		Filters data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Search = app.readString(qs, "search", "")
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Status = app.readString(qs, "status", "active")
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "-id"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	requests, metadata, err := app.models.Requests.GetAll(input.Search, input.Status, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"requests": requests, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listUserRequestsHandler(w http.ResponseWriter, r *http.Request) {
	user := app.contextGetUser(r)
	var input struct {
		Status  string
		Filters data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Status = app.readString(qs, "status", "active")
	input.Filters.Sort = app.readString(qs, "sort", "-datetime")
	input.Filters.SortSafeList = []string{"datetime", "-datetime"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	requests, metadata, err := app.models.Requests.GetAllForUser(user.ID, input.Status, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"requests": requests, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) subscribeRequestHandler(w http.ResponseWriter, r *http.Request) {
	requestId, err := app.readIDParam(r, "requestId")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	request, err := app.models.Requests.Get(requestId)
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
	err = app.models.Requests.AddForUser(user.ID, requestId, time.Now().Add(time.Hour*24*182))
	if err != nil {
		switch {
		case errors.Is(err, data.ErrDuplicateRequest):
			app.recordAlreadyExistsResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Update request waitlist
	request.Waitlist++
	err = app.models.Requests.Update(request)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "you've successfully subscribed to this book request"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) unsubscribeRequestHandler(w http.ResponseWriter, r *http.Request) {
	requestId, err := app.readIDParam(r, "requestId")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	request, err := app.models.Requests.Get(requestId)
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
	err = app.models.Requests.DeleteForUser(user.ID, requestId)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	// Update request waitlist
	request.Waitlist--
	err = app.models.Requests.Update(request)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "you've successfully unsubscribed from this book request"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
