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

type users interface {
	RegisterUser(name string, email string, password string) (*data.User, error)
	ActivateUser(token string) (*data.User, error)
	ShowUser(userID int64) (*data.User, error)
	UpdateUser(ID int64, name *string, email *string) (*data.User, error)
	UpdateUserPassword(ID int64, old string, new string, confirm string) (*data.User, error)
	DeleteUser(ID int64) error
	ResetUserPassword(password string, token string) error
	GetUserForToken(tokenScope string, tokenPlaintext string) (*data.User, error)
	ListUserFavouriteBooklists(userID int64, filters data.Filters) ([]*data.Booklist, data.Metadata, error)
	ListUserBooklist(userID int64, filters data.Filters) ([]*data.Booklist, data.Metadata, error)
	ListUserRequests(userID int64, status string, filters data.Filters) ([]*data.Request, data.Metadata, error)
	ListUserBooks(userID int64, filters data.Filters) ([]*data.Book, data.Metadata, error)
	ListUserFavouriteBooks(userID int64, filters data.Filters) ([]*data.Book, data.Metadata, error)
	ListUserDownloads(userID int64, fromDate string, toDate string, filters data.Filters) ([]*data.Book, data.Metadata, error)
}

// RegisterUser service registers a new user.
func (s *service) RegisterUser(name string, email string, password string) (*data.User, error) {
	user := &data.User{
		Name:      name,
		Email:     email,
		Activated: false,
	}
	err := user.Password.Set(password)
	if err != nil {
		return nil, err
	}
	v := validator.New()
	if data.ValidateUser(v, user); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	err = s.repo.RegisterUser(user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateRecord):
			v.AddError("email", "a user with this email address already exists")
			ErrDuplicateRecord = s.failedValidation(v.Errors)
			return nil, ErrDuplicateRecord
		default:
			return nil, err
		}
	}
	// Generate a new activation token for user
	token, err := s.repo.CreateNewToken(user.ID, 3*24*time.Hour, data.ScopeActivation)
	if err != nil {
		return nil, err
	}
	// Send welcome email in a background goroutine to speed up response time
	s.background(func() {
		data := map[string]string{
			"userName":        strings.Split(user.Name, " ")[0],
			"activationToken": token.Plaintext,
		}
		mailer := mailer.New(s.config.SMTP.Host, s.config.SMTP.Port, s.config.SMTP.Username, s.config.SMTP.Password, s.config.SMTP.Sender)
		err := mailer.Send(user.Email, "user_welcome.tmpl", data)
		if err != nil {
			s.logger.PrintError(err, nil)
		}
	})
	return user, nil
}

// ActivateUser service activates a newly registered user.
func (s *service) ActivateUser(token string) (*data.User, error) {
	v := validator.New()
	if data.ValidateTokenPlaintext(v, token); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	// Retrieve user associated with the activation token. If the user record
	// isn't found, it means the token is invalid
	user, err := s.repo.GetUserForToken(data.ScopeActivation, token)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("token", "invalid or expired activation token")
			ErrFailedValidation = s.failedValidation(v.Errors)
			return nil, ErrFailedValidation
		default:
			return nil, err
		}
	}
	// Activate user
	user.Activated = true
	err = s.repo.UpdateUser(user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}
	// Delete all activation tokens for user
	err = s.repo.DeleteAllTokensForUser(data.ScopeActivation, user.ID)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// ShowUser service shows the details of a specific user.
func (s *service) ShowUser(userID int64) (*data.User, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return user, nil
}

// UpdateUser service updates the details of a specific user.
func (s *service) UpdateUser(ID int64, name *string, email *string) (*data.User, error) {
	user, err := s.repo.GetUserByID(ID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	if name != nil {
		user.Name = *name
	}
	if email != nil {
		user.Email = *email
	}
	v := validator.New()
	data.ValidateName(v, user.Name)
	data.ValidateEmail(v, user.Email)
	if !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	err = s.repo.UpdateUser(user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateRecord):
			v.AddError("email", "a user with this email address already exists")
			ErrFailedValidation = s.failedValidation(v.Errors)
			return nil, ErrFailedValidation
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return user, nil
}

// UpdateUserPassword service updates an authenticated user's password.
func (s *service) UpdateUserPassword(ID int64, old string, new string, confirm string) (*data.User, error) {
	// Validate password data
	v := validator.New()
	data.ValidatePasswordPlaintext(v, old)
	data.ValidatePasswordPlaintext(v, new)
	if !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	if new != confirm {
		return nil, ErrPasswordMismatch
	}
	// Retrieve user by ID
	user, err := s.repo.GetUserByID(ID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrInvalidCredentials
		default:
			return nil, err
		}
	}
	// Check whether old matches the old password hash equivalent in the User data.
	// If there is a match, proceed and update password. Otherwise return with error.
	match, err := user.Password.Matches(old)
	if err != nil {
		return nil, err
	}
	if !match {
		return nil, ErrInvalidCredentials
	}
	err = user.Password.Set(new)
	if err != nil {
		return nil, err
	}
	err = s.repo.UpdateUser(user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateRecord):
			v.AddError("email", "a user with this email address already exists")
			ErrFailedValidation = s.failedValidation(v.Errors)
			return nil, ErrFailedValidation
		case errors.Is(err, repository.ErrEditConflict):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}
	// Send password change notification email in a background goroutine to speed up response time
	s.background(func() {
		data := map[string]string{
			"userName": strings.Split(user.Name, " ")[0],
		}
		mailer := mailer.New(s.config.SMTP.Host, s.config.SMTP.Port, s.config.SMTP.Username, s.config.SMTP.Password, s.config.SMTP.Sender)
		err = mailer.Send(user.Email, "user_password_change.tmpl", data)
		if err != nil {
			s.logger.PrintError(err, nil)
		}
	})
	return user, nil
}

// DeleteUser service deletes a user.
func (s *service) DeleteUser(ID int64) error {
	err := s.repo.DeleteUser(ID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	return nil
}

// ResetUserPassword service resets a registered user's password.
func (s *service) ResetUserPassword(password string, token string) error {
	v := validator.New()
	if data.ValidateTokenPlaintext(v, token); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return ErrFailedValidation
	}
	// Retrieve user associated with password reset token
	user, err := s.repo.GetUserForToken(data.ScopePasswordReset, token)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("token", "invalid or expired token")
			ErrFailedValidation = s.failedValidation(v.Errors)
			return ErrFailedValidation
		default:
			return err
		}
	}
	// Set new passsword
	err = user.Password.Set(password)
	if err != nil {
		return err
	}
	err = s.repo.UpdateUser(user)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return ErrEditConflict
		default:
			return err
		}
	}
	// Delete all password reset tokens for user
	err = s.repo.DeleteAllTokensForUser(data.ScopePasswordReset, user.ID)
	if err != nil {
		return err
	}
	// Send password change notification email in a background goroutine to speed up response time
	s.background(func() {
		data := map[string]string{
			"userName": strings.Split(user.Name, " ")[0],
		}
		mailer := mailer.New(s.config.SMTP.Host, s.config.SMTP.Port, s.config.SMTP.Username, s.config.SMTP.Password, s.config.SMTP.Sender)
		err = mailer.Send(user.Email, "user_password_change.tmpl", data)
		if err != nil {
			s.logger.PrintError(err, nil)
		}
	})
	return nil
}

// GetUserForToken retrieves the user associated with a token.
func (s *service) GetUserForToken(tokenScope string, token string) (*data.User, error) {
	v := validator.New()
	user, err := s.repo.GetUserForToken(tokenScope, token)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			v.AddError("token", "invalid or expired token")
			ErrFailedValidation = s.failedValidation(v.Errors)
			return nil, ErrFailedValidation
		default:
			return nil, err
		}
	}
	return user, nil
}

// ListFavouriteBooklists retrieves a paginated list of user's favourite booklist.
func (s *service) ListUserFavouriteBooklists(userID int64, filters data.Filters) ([]*data.Booklist, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	booklists, metadata, err := s.repo.GetAllFavouriteBooklistsForUser(userID, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	for _, booklist := range booklists {
		booklist.Content.Books, booklist.Content.Metadata, err = s.repo.GetAllBooksForBooklist(booklist.ID, filters)
		if err != nil {
			return nil, data.Metadata{}, err
		}
	}
	return booklists, metadata, nil
}

// ListUserBooklist service retrieves a user's booklists.
func (s *service) ListUserBooklist(userID int64, filters data.Filters) ([]*data.Booklist, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	booklists, metadata, err := s.repo.GetAllBooklistsForUser(userID, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	for _, booklist := range booklists {
		booklist.Content.Books, booklist.Content.Metadata, err = s.repo.GetAllBooksForBooklist(booklist.ID, filters)
		if err != nil {
			return nil, data.Metadata{}, nil
		}
	}
	return booklists, metadata, nil
}

// ListUserRequests service retrieves a paginated list of all user requests.
// Records can be filtered and sorted.
func (s *service) ListUserRequests(userID int64, status string, filters data.Filters) ([]*data.Request, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	requests, metadata, err := s.repo.GetAllRequestsForUser(userID, status, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	return requests, metadata, nil
}

// ListUserBooks service retrieves a paginated list of all books for a user.
// List can be filtered and sorted.
func (s *service) ListUserBooks(userID int64, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	books, metadata, err := s.repo.GetAllBooksForUser(userID, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	return books, metadata, nil
}

// ListUserFavouriteBooks service retrieves a paginated list of all favourite books for a user.
// List can be filtered and sorted.
func (s *service) ListUserFavouriteBooks(userID int64, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	books, metadata, err := s.repo.GetAllFavouriteBooksForUser(userID, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	return books, metadata, nil
}

// ListUserDownloads service retrieves a paginated list of user downloads.
// List can be filtered and sorted.
func (s *service) ListUserDownloads(userID int64, fromDate string, toDate string, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	books, metadata, err := s.repo.GetAllDownloadsForUser(userID, fromDate, toDate, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	return books, metadata, nil
}
