package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

// CreateBook godoc
// @Summary Upload a new book
// @Description This endpoint uploads a new book
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param book formData file true "File to upload"
// @Success 201 {object} data.Book
// @Failure 400
// @Failure 413
// @Failure 415
// @Failure 500
// @Router /v1/books [post]
func (h *Handler) createBookHandler(w http.ResponseWriter, r *http.Request) {
	// Set 10MB limit for request body size
	maxBytes := int64(10_485_760)
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	user := h.contextGetUser(r)
	book, err := h.service.CreateBook(user.ID, r)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrContentTooLarge):
			h.contentTooLargeResponse(w, r)
		case errors.Is(err, service.ErrBadRequest):
			h.badRequestResponse(w, r, err)
		case errors.Is(err, service.ErrUnsupportedMediaType):
			h.unsupportedMediaTypeResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/books/%d", book.ID))
	err = h.encodeJSON(w, http.StatusCreated, envelope{"book": book}, headers)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ShowBook godoc
// @Summary Show details of a book
// @Description This endpoint shows the details of a specific book
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book to show"
// @Success 200 {object} data.Book
// @Failure 404
// @Failure 500
// @Router /v1/books/{bookId} [get]
func (h *Handler) showBookHandler(w http.ResponseWriter, r *http.Request) {
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	book, err := h.service.GetBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"book": book}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ListBooks godoc
// @Summary List all books
// @Description This endpoint lists all books
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param search query string false "Query string param for search"
// @Param from_year query string false "Query string param to filter by year"
// @Param to_year query string false "Query string param to filter by year"
// @Param language query string false "Query string param to filter by language"
// @Param extension query string false "Query string param to filter by file extension"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param sort query string false "Sort by ascending or descending order. Asc: id, title, year, size, created_at, popularity. Desc: -id, -title, -year, -size, -created_at, -popularity"
// @Success 200 {array} data.Book
// @Failure 422
// @Failure 500
// @Router /v1/books [get]
func (h *Handler) listBooksHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListBooks
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Search = h.readString(qs, "search", "")
	qsInput.FromYear = h.readInt(qs, "from_year", 0, v)
	qsInput.ToYear = h.readInt(qs, "to_year", 0, v)
	qsInput.Language = h.readCSV(qs, "language", []string{})
	qsInput.Extension = h.readCSV(qs, "extension", []string{})
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "id")
	qsInput.Filters.SortSafeList = []string{"id", "title", "year", "size", "created_at", "popularity", "-id", "-title", "-year", "-size", "-created_at", "-popularity"}
	books, metadata, err := h.service.ListBooks(qsInput.Search, qsInput.FromYear, qsInput.ToYear, qsInput.Language, qsInput.Extension, qsInput.Filters)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"books": books, "metadata": metadata}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// UpdateBook godoc
// @Summary Update the details of a book
// @Description This endpoint updates the details of a specific book
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param book body dto.UpdateBookRequestBody true "JSON Payload required to update a book"
// @Param bookId path int true "ID of book to update"
// @Success 200 {object} data.Book
// @Failure 400
// @Failure 404
// @Failure 409
// @Failure 500
// @Router /v1/books/{bookId} [patch]
func (h *Handler) updateBookHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.UpdateBookRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil || bookID < 1 {
		h.notFoundResponse(w, r)
		return
	}
	book, err := h.service.UpdateBook(bookID, requestBody)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		case errors.Is(err, service.ErrDuplicateRecord):
			h.recordAlreadyExistsResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"book": book}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// updateBookCover godoc
// @Summary Upload a book cover
// @Description This endpoint uploads a book cover
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book for cover upload"
// @Param cover formData file true "Image to upload"
// @Success 201 {object} data.Book
// @Failure 400
// @Failure 404
// @Failure 409
// @Failure 413
// @Failure 415
// @Failure 500
// @Router /v1/books/{bookId}/cover [post]
func (h *Handler) updateBookCoverHandler(w http.ResponseWriter, r *http.Request) {
	// Set 3MB limit for request body size
	maxBytes := int64(2_097_152)
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	book, err := h.service.UpdateBookCover(bookID, r)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrContentTooLarge):
			h.contentTooLargeResponse(w, r)
		case errors.Is(err, service.ErrBadRequest):
			h.badRequestResponse(w, r, err)
		case errors.Is(err, service.ErrUnsupportedMediaType):
			h.unsupportedMediaTypeResponse(w, r)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"book": book}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DeleteBook godoc
// @Summary Delete a book
// @Description This endpoint deletes a specific book
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book to delete"
// @Success 200
// @Failure 404
// @Failure 500
// @Router /v1/books/{bookId} [delete]
func (h *Handler) deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	err = h.service.DeleteBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully deleted"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DownloadBook godoc
// @Summary Download a book
// @Description This endpoint downloads a specific book
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book to download"
// @Success 200
// @Failure 403
// @Failure 404
// @Failure 422
// @Failure 500
// @Router /v1/books/{bookId}/download [get]
func (h *Handler) downloadBookHandler(w http.ResponseWriter, r *http.Request) {
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil || bookID < 1 {
		h.notFoundResponse(w, r)
		return
	}
	userID := h.contextGetUser(r).ID
	err = h.service.DownloadBook(bookID, userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		case errors.Is(err, service.ErrNotPermitted):
			h.notPermittedResponse(w, r)
		case errors.Is(err, service.ErrDuplicateRecord):
			h.recordAlreadyExistsResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully downloaded"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DeleteBookFromDownloads godoc
// @Summary Delete a book from downloads
// @Description This endpoint deletes a specific book from download history
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book to delete from download history"
// @Success 200
// @Failure 404
// @Failure 500
// @Router /v1/books/{bookId}/download [delete]
func (h *Handler) deleteBookFromDownloadsHandler(w http.ResponseWriter, r *http.Request) {
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	user := h.contextGetUser(r)
	err = h.service.DeleteBookFromDownloads(user.ID, bookID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully removed from download history"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// FavouriteBook godoc
// @Summary Mark a book as favourite
// @Description This endpoint marks a specific book as favourite
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book to favourite"
// @Success 200
// @Failure 422
// @Failure 500
// @Router /v1/books/{bookId}/favourite [post]
func (h *Handler) favouriteBookHandler(w http.ResponseWriter, r *http.Request) {
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	user := h.contextGetUser(r)
	err = h.service.FavouriteBook(user.ID, bookID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDuplicateRecord):
			h.recordAlreadyExistsResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "book sucessfully added to favourites"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DeleteFavouriteBook godoc
// @Summary Delete a book from favourites
// @Description This endpoint deletes a specific book from favourites
// @Tags books
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book to delete from favourites"
// @Success 200
// @Failure 404
// @Failure 500
// @Router /v1/books/{bookId}/favourite [delete]
func (h *Handler) deleteFavouriteBookHandler(w http.ResponseWriter, r *http.Request) {
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	user := h.contextGetUser(r)
	err = h.service.DeleteFavouriteBook(user.ID, bookID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDuplicateRecord):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully removed from favourites"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
