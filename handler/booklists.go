package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

// CreateBooklist godoc
// @Summary Create a new booklist
// @Description This endpoint creates a new booklist
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param body body dto.CreateBooklistRequestBody true "JSON payload required to create a booklist"
// @Success 201 {object} data.Booklist
// @Failure 400
// @Failure 422
// @Failure 500
// @Router /v1/booklists [post]
func (h *Handler) createBooklistHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.CreateBooklistRequestBody
	if err := h.decodeJSON(w, r, &requestBody); err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	user := h.contextGetUser(r)
	booklist, err := h.service.CreateBooklist(requestBody.Name, requestBody.Description, requestBody.Private, user.ID, user.Name)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/booklists/%d", booklist.ID))
	if err := h.encodeJSON(w, http.StatusCreated, envelope{"booklist": booklist}, headers); err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ShowBooklist godoc
// @Summary Show details of a booklist
// @Description This endpoint shows the details of a specific booklist
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param booklistId path int true "ID of booklist to show"
// @Success 200 {object} data.Booklist
// @Failure 404
// @Failure 422
// @Failure 500
// @Router /v1/booklists/{booklistId} [get]
func (h *Handler) showBooklistHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsShowBooklist
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "-datetime")
	qsInput.Filters.SortSafeList = []string{"datetime", "-datetime"}
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	booklist, err := h.service.GetBooklist(booklistID, qsInput.Filters)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"booklist": booklist}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ListBooklists godoc
// @Summary List all booklists
// @Description This endpoint lists all booklists
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param search query string false "Query string param for search"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param sort query string false "Sort by ascending or descending order. Asc: id, created_at, updated_at. Desc: -id, -created_at, -updated_at"
// @Success 200 {array} data.Booklist
// @Failure 422
// @Failure 500
// @Router /v1/booklists [get]
func (h *Handler) listBooklistsHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListBooklists
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Search = h.readString(qs, "search", "")
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "id")
	qsInput.Filters.SortSafeList = []string{"id", "created_at", "updated_at", "-id", "-created_at", "-updated_at"}
	booklists, metadata, err := h.service.ListBooklist(qsInput.Search, qsInput.Filters)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"booklists": booklists, "metadata": metadata}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// FindBooksForBooklist godoc
// @Summary Find books for a booklist
// @Description This endpoint searches for books in a specific booklist
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param search query string false "Query string param for search"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param sort query string false "Sort by ascending or descending order. Asc: id. Desc: -id."
// @Success 200 {array} data.Book
// @Failure 422
// @Failure 500
// @Router /v1/booklists/{booklistId}/books [get]
func (h *Handler) findBooksForBooklistHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsFindBooksForBooklist
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Search = h.readString(qs, "search", "")
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "id")
	qsInput.Filters.SortSafeList = []string{"id", "-id"}
	books, metadata, err := h.service.FindBooksForBooklist(qsInput.Search, qsInput.Filters)
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

// UpdateBooklist godoc
// @Summary Update the details of a booklist
// @Description This endpoint updates the details of a specific booklist
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param booklist body dto.UpdateBooklistRequestBody true "JSON Payload required to update a booklist"
// @Param booklistId path int true "ID of booklist to update"
// @Success 200 {object} data.Booklist
// @Failure 400
// @Failure 404
// @Failure 409
// @Failure 500
// @Router /v1/booklists/{booklistId} [patch]
func (h *Handler) updateBooklistHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.UpdateBooklistRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	booklist, err := h.service.UpdateBooklist(booklistID, requestBody.Name, requestBody.Description, requestBody.Private)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"booklist": booklist}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DeleteBooklist godoc
// @Summary Delete a booklist
// @Description This endpoint deletes a specific booklist
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param booklistId path int true "ID of booklist to delete"
// @Success 200
// @Failure 404
// @Failure 500
// @Router /v1/booklists/{booklistId} [delete]
func (h *Handler) deleteBooklistHandler(w http.ResponseWriter, r *http.Request) {
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	err = h.service.DeleteBooklist(booklistID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "booklist successfully deleted"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// FavouriteBooklist godoc
// @Summary Mark a booklist as favourite
// @Description This endpoint marks a specific booklist as favourite
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param booklistId path int true "ID of booklist to favourite"
// @Success 200
// @Failure 422
// @Failure 500
// @Router /v1/booklists/{booklistId}/favourite [post]
func (h *Handler) favouriteBooklistHandler(w http.ResponseWriter, r *http.Request) {
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	user := h.contextGetUser(r)
	err = h.service.FavouriteBooklist(user.ID, booklistID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrDuplicateRecord):
			h.recordAlreadyExistsResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "booklist sucessfully added to favourites"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DeleteFavouriteBooklist godoc
// @Summary Delete a booklist from favourites
// @Description This endpoint deletes a specific booklist from favourites
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param booklistId path int true "ID of booklist to delete from favourites"
// @Success 200
// @Failure 404
// @Failure 500
// @Router /v1/booklists/{booklistId}/favourite [delete]
func (h *Handler) DeleteFavouriteBooklistHandler(w http.ResponseWriter, r *http.Request) {
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	user := h.contextGetUser(r)
	err = h.service.DeleteFavouriteBooklist(user.ID, booklistID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "booklist successfully removed from favourites"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// AddBookToBooklist godoc
// @Summary Add a book to a booklist
// @Description This endpoint adds a book to a booklist
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param booklistId path int true "ID of booklist to add book to"
// @Param bookId path int true "ID of book to add"
// @Success 200
// @Failure 422
// @Failure 500
// @Router /v1/booklists/{booklistId}/books/{bookId} [post]
func (h *Handler) addBookToBooklistHandler(w http.ResponseWriter, r *http.Request) {
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	err = h.service.AddBookToBooklist(bookID, booklistID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully added to booklist"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DeleteBookFromBooklist godoc
// @Summary Delete a book from a booklist
// @Description This endpoint deletes a book from a booklist
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param booklistId path int true "ID of booklist to delete book from"
// @Param bookId path int true "ID of book to delete"
// @Success 200
// @Success 404
// @Failure 500
// @Router /v1/booklists/{booklistId}/books/{bookId} [delete]
func (h *Handler) deleteBookFromBooklistHandler(w http.ResponseWriter, r *http.Request) {
	booklistID, err := h.readIDParam(r, "booklistId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	err = h.service.DeleteBookFromBooklist(bookID, booklistID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "book successfully removed from booklist"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// ShowBookInBooklist godoc
// @Summary Show details of a book in a booklist
// @Description This endpoint shows the details of a book when searched for in a booklist
// @Tags booklists
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param bookId path int true "ID of book to show"
// @Success 200 {object} data.Book
// @Failure 404
// @Failure 500
// @Router /v1/booklists/{booklistId}/books/{bookId} [get]
func (h *Handler) showBookInBooklistHandler(w http.ResponseWriter, r *http.Request) {
	bookID, err := h.readIDParam(r, "bookId")
	if err != nil {
		h.notFoundResponse(w, r)
		return
	}
	book, err := h.service.ShowBookInBooklist(bookID)
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
