package main

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/validator"
)

func (app *application) createBookHandler(w http.ResponseWriter, r *http.Request) {
	maxBytes := int64(10_485_760)
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	err := r.ParseMultipartForm(5000)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &maxBytesError):
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
	buffer, mtype, err := app.detectMimeType(file, fileHeader, ScopeBook)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidMimeType):
			app.unsupportedMediaTypeResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	s3FileKey, err := app.uploadFileToS3(app.config.s3.client, buffer, mtype, fileHeader, ScopeBook)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	book := &data.Book{}
	book.UserID = app.contextGetUser(r).ID
	book.Title = strings.TrimSuffix(fileHeader.Filename, filepath.Ext(fileHeader.Filename))
	book.S3FileKey = s3FileKey
	book.Filename = fileHeader.Filename
	book.Extension = strings.ToUpper(strings.TrimPrefix(filepath.Ext(fileHeader.Filename), "."))
	book.Size = fileHeader.Size
	err = app.models.Books.Insert(book)
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

func (app *application) listBooksHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title     string
		Author    []string
		Isbn10    string
		Isbn13    string
		Publisher string
		FromYear  int
		ToYear    int
		Language  []string
		Extension []string
		Filters   data.Filters
	}
	v := validator.New()
	qs := r.URL.Query()
	input.Title = app.readString(qs, "title", "")
	input.Author = app.readCSV(qs, "author", []string{})
	input.Isbn10 = app.readString(qs, "isbn_10", "")
	input.Isbn13 = app.readString(qs, "isbn_13", "")
	input.Publisher = app.readString(qs, "publisher", "")
	input.FromYear = app.readInt(qs, "from_year", 0, v)
	input.ToYear = app.readInt(qs, "to_year", 0, v)
	input.Language = app.readCSV(qs, "language", []string{})
	input.Extension = app.readCSV(qs, "extension", []string{})
	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 10, v)
	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafeList = []string{"id", "title", "year", "size", "created_at", "popularity", "-id", "-title", "-year", "-size", "-created_at", "-popularity"}
	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	books, metadata, err := app.models.Books.GetAll(input.Title, input.Author, input.Isbn10, input.Isbn13, input.Publisher, input.FromYear, input.ToYear, input.Language, input.Extension, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"books": books, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
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
	// The input struct holds expected data from decoded JSON input. This is done to limit the input fields
	// that can be supplied by the client as JSON (e.g so that a client doesn't supply an ID field).
	// The fields are set to a pointer type to allow partial updates based on nil value.
	var input struct {
		Title       *string  `json:"title"`
		Description *string  `json:"description,omitempty"`
		Author      []string `json:"author,omitempty"`
		Category    *string  `json:"category,omitempty"`
		Publisher   *string  `json:"publisher,omitempty"`
		Language    *string  `json:"language,omitempty"`
		Series      *string  `json:"series,omitempty"`
		Volume      *int32   `json:"volume,omitempty"`
		Edition     *string  `json:"edition,omitempty"`
		Year        *int32   `json:"year,omitempty"`
		PageCount   *int32   `json:"page_count,omitempty"`
		Isbn10      *string  `json:"isbn_10,omitempty"`
		Isbn13      *string  `json:"isbn_13,omitempty"`
		Popularity  *int8    `json:"popularity,omitempty"`
	}
	err = app.decodeJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	if input.Title != nil {
		book.Title = *input.Title
	}
	if input.Description != nil {
		book.Description = *input.Description
	}
	if input.Author != nil {
		book.Author = input.Author
	}
	if input.Category != nil {
		book.Category = *input.Category
	}
	if input.Publisher != nil {
		book.Publisher = *input.Publisher
	}
	if input.Language != nil {
		book.Language = *input.Language
	}
	if input.Series != nil {
		book.Series = *input.Series
	}
	if input.Volume != nil {
		book.Volume = *input.Volume
	}
	if input.Edition != nil {
		book.Edition = *input.Edition
	}
	if input.Year != nil {
		book.Year = *input.Year
	}
	if input.PageCount != nil {
		book.PageCount = *input.PageCount
	}
	if input.Isbn10 != nil {
		book.Isbn10 = *input.Isbn10
	}
	if input.Isbn13 != nil {
		book.Isbn13 = *input.Isbn13
	}
	if input.Popularity != nil {
		book.Popularity = *input.Popularity
	}
	v := validator.New()
	if data.ValidateBook(v, book); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}
	err = app.models.Books.Update(book)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
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

func (app *application) updateBookCoverHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
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
	maxBytes := int64(2_097_152)
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	err = r.ParseMultipartForm(5000)
	if err != nil {
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &maxBytesError):
			app.contentTooLargeResponse(w, r)
		default:
			app.badRequestResponse(w, r, err)
		}
		return
	}
	file, fileHeader, err := r.FormFile("cover")
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}
	buffer, mtype, err := app.detectMimeType(file, fileHeader, ScopeCover)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidMimeType):
			app.unsupportedMediaTypeResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	s3CoverPath, err := app.uploadFileToS3(app.config.s3.client, buffer, mtype, fileHeader, ScopeCover)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}
	book.CoverPath = s3CoverPath
	err = app.models.Books.Update(book)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
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

func (app *application) downloadBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil || id < 1 {
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
	err = app.downloadFileFromS3(app.config.s3.client, book)
	if err != nil {
		app.badRequestResponse(w, r, err)
	}
}

func (app *application) deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	err = app.models.Books.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	err = app.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
