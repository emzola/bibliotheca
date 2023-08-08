package service

import (
	"errors"
	"strings"
	"time"

	"github.com/emzola/bibliotheca/data"
	"github.com/emzola/bibliotheca/internal/mailer"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/repository"
)

type tokens interface {
	CreateActivationToken(email string) error
	CreateAuthenticationToken(email string, password string) (*data.Token, error)
	DeleteAuthenticationToken(userID int64) error
	CreatePasswordResetToken(email string) error
}

// CreateActivationToken service creates a new activation token.
func (s *service) CreateActivationToken(email string) error {
	v := validator.New()
	if data.ValidateEmail(v, email); !v.Valid() {
		return ErrFailedValidation
	}
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("email", "no matching email address found")
			ErrFailedValidation = s.failedValidation(v.Errors)
			return ErrFailedValidation
		default:
			return err
		}
	}
	// if user is already activated, no need to proceed
	if user.Activated {
		v.AddError("email", "user with this email has already been activated")
		ErrFailedValidation = s.failedValidation(v.Errors)
		return ErrFailedValidation
	}
	token, err := s.repo.CreateNewToken(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		return err
	}
	// Send new activation token via email
	s.background(func() {
		data := map[string]string{
			"userName":        strings.Split(user.Name, " ")[0],
			"activationToken": token.Plaintext,
		}
		mailer := mailer.New(s.config.SMTP.Host, s.config.SMTP.Port, s.config.SMTP.Username, s.config.SMTP.Password, s.config.SMTP.Sender)
		err := mailer.Send(user.Email, "token_activation.tmpl", data)
		if err != nil {
			s.logger.PrintError(err, nil)
		}
	})
	return nil
}

// CreateAuthenticationToken service creates a new authentication token.
func (s *service) CreateAuthenticationToken(email string, password string) (*data.Token, error) {
	v := validator.New()
	data.ValidateEmail(v, email)
	data.ValidatePasswordPlaintext(v, password)
	if !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrInvalidCredentials
		default:
			return nil, err
		}
	}
	match, err := user.Password.Matches(password)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, ErrInvalidCredentials
	}
	token, err := s.repo.CreateNewToken(user.ID, 24*time.Hour, data.ScopeAuthentication)
	if err != nil {
		return nil, err
	}
	return token, nil
}

// DeleteAuthenticationToken deletes all authentication tokens for a user.
func (s *service) DeleteAuthenticationToken(userID int64) error {
	// Delete all authentication tokens for user
	err := s.repo.DeleteAllTokensForUser(data.ScopeAuthentication, userID)
	if err != nil {
		return err
	}
	return nil
}

// CreatePasswordResetToken service creates a new password reset token.
func (s *service) CreatePasswordResetToken(email string) error {
	v := validator.New()
	if data.ValidateEmail(v, email); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return ErrFailedValidation
	}
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("email", "no matching email address found")
			ErrFailedValidation = s.failedValidation(v.Errors)
			return ErrFailedValidation
		default:
			return err
		}
	}
	// if user isn't activated, no need to proceed
	if !user.Activated {
		v.AddError("email", "user account must be activated")
		ErrFailedValidation = s.failedValidation(v.Errors)
		return ErrFailedValidation
	}
	token, err := s.repo.CreateNewToken(user.ID, 30*time.Minute, data.ScopePasswordReset)
	if err != nil {
		return err
	}
	// Send password reset token via email
	s.background(func() {
		data := map[string]string{
			"userName":           strings.Split(user.Name, " ")[0],
			"passwordResetToken": token.Plaintext,
		}
		mailer := mailer.New(s.config.SMTP.Host, s.config.SMTP.Port, s.config.SMTP.Username, s.config.SMTP.Password, s.config.SMTP.Sender)
		err := mailer.Send(user.Email, "token_password_reset.tmpl", data)
		if err != nil {
			s.logger.PrintError(err, nil)
		}
	})
	return nil
}
