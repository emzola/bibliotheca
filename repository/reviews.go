package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/emzola/bibliotheca/data"
)

type reviews interface {
	CreateReview(review *data.Review) error
	GetReview(reviewID int64) (*data.Review, error)
	UpdateReview(review *data.Review) error
	DeleteReview(reviewID int64) error
	ReviewExistsForUser(userID int64, bookID int64) bool
	GetReviewRatings() (data.Rating, error)
	GetAllReviews(filters data.Filters) (data.Rating, []*data.Review, data.Metadata, error)
}

// CreateReview creates a review record for book.
func (r *repository) CreateReview(review *data.Review) error {
	query := `
		INSERT INTO reviews (book_id, user_id, rating, comment)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`
	args := []interface{}{review.BookID, review.UserID, review.Rating, review.Comment}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.db.QueryRowContext(ctx, query, args...).Scan(&review.ID, &review.CreatedAt, &review.Version)
}

// ReviewExistsForUser checks whether a review record already exists for user.
func (r *repository) ReviewExistsForUser(userID int64, bookID int64) bool {
	query := `
		SELECT id, book_id, user_id, created_at, rating, comment, version
		FROM reviews
		WHERE book_id = $1 AND user_id = $2`
	args := []interface{}{bookID, userID}
	var review data.Review
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, args...).Scan(
		&review.ID,
		&review.BookID,
		&review.UserID,
		&review.CreatedAt,
		&review.Rating,
		&review.Comment,
		&review.Version,
	)
	return !errors.Is(err, sql.ErrNoRows)
}

// GetReview retrieves a review record for a specific book.
func (r *repository) GetReview(reviewID int64) (*data.Review, error) {
	if reviewID < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT reviews.id, reviews.book_id, reviews.user_id, users.name, reviews.created_at, reviews.rating, reviews.comment, reviews.version
		FROM reviews
		INNER JOIN users ON reviews.user_id = users.id
		WHERE reviews.id = $1`
	var review data.Review
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, reviewID).Scan(
		&review.ID,
		&review.BookID,
		&review.UserID,
		&review.UserName,
		&review.CreatedAt,
		&review.Rating,
		&review.Comment,
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

// UpdateReview updates a review record.
func (r *repository) UpdateReview(review *data.Review) error {
	query := `
		UPDATE reviews
		SET rating = $1, comment = $2, version = version + 1
		WHERE id = $3 AND version = $4
		RETURNING version`
	args := []interface{}{review.Rating, review.Comment, review.ID, review.Version}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&review.Version)
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

// DeleteReview deletes a review record.
func (r *repository) DeleteReview(reviewID int64) error {
	if reviewID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM reviews
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := r.db.ExecContext(ctx, query, reviewID)
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

// GetReviewRatings retrieves the ratings for a review record.
func (r *repository) GetReviewRatings() (data.Rating, error) {
	query := `
		SELECT id, rating
		FROM reviews  
		ORDER BY id ASC`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return data.Rating{}, err
	}
	defer rows.Close()
	ratings := data.Rating{}
	sumRatings := int64(0)
	totalRecords := 0
	for rows.Next() {
		var review data.Review
		err := rows.Scan(
			&review.ID,
			&review.Rating,
		)
		if err != nil {
			return data.Rating{}, err
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
		totalRecords++
	}
	if err = rows.Err(); err != nil {
		return data.Rating{}, err
	}
	avgRatingString := fmt.Sprintf("%.1f", float64(sumRatings)/float64(totalRecords))
	avgRating, err := strconv.ParseFloat(avgRatingString, 64)
	if err != nil {
		return data.Rating{}, err
	}
	// Because averageRating calculation could result in NAN,
	// check that it isn't NAN before updating rating's average.
	// This ensures that JSON encoding works without NAN error
	if !math.IsNaN(avgRating) {
		ratings.Average = avgRating
	}
	ratings.Total = int64(totalRecords)
	return ratings, nil
}

// GetAllReviews retrieves a paginated list of all review records (including it's ratings).
// Records can be filtered and sorted.
func (r *repository) GetAllReviews(filters data.Filters) (data.Rating, []*data.Review, data.Metadata, error) {
	query := fmt.Sprintf(`
		SELECT count(*) OVER(), reviews.id, reviews.book_id, reviews.user_id, users.name, reviews.created_at, reviews.rating, reviews.comment, reviews.version
		FROM reviews  
		INNER JOIN users ON reviews.user_id = users.id
		ORDER BY %s %s, id ASC
		LIMIT $1 OFFSET $2`,
		filters.SortColumn(), filters.SortDirection())
	args := []interface{}{filters.Limit(), filters.Offset()}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return data.Rating{}, nil, data.Metadata{}, err
	}
	defer rows.Close()
	ratings := data.Rating{}
	sumRatings := int64(0)
	totalRecords := 0
	reviews := []*data.Review{}
	for rows.Next() {
		var review data.Review
		err := rows.Scan(
			&totalRecords,
			&review.ID,
			&review.BookID,
			&review.UserID,
			&review.UserName,
			&review.CreatedAt,
			&review.Rating,
			&review.Comment,
			&review.Version,
		)
		if err != nil {
			return data.Rating{}, nil, data.Metadata{}, err
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
		return data.Rating{}, nil, data.Metadata{}, err
	}
	avgRatingString := fmt.Sprintf("%.1f", float64(sumRatings)/float64(totalRecords))
	avgRating, err := strconv.ParseFloat(avgRatingString, 64)
	if err != nil {
		return data.Rating{}, nil, data.Metadata{}, err
	}
	// Because averageRating calculation could result in NAN,
	// check that it isn't NAN before updating rating's average.
	// This ensures that JSON encoding works without NAN error
	if !math.IsNaN(avgRating) {
		ratings.Average = avgRating
	}
	ratings.Total = int64(totalRecords)
	metadata := data.CalculateMetadata(totalRecords, filters.Page, filters.PageSize)
	return ratings, reviews, metadata, nil
}
