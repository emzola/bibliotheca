package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

// The Rating struct contains the data fields for a book's review ratings.
type Rating struct {
	FiveStars  int64
	FourStars  int64
	ThreeStars int64
	TwoStars   int64
	OneStar    int64
	Average    float64
	Total      int64
}

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

func (m ReviewModel) GetAll(filters Filters) (*Rating, []*Review, Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), id, book_id, user_id, created_at, rating, comment, vote, version
		FROM reviews  
		ORDER BY %s %s, id ASC
		LIMIT $1 OFFSET $2`,
		filters.sortColumn(), filters.sortDirection())
	args := []interface{}{filters.limit(), filters.offset()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, nil, Metadata{}, err
	}
	defer rows.Close()
	ratings := &Rating{}
	sumRatings := int64(0)
	totalRecords := 0
	reviews := []*Review{}
	for rows.Next() {
		var review Review
		err := rows.Scan(
			&totalRecords,
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
			return nil, nil, Metadata{}, err
		}
		switch review.Rating {
		case 5:
			ratings.FiveStars++
		case 4:
			ratings.FourStars++
		case 3:
			ratings.ThreeStars++
		case 2:
			ratings.TwoStars++
		case 1:
			ratings.OneStar++
		}
		sumRatings += int64(review.Rating)
		reviews = append(reviews, &review)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, Metadata{}, err
	}
	avgRatingString := fmt.Sprintf("%.1f", float64(sumRatings)/float64(totalRecords))
	avgRating, err := strconv.ParseFloat(avgRatingString, 64)
	if err != nil {
		return nil, nil, Metadata{}, err
	}
	ratings.Average = avgRating
	ratings.Total = int64(totalRecords)
	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return ratings, reviews, metadata, nil
}

func (m ReviewModel) RecordExistsForUser(bookId, userId int64) bool {
	query := `
		SELECT id, book_id, user_id, created_at, rating, comment, vote, version
		FROM reviews
		WHERE book_id = $1 AND user_id = $2`
	args := []interface{}{bookId, userId}
	var review Review
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(
		&review.ID,
		&review.BookID,
		&review.UserID,
		&review.CreatedAt,
		&review.Rating,
		&review.Comment,
		&review.Vote,
		&review.Version,
	)
	return !errors.Is(err, sql.ErrNoRows)
}
