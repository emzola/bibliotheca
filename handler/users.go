package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

func (h *Handler) registerUserHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.RegisterUserRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	user, err := h.service.RegisterUser(requestBody.Name, requestBody.Email, requestBody.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation) || errors.Is(err, service.ErrDuplicateRecord):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *Handler) activateUserHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.ActivateUserRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	user, err := h.service.ActivateUser(requestBody.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *Handler) showUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.contextGetUser(r).ID
	user, err := h.service.ShowUser(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *Handler) updateUserHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.UpdateUserRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	userID := h.contextGetUser(r).ID
	user, err := h.service.UpdateUser(userID, requestBody.Name, requestBody.Email)
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
	err = h.encodeJSON(w, http.StatusOK, envelope{"user": user}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *Handler) updateUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.UpdateUserPasswordRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	userID := h.contextGetUser(r).ID
	user, err := h.service.UpdateUserPassword(userID, requestBody.OldPassword, requestBody.NewPassword, requestBody.ConfirmNewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		case errors.Is(err, service.ErrPasswordMismatch):
			h.passwordMismatchResponse(w, r)
		case errors.Is(err, service.ErrInvalidCredentials):
			h.invalidCredentialsResponse(w, r)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusAccepted, envelope{"user": user}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *Handler) resetUserPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var requestbody dto.ResetUserPasswordRequestBody
	err := h.decodeJSON(w, r, &requestbody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	err = h.service.ResetUserPassword(requestbody.Password, requestbody.TokenPlaintext)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		case errors.Is(err, service.ErrEditConflict):
			h.editConflictResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "your password was successfully reset"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *Handler) deleteUserHandler(w http.ResponseWriter, r *http.Request) {
	userID := h.contextGetUser(r).ID
	err := h.service.DeleteUser(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrRecordNotFound):
			h.notFoundResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "user account successfully deleted"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *Handler) listUserFavouriteBooklistsHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListUserFavouriteBooklists
	user := h.contextGetUser(r)
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "-datetime")
	qsInput.Filters.SortSafeList = []string{"datetime", "created_at", "updated_at", "-datetime", "-created_at", "-updated_at"}
	booklists, metadata, err := h.service.ListUserFavouriteBooklists(user.ID, qsInput.Filters)
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

func (h *Handler) listUserBooklistsHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListUserBooklists
	user := h.contextGetUser(r)
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "-created_at")
	qsInput.Filters.SortSafeList = []string{"created_at", "updated_at", "-created_at", "-updated_at"}
	booklists, metadata, err := h.service.ListUserBooklist(user.ID, qsInput.Filters)
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

func (h *Handler) listUserRequestsHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListUserRequests
	v := validator.New()
	user := h.contextGetUser(r)
	qs := r.URL.Query()
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Status = h.readString(qs, "status", "active")
	qsInput.Filters.Sort = h.readString(qs, "sort", "-datetime")
	qsInput.Filters.SortSafeList = []string{"datetime", "-datetime"}
	requests, metadata, err := h.service.ListUserRequests(user.ID, qsInput.Status, qsInput.Filters)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"requests": requests, "metadata": metadata}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

func (h *Handler) listUserBooksHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListUserBooks
	user := h.contextGetUser(r)
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "-created_at")
	qsInput.Filters.SortSafeList = []string{"created_at", "popularity", "size", "-created_at", "-popularity", "-size"}
	books, metadata, err := h.service.ListUserBooks(user.ID, qsInput.Filters)
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

func (h *Handler) listUserFavouriteBooksHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListUserFavouriteBooks
	user := h.contextGetUser(r)
	v := validator.New()
	qs := r.URL.Query()
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "-datetime")
	qsInput.Filters.SortSafeList = []string{"title", "size", "year", "datetime", "-title", "-size", "-year", "-datetime"}
	books, metadata, err := h.service.ListUserFavouriteBooks(user.ID, qsInput.Filters)
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

func (h *Handler) listUserDownloadsHandler(w http.ResponseWriter, r *http.Request) {
	var qsInput dto.QsListUserDownloads
	user := h.contextGetUser(r)
	v := validator.New()
	qs := r.URL.Query()
	qsInput.FromDate = h.readString(qs, "from_date", "2023-01-01")
	qsInput.ToDate = h.readString(qs, "to_date", time.Now().Format("2006-01-02"))
	qsInput.Filters.Page = h.readInt(qs, "page", 1, v)
	qsInput.Filters.PageSize = h.readInt(qs, "page_size", 10, v)
	qsInput.Filters.Sort = h.readString(qs, "sort", "-datetime")
	qsInput.Filters.SortSafeList = []string{"datetime", "-datetime"}
	books, metadata, err := h.service.ListUserDownloads(user.ID, qsInput.FromDate, qsInput.ToDate, qsInput.Filters)
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
