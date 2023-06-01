package data

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

// The Comment struct contains the data fields for a booklist comment.
type Comment struct {
	ID         int64     `json:"id"`
	ParentID   int64     `json:"parent_id"`
	BooklistID int64     `json:"booklist_id"`
	UserID     int64     `json:"user_id"`
	CreatedAt  time.Time `json:"created_at"`
	Content    string    `json:"content"`
	Version    int32     `json:"-"`
}

func ValidateComment(v *validator.Validator, comment *Comment) {
	v.Check(comment.Content != "", "content", "must be provided")
}

// The CommentModel struct wraps a sql.DB connection pool for Comment.
type CommentModel struct {
	DB *sql.DB
}

func (m CommentModel) Insert(comment *Comment) error {
	query := `
		INSERT INTO comments (booklist_id, user_id, content)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, version`
	args := []interface{}{comment.BooklistID, comment.UserID, comment.Content}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&comment.ID, &comment.CreatedAt, &comment.Version)
}

func (m CommentModel) Get(id int64) (*Comment, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}
	query := `
		SELECT id, parent_id, booklist_id, user_id, created_at, content, version
		FROM comments
		WHERE id = $1`
	var comment Comment
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, id).Scan(
		&comment.ID,
		&comment.ParentID,
		&comment.BooklistID,
		&comment.UserID,
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

func (m CommentModel) Update(comment *Comment) error {
	query := `
		UPDATE comments
		SET content = $1, version = version + 1
		WHERE id = $2 AND version = $3
		RETURNING version`
	args := []interface{}{comment.Content, comment.Version}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := m.DB.QueryRowContext(ctx, query, args...).Scan(&comment.Version)
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

func (m CommentModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}
	query := `
		DELETE FROM comments
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

func (m CommentModel) GetAll(id int64) ([]*Comment, error) {
	query := `
	SELECT id, COALESCE(parent_id, 0), booklist_id, user_id, created_at, content, version
	FROM comments 
	WHERE booklist_id = $1
	`
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	rows, err := m.DB.QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	comments := []*Comment{}
	for rows.Next() {
		var comment Comment
		err := rows.Scan(
			&comment.ID,
			&comment.ParentID,
			&comment.BooklistID,
			&comment.UserID,
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

func (m CommentModel) InsertReply(comment *Comment) error {
	query := `
		INSERT INTO comments (parent_id, booklist_id, user_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, version`
	args := []interface{}{comment.ParentID, comment.BooklistID, comment.UserID, comment.Content}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return m.DB.QueryRowContext(ctx, query, args...).Scan(&comment.ID, &comment.CreatedAt, &comment.Version)
}
