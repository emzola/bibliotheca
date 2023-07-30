package service

import (
	"errors"

	"github.com/emzola/bibliotheca/data"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/repository"
)

type comments interface {
	CreateComment(userID int64, booklistID int64, username string, content string) (*data.Comment, error)
	GetComment(commentID int64) (*data.Comment, error)
	UpdateComment(commentID int64, content *string) (*data.Comment, error)
	DeleteComment(commentID int64) error
	ListComments(booklistID int64) ([]*data.Comment, error)
	CreateCommentReply(userID int64, booklistID int64, commentID int64, content string) (*data.Comment, error)
}

// CreateComment service creates a new comment.
func (s *service) CreateComment(userID int64, booklistID int64, username string, content string) (*data.Comment, error) {
	comment := &data.Comment{
		BooklistID: booklistID,
		UserID:     userID,
		UserName:   username,
		Content:    content,
	}
	v := validator.New()
	if data.ValidateComment(v, comment); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	err := s.repo.CreateComment(comment)
	if err != nil {
		return nil, err
	}
	return comment, nil
}

// GetComment service retrieves a comment.
func (s *service) GetComment(commentID int64) (*data.Comment, error) {
	comment, err := s.repo.GetComment(commentID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return comment, nil
}

// UpdateComment service updates the details of a comment.
func (s *service) UpdateComment(commentID int64, content *string) (*data.Comment, error) {
	comment, err := s.repo.GetComment(commentID)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrRecordNotFound):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	v := validator.New()
	if data.ValidateComment(v, comment); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	if content != nil {
		comment.Content = *content
	}
	err = s.repo.UpdateComment(comment)
	if err != nil {
		switch {
		case errors.Is(err, repository.ErrEditConflict):
			return nil, ErrEditConflict
		default:
			return nil, err
		}
	}
	return comment, nil
}

// DeleteComment service deletes a comment.
func (s *service) DeleteComment(commentID int64) error {
	err := s.repo.DeleteComment(commentID)
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

// ListComments service retrieves all comments for a booklist.
func (s *service) ListComments(booklistID int64) ([]*data.Comment, error) {
	comments, err := s.repo.GetAllComments(booklistID)
	if err != nil {
		return nil, err
	}
	return comments, nil
}

// CreateCommentReply service creates a creply for a comment.
func (s *service) CreateCommentReply(userID int64, booklistID int64, commentID int64, content string) (*data.Comment, error) {
	comment := &data.Comment{
		ParentID:   commentID,
		BooklistID: booklistID,
		UserID:     userID,
		Content:    content,
	}
	v := validator.New()
	if data.ValidateComment(v, comment); !v.Valid() {
		ErrFailedValidation = s.failedValidation(v.Errors)
		return nil, ErrFailedValidation
	}
	err := s.repo.CreateReply(comment)
	if err != nil {
		return nil, err
	}
	return comment, nil
}
