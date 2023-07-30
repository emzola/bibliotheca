package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

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

func (h *Handler) updateBooklistHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.UpdateBooklist
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
	err = h.encodeJSON(w, http.StatusOK, envelope{"booklist": booklist}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

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
