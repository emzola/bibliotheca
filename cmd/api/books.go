package main

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/emzola/bibliotheca/internal/data"
)

func (app *application) createBookHandler(w http.ResponseWriter, r *http.Request) {
	maxBytes := int64(10_485_760)
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	err := r.ParseMultipartForm(5000)
	if err != nil {
		switch {
		case err.Error() == "http: request body too large":
			app.contentTooLargeResponse(w, r)
		default:
			app.badRequestResponse(w, r, err)
		}
		return
	}
	file, fileHeader, err := r.FormFile("book")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	defer file.Close()
	buffer, mtype, err := app.detectMimeType(file, fileHeader)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidMimeType):
			app.unsupportedMediaTypeResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	s3FileKey, err := app.uploadFileToS3(app.config.s3.client, buffer, mtype, fileHeader)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	book := &data.Book{
		Title:     strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename)),
		S3FileKey: s3FileKey,
		AdditionalInfo: data.AdditionalInfo{
			Filename: fileHeader.Filename,
			Size:     app.formatFileSize(fileHeader.Size),
		},
	}
	err = app.models.Book.Insert(book)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/books/%d", book.ID))
	err = app.encodeJSON(w, http.StatusCreated, envelope{"book": book}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) showBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	book, err := app.models.Book.Get(id)
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
