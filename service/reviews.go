package service

import (
	"errors"

	"github.com/emzola/bibliotheca/data"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/repository"
)

type reviews interface {
	CreateReview(userID int64, bookID int64, username string, rating int8, comment string) (*data.Review, error)
	GetReview(reviewID int64) (*data.Review, error)
	UpdateReview(reviewID int64, bookID int64, rating *int8, comment *string) (*data.Review, error)
	DeleteReview(reviewID int64, bookID int64) error
	ListReviews(filters data.Filters) (data.Rating, []*data.Review, data.Metadata, error)
}

func (s *service) CreateReview(userID int64, bookID int64, username string, rating int8, comment string) (*data.Review, error) {
	// First check whether a review from user already exists. If it does, do not process further request
	exists := s.repo.ReviewExistsForUser(bookID, userID)
	if exists {
		return nil, ErrDuplicateRecord
	}
	// From this point, retrieve the book for which a review is to be left for
	// since user does not have a review
	book, err := s.repo.GetBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	// Create review as usual
	review := &data.Review{
		BookID:   bookID,
		UserID:   userID,
		UserName: username,
		Rating:   rating,
		Comment:  comment,
	}
	v := validator.New()
	if data.ValidateReview(v, review); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	err = s.repo.CreateReview(review)
	if err != nil {
		return nil, err
	}
	// Get ratings and Update the popularity field of a book
	ratings, err := s.repo.GetReviewRatings()
	if err != nil {
		return nil, err
	}
	book.Popularity = ratings.Average
	err = s.repo.UpdateBook(book)
	if err != nil {
		return nil, err
	}
	return review, nil
}

// ShowReview service retrieves the details of a book.
func (s *service) GetReview(reviewID int64) (*data.Review, error) {
	review, err := s.repo.GetReview(reviewID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return review, nil
}

func (s *service) UpdateReview(reviewID int64, bookID int64, rating *int8, comment *string) (*data.Review, error) {
	review, err := s.repo.GetReview(reviewID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	if rating != nil {
		review.Rating = *rating
	}
	if comment != nil {
		review.Comment = *comment
	}
	v := validator.New()
	if data.ValidateReview(v, review); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	err = s.repo.UpdateReview(review)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}
	// Get ratings and update the popularity field of a book
	ratings, err := s.repo.GetReviewRatings()
	if err != nil {
		return nil, err
	}
	book, err := s.repo.GetBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	book.Popularity = ratings.Average
	err = s.repo.UpdateBook(book)
	if err != nil {
		return nil, err
	}
	return review, nil
}

func (s *service) DeleteReview(reviewID int64, bookID int64) error {
	err := s.repo.DeleteReview(reviewID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	// Also get ratings and update the popularity field of a book
	book, err := s.repo.GetBook(bookID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return ErrRecordNotFound
		default:
			return err
		}
	}
	ratings, err := s.repo.GetReviewRatings()
	if err != nil {
		return err
	}
	book.Popularity = ratings.Average
	err = s.repo.UpdateBook(book)
	if err != nil {
		return err
	}
	return nil
}

// ListReviews service retrieves a paginated list of all reviews for a book.
func (s *service) ListReviews(filters data.Filters) (data.Rating, []*data.Review, data.Metadata, error) {
	v := validator.New()
	if data.ValidateFilters(v, filters); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return data.Rating{}, nil, data.Metadata{}, ErrFailedValidation
	}
	ratings, reviews, metadata, err := s.repo.GetAllReviews(filters)
	if err != nil {
		return data.Rating{}, nil, data.Metadata{}, err
	}
	return ratings, reviews, metadata, nil
}
