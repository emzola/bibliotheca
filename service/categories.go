package service

import (
	"errors"

	"github.com/emzola/bibliotheca/data"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/repository"
)

type categories interface {
	GetCategory(categoryID int64) (*data.Category, error)
	ShowCategory(categoryID int64, filters data.Filters) ([]*data.Book, data.Metadata, error)
	ListCategories() ([]*data.Category, error)
}

// ListCategories service retrieves a list of categories.
func (s *service) ListCategories() ([]*data.Category, error) {
	categories, err := s.repo.GetAllCategories()
	if err != nil {
		return nil, err
	}
	return categories, nil
}

// GetCategory service retrieves a category record.
func (s *service) GetCategory(categoryID int64) (*data.Category, error) {
	category, err := s.repo.GetCategory(categoryID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return category, nil
}

// ShowCategory displays details of a specifi category and its book/metadata content.
func (s *service) ShowCategory(categoryID int64, filters data.Filters) ([]*data.Book, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, data.Metadata{}, ErrFailedValidation
	}
	books, metadata, err := s.repo.GetAllBooksForCategory(categoryID, filters)
	if err != nil {
		return nil, data.Metadata{}, err
	}
	return books, metadata, nil
}
