package handler

import (
	"errors"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/repository"
	"github.com/emzola/bibliotheca/service"
)

// CreateActivationToken godoc
// @Summary Create a new activation token
// @Description This endpoint creates a new activation token
// @Tags tokens
// @Accept  json
// @Produce json
// @Param body body dto.CreateActivationTokenRequestBody true "JSON payload required to create an activation token"
// @Success 202
// @Failure 400
// @Failure 422
// @Failure 500
// @Router /v1/tokens/activation [post]
func (h *Handler) createActivationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.CreateActivationTokenRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	err = h.service.CreateActivationToken(requestBody.Email)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusAccepted, envelope{"message": "an email will be sent to you containing activation instructions"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// CreatePasswordResetToken godoc
// @Summary Create a password reset token
// @Description This endpoint creates a password reset token
// @Tags tokens
// @Accept  json
// @Produce json
// @Param body body dto.CreatePasswordResetTokenRequestBody true "JSON payload required to create a password reset token"
// @Success 202
// @Failure 400
// @Failure 422
// @Failure 500
// @Router /v1/tokens/password-reset [post]
func (h *Handler) createPasswordResetTokenHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.CreatePasswordResetTokenRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	err = h.service.CreatePasswordResetToken(requestBody.Email)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusAccepted, envelope{"message": "an email will be sent to you containing password reset instructions"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// CreateAuthenticationToken godoc
// @Summary Login
// @Description This endpoint logs in a user by creating a user authentication token
// @Tags tokens
// @Accept  json
// @Produce json
// @Param body body dto.CreateAuthenticationTokenRequestBody true "JSON payload required to create an authentication token"
// @Success 201 {object} data.Token
// @Failure 400
// @Failure 401
// @Failure 422
// @Failure 500
// @Router /v1/tokens/authentication [post]
func (h *Handler) createAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	var requestBody dto.CreateAuthenticationTokenRequestBody
	err := h.decodeJSON(w, r, &requestBody)
	if err != nil {
		h.badRequestResponse(w, r, err)
		return
	}
	token, err := h.service.CreateAuthenticationToken(requestBody.Email, requestBody.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrFailedValidation):
			h.failedValidationResponse(w, r, err)
		case errors.Is(err, service.ErrInvalidCredentials):
			h.invalidCredentialsResponse(w, r)
		default:
			h.serverErrorResponse(w, r, err)
		}
		return
	}
	err = h.encodeJSON(w, http.StatusCreated, envelope{"authentication_token": token}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}

// DeleteAuthenticationToken godoc
// @Summary Logout
// @Description This endpoint logs out a user by deleteting a user authentication token
// @Tags tokens
// @Accept  json
// @Produce json
// @Param token header string true "Bearer token"
// @Success 200
// @Failure 500
// @Router /v1/tokens/authentication [delete]
func (h *Handler) deleteAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	user := h.contextGetUser(r)
	err := h.service.DeleteAuthenticationToken(user.ID)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
	err = h.encodeJSON(w, http.StatusOK, envelope{"message": "authentication token successfully deleted"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
