package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/emzola/bibliotheca/data"
)

type comments interface {
	CreateComment(comment *data.Comment) error
	GetComment(commentID int64) (*data.Comment, error)
	UpdateComment(comment *data.Comment) error
	DeleteComment(commentID int64) error
	GetAllComments(booklistID int64) ([]*data.Comment, error)
	CreateReply(comment *data.Comment) error
}

// CreateComment creates a comment record.
func (r *repository) CreateComment(comment *data.Comment) error {
	query := `
		INSERT INTO comments (booklist_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, version`
	args := []interface{}{comment.BooklistID, comment.UserID, comment.Content}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.db.QueryRowContext(ctx, query, args...).Scan(&comment.ID, &comment.CreatedAt, &comment.Version)
}

// GetComment retrieves a comment record.
func (r *repository) GetComment(commentID int64) (*data.Comment, error) {
	if commentID < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT comments.id, comments.parent_id, comments.booklist_id, comments.user_id, users.name, comments.created_at, comments.content, comments.version
		FROM comments
		INNER JOIN users on comments.user_id = users.id
		WHERE comments.id = $1`
	var comment data.Comment
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, commentID).Scan(
		&comment.ID,
		&comment.ParentID,
		&comment.BooklistID,
		&comment.UserID,
		&comment.UserName,
		&comment.CreatedAt,
		&comment.Content,
		&comment.Version,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}
	return &comment, nil
}

// UpdateComment updates a comment record.
func (r *repository) UpdateComment(comment *data.Comment) error {
	query := `
		UPDATE comments
		SET content = $1, version = version + 1
		WHERE id = $2 AND version = $3
		RETURNING version`
	args := []interface{}{comment.Content, comment.Version}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&comment.Version)
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

// DeleteComment deletes a comment record.
func (r *repository) DeleteComment(commentID int64) error {
	if commentID < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM comments
		WHERE id = $1`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	result, err := r.db.ExecContext(ctx, query, commentID)
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

// GetAllComments retrievs all comment records for a booklist.
func (r *repository) GetAllComments(booklistID int64) ([]*data.Comment, error) {
	query := `
	SELECT comments.id, COALESCE(comments.parent_id, 0), comments.booklist_id, comments.user_id, users.name, comments.created_at, comments.content, comments.version
	FROM comments 
	INNER JOIN users on comments.user_id = users.id
	WHERE comments.booklist_id = $1
	ORDER BY id DESC`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := r.db.QueryContext(ctx, query, booklistID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := []*data.Comment{}
	for rows.Next() {
		var comment data.Comment
		err := rows.Scan(
			&comment.ID,
			&comment.ParentID,
			&comment.BooklistID,
			&comment.UserID,
			&comment.UserName,
			&comment.CreatedAt,
			&comment.Content,
			&comment.Version,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &comment)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return comments, nil
}

// CreateReply creates a reply record for a comment.
func (r *repository) CreateReply(comment *data.Comment) error {
	query := `
		INSERT INTO comments (parent_id, booklist_id, user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`
	args := []interface{}{comment.ParentID, comment.BooklistID, comment.UserID, comment.Content}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return r.db.QueryRowContext(ctx, query, args...).Scan(&comment.ID, &comment.CreatedAt, &comment.Version)
}
