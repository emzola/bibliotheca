package handler

import (
	"errors"
	"net/http"

	"github.com/emzola/bibliotheca/data/dto"
	"github.com/emzola/bibliotheca/repository"
	"github.com/emzola/bibliotheca/service"
)

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

func (h *Handler) deleteAuthenticationTokenHandler(w http.ResponseWriter, r *http.Request) {
	user := h.contextGetUser(r)
	err := h.service.DeleteAuthenticationToken(user.ID)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
	err = h.encodeJSON(w, http.StatusCreated, envelope{"message": "authentication token successfully deleted"}, nil)
	if err != nil {
		h.serverErrorResponse(w, r, err)
	}
}
