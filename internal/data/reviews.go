package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

// The Review struct contains the data fields for a book review.
type Review struct {
	ID        int64     `json:"id"`
	BookID    int64     `json:"book_id"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	Rating    int8      `json:"rating"`
	Comment   string    `json:"comment"`
	Vote      int64     `json:"vote"`
	Version   int32     `json:"-"`
}

func ValidateReview(v *validator.Validator, review *Review) {
	v.Check(review.Rating != 0, "rating", "must be provided")
	v.Check(review.Rating <= 5, "rating", "must not be greater than five")
}

// The ReviewModel struct wraps a sql.DB connection pool for Review.
type ReviewModel struct {
	DB *sql.DB
}

func (m ReviewModel) Insert(review *Review) error {
	query := `
		INSERT INTO reviews (book_id, user_id, rating, comment)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`
	args := []interface{}{review.BookID, review.UserID, review.Rating, review.Comment}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&review.ID, &review.CreatedAt, &review.Version)
}

func (m ReviewModel) Get(id int64) (*Review, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, book_id, user_id, created_at, rating, comment, vote, version
		FROM reviews
		WHERE id = $1`
	var review Review
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&review.ID,
		&review.BookID,
		&review.UserID,
		&review.CreatedAt,
		&review.Rating,
		&review.Comment,
		&review.Vote,
		&review.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &review, nil
}

func (m ReviewModel) Update(review *Review) error {
	query := `
		UPDATE reviews
		SET rating = $1, comment = $2, vote = $3, version = version + 1
		WHERE id = $4 AND version = $5
		RETURNING version`
	args := []interface{}{review.Rating, review.Comment, review.Vote, review.ID, review.Version}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&review.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}
	return nil
}

func (m ReviewModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM reviews
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := m.DB.ExecContext(ctx, query, id)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}
