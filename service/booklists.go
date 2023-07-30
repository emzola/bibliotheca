package service

import (
	"errors"

	"github.com/emzola/bibliotheca/data"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/repository"
)

type booklists interface {
	CreateBooklist(name string, description string, private bool, userID int64, username string) (*data.Booklist, error)
	GetBooklist(booklistID int64, filters data.Filters) (*data.Booklist, error)
	UpdateBooklist(booklistID int64, name *string, description *string, private *bool) (*data.Booklist, error)
	DeleteBooklist(booklistID int64) error
	ListBooklist(search string, filters data.Filters) ([]*data.Booklist, data.Metadata, error)
	FavouriteBooklist(userID int64, booklistID int64) error
	DeleteFavouriteBooklist(userID int64, booklistID int64) error
	AddBookToBooklist(bookID int64, booklistID int64) error
	DeleteBookFromBooklist(bookID int64, booklistID int64) error
	FindBooksForBooklist(search string, filters data.Filters) ([]*data.Book, data.Metadata, error)
	ShowBookInBooklist(bookID int64) (*data.Book, error)
}

// CreateBooklist service creates a booklist.
func (s *service) CreateBooklist(name string, description string, private bool, userID int64, username string) (*data.Booklist, error) {
	booklist := &data.Booklist{
		UserID:      userID,
		CreatorName: username,
		Name:        name,
		Description: description,
		Private:     private,
	}
	v := validator.New()
	if data.ValidateBooklist(v, booklist); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return booklist, ErrFailedValidation
	}
	err := s.repo.CreateBooklist(booklist)
	if err != nil {
		return booklist, err
	}
	return booklist, nil
}

func (s *service) GetBooklist(booklistID int64, filters data.Filters) (*data.Booklist, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	booklist, err := s.repo.GetBooklist(booklistID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	booklist.Content.Books, booklist.Content.Metadata, err = s.repo.GetAllBooksForBooklist(booklistID, filters)
	if err != nil {
		return nil, err
	}
	return booklist, nil
}

// UpdateBooklist service updates a booklist.
func (s *service) UpdateBooklist(booklistID int64, name *string, description *string, private *bool) (*data.Booklist, error) {
	booklist, err := s.repo.GetBooklist(booklistID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	if name != nil {
		booklist.Name = *name
	}
	if description != nil {
		booklist.Description = *description
	}
	if private != nil {
		booklist.Private = *private
	}
	v := validator.New()
	if data.ValidateBooklist(v, booklist); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	err = s.repo.UpdateBooklist(booklist)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}
	return booklist, nil
}

// ListBooklist service retrieves a paginated list of all booklists.
func (s *service) ListBooklist(search string, filters data.Filters) ([]*data.Booklist, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	booklists, metadata, err := s.repo.GetAllBooklists(search, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	// Loop through each booklists and add books to the each one
	for _, booklist := range booklists {
		booklist.Content.Books, booklist.Content.Metadata, err = s.repo.GetAllBooksForBooklist(booklist.ID, filters)
		if err != nil {
			return nil, data.Metadata{}, err
		}
	}
	return booklists, metadata, nil
}

// DeleteBooklist deletes a booklist.
func (s *service) DeleteBooklist(booklistID int64) error {
	err := s.repo.DeleteBooklist(booklistID)
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

// FavouriteBooklist service marks a booklist as favourite.
func (s *service) FavouriteBooklist(userID int64, booklistID int64) error {
	err := s.repo.FavouriteBooklist(userID, booklistID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrDuplicateRecord):
			return ErrDuplicateRecord
		default:
			return err
		}
	}
	return nil
}

// DeleteFavouriteBooklist unmarks a booklist as favourite.
func (s *service) DeleteFavouriteBooklist(userID int64, booklistID int64) error {
	err := s.repo.DeleteFavouriteBooklist(userID, booklistID)
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

// AddBookToBooklist service adds a book to a booklist.
func (s *service) AddBookToBooklist(bookID int64, booklistID int64) error {
	booklist, err := s.repo.GetBooklist(booklistID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	err = s.repo.AddBookToBooklist(booklist.ID, bookID)
	if err != nil {
		return err
	}
	err = s.repo.UpdateBooklist(booklist)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

// DeleteBookFromBooklist service deletes a book from a booklist.
func (s *service) DeleteBookFromBooklist(bookID int64, booklistID int64) error {
	err := s.repo.DeleteBookFromBooklist(booklistID, bookID)
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

// FindBooksForBooklist service finds books for booklist.
func (s *service) FindBooksForBooklist(search string, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	books, metadata, err := s.repo.SearchBooksInBooklist(search, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	return books, metadata, nil
}

// ShowBookInBooklist service shows the details of a specifi book when searched for in a booklist.
func (s *service) ShowBookInBooklist(bookID int64) (*data.Book, error) {
	book, err := s.repo.GetBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return book, nil
}
