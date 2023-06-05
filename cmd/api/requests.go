package main

import (
	"net/http"
	"strconv"
	"strings"

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
	// Fetch Author data from openlibrary api
	author := &data.Author{}
	url = "https://openlibrary.org" + bookJSON.Author[0].Key + ".json"
	err = app.fetchRemoteResource(app.client(), url, &author)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	// Fetch Language data from openlibrary api
	language := &data.Language{}
	url = "https://openlibrary.org" + bookJSON.Language[0].Key + ".json"
	err = app.fetchRemoteResource(app.client(), url, &language)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	user := app.contextGetUser(r)
	request := &data.Request{}
	request.UserID = user.ID
	request.Title = bookJSON.Title
	request.Author = []string{author.Name}
	request.Publisher = bookJSON.Publisher[0]
	request.Isbn = input.Isbn
	request.Language = language.Name
	dateString := strings.Split(bookJSON.Date, ",")
	year, err := strconv.Atoi(strings.TrimSpace(dateString[1]))
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	request.Year = int32(year)
	err = app.models.Requests.Insert(request)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.encodeJSON(w, http.StatusCreated, envelope{"request": request}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
