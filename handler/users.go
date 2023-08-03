package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
)

// RegisterUser godoc
// @Summary Register a user account
// @Description This endpoint registers a new user
// @Tags users
// @Accept  json
// @Produce json
// @Param body body dto.RegisterUserRequestBody true "JSON payload required to create a new user"
// @Success 202 {object} data.User
// @Failure 400
// @Failure 422
// @Failure 500
// @Router /v1/users [post]
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

// ActivateUser godoc
// @Summary Activate a user account
// @Description This endpoint activates a newly registered user
// @Tags users
// @Accept  json
// @Produce json
// @Param body body dto.ActivateUserRequestBody true "JSON payload required to activate a user"
// @Success 202 {object} data.User
// @Failure 400
// @Failure 409
// @Failure 422
// @Failure 500
// @Router /v1/users/activated [put]
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

// ShowUser godoc
// @Summary Show details of a logged in user
// @Description This endpoint shows the details of a logged in user
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Success 200 {object} data.User
// @Failure 404
// @Failure 500
// @Router /v1/users/profile [get]
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

// UpdateUser godoc
// @Summary Update a user
// @Description This endpoint updates a user
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param body body dto.UpdateUserRequestBody true "JSON payload required to update a user"
// @Success 200 {object} data.User
// @Failure 400
// @Failure 404
// @Failure 422
// @Failure 500
// @Router /v1/users/activated [patch]
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

// UpdateUserPassword godoc
// @Summary Update a user's password
// @Description This endpoint updates a logged in user's password
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param body body dto.UpdateUserPasswordRequestBody true "JSON payload required to update a user's password"
// @Success 202 {object} data.User
// @Failure 400
// @Failure 401
// @Failure 409
// @Failure 422
// @Failure 500
// @Router /v1/users/profile [put]
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

// ResetUserPassword godoc
// @Summary Reset password
// @Description This endpoint resets a logged out user's password
// @Tags users
// @Accept  json
// @Produce json
// @Param body body dto.ResetUserPasswordRequestBody true "JSON payload required to reset a non logged in user's password"
// @Success 200
// @Failure 400
// @Failure 409
// @Failure 422
// @Failure 500
// @Router /v1/users/password [put]
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

// DeleteUser godoc
// @Summary Delete user
// @Description This endpoint deletes a user's account
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Success 200
// @Failure 404
// @Failure 500
// @Router /v1/users/profile [delete]
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

// ListUserFavouriteBooklists godoc
// @Summary List all user's favourite booklists
// @Description This endpoint lists all user's favourite booklists
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param sort query string false "Sort by ascending or descending order. Asc: datetime, created_at, updated_at. Desc: -datetime, -created_at, -updated_at"
// @Success 200 {array} data.Booklist
// @Failure 422
// @Failure 500
// @Router /v1/users/booklists/favourite [get]
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

// ListUserBooklists godoc
// @Summary List all user's booklists
// @Description This endpoint lists all user's booklists
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param sort query string false "Sort by ascending or descending order. Asc: id, created_at, updated_at. Desc: -id, -created_at, -updated_at"
// @Success 200 {array} data.Booklist
// @Failure 422
// @Failure 500
// @Router /v1/users/booklists [get]
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

// listUserRequests godoc
// @Summary List all user's book requests
// @Description This endpoint lists all user's book requests
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param status query int false "Query string param for book status (options: active, expired, completed)"
// @Param sort query string false "Sort by ascending or descending order. Asc: datetime. Desc: -datetime"
// @Success 200 {array} data.Request
// @Failure 422
// @Failure 500
// @Router /v1/users/requests [get]
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

// listUserBooks godoc
// @Summary List all user's books
// @Description This endpoint lists all user's books
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param status query int false "Query string param for book status (options: active, expired, completed)"
// @Param sort query string false "Sort by ascending or descending order. Asc: created_at, popularity, size. Desc: -created_at, -popularity, -size"
// @Success 200 {array} data.Book
// @Failure 422
// @Failure 500
// @Router /v1/users/books [get]
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

// listUserFavouriteBooks godoc
// @Summary List all user's favourite books
// @Description This endpoint lists all user's favourite books
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param status query int false "Query string param for book status (options: active, expired, completed)"
// @Param sort query string false "Sort by ascending or descending order. Asc: title, size, year, datetime. Desc: -title, -size, -year, -datetime"
// @Success 200 {array} data.Book
// @Failure 422
// @Failure 500
// @Router /v1/users/books/favourite [get]
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

// listUserDownloads godoc
// @Summary List all user's downloads
// @Description This endpoint lists all user's downloads
// @Tags users
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Param from_date query string false "Query string param to filter by date"
// @Param to_date query string false "Query string param to filter by date"
// @Param page query int false "Query string param for pagination (min 1)"
// @Param page_size query int false "Query string param for pagination (max 100)"
// @Param status query int false "Query string param for book status (options: active, expired, completed)"
// @Param sort query string false "Sort by ascending or descending order. Asc: datetime. Desc: -datetime"
// @Success 200 {array} data.Book
// @Failure 422
// @Failure 500
// @Router /v1/users/downloads [get]
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
