package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

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
