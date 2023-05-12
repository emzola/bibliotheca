package main

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrInvalidMimeType = errors.New("content type is not supported")
)

func (app *application) logError(r *http.Request, err error) {
	app.logger.PrintError(err, map[string]string{
		"request_method": r.Method,
		"request_url":    r.URL.String(),
	})
}

func (app *application) errorResponse(w http.ResponseWriter, r *http.Request, status int, message interface{}) {
	env := envelope{"error": message}
	err := app.encodeJSON(w, status, env, nil)
	if err != nil {
		app.logError(r, err)
		w.WriteHeader(500)
	}
}

func (app *application) serverErrorResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.logError(r, err)
	message := "the server encountered a problem and could not process your request"
	app.errorResponse(w, r, http.StatusInternalServerError, message)
}

func (app *application) notFoundResponse(w http.ResponseWriter, r *http.Request) {
	message := "the requested resource could not be found"
	app.errorResponse(w, r, http.StatusNotFound, message)
}

func (app *application) methodNotAllowed(w http.ResponseWriter, r *http.Request) {
	message := fmt.Sprintf("the %s method is not supported for this resource", r.Method)
	app.errorResponse(w, r, http.StatusMethodNotAllowed, message)
}

// func (app *application) fileNotExistResponse(w http.ResponseWriter, r *http.Request, err error) {
// 	app.logError(r, err)
// 	message := "the requested file does not exist"
// 	app.errorResponse(w, r, http.StatusInternalServerError, message)
// }

func (app *application) contentTooLargeResponse(w http.ResponseWriter, r *http.Request) {
	message := "the request body is too large"
	app.errorResponse(w, r, http.StatusRequestEntityTooLarge, message)
}

func (app *application) unsupportedMediaTypeResponse(w http.ResponseWriter, r *http.Request) {
	message := "the file type is not supported for this resource"
	app.errorResponse(w, r, http.StatusUnsupportedMediaType, message)
}

func (app *application) badRequestResponse(w http.ResponseWriter, r *http.Request, err error) {
	app.errorResponse(w, r, http.StatusBadRequest, err.Error())
}

func (app *application) editConflictResponse(w http.ResponseWriter, r *http.Request) {
	message := "unable to update the record due to an edit conflict, please try again"
	app.errorResponse(w, r, http.StatusConflict, message)
}

func (app *application) failedValidationResponse(w http.ResponseWriter, r *http.Request, errors map[string]string) {
	app.errorResponse(w, r, http.StatusUnprocessableEntity, errors)
}
