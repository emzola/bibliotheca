package data

import (
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

// Comment defines a booklist comment.
type Comment struct {
	ID         int64     `json:"id"`
	ParentID   int64     `json:"parent_id"`
	BooklistID int64     `json:"booklist_id"`
	UserID     int64     `json:"user_id"`
	UserName   string    `json:"username"`
	CreatedAt  time.Time `json:"created_at"`
	Content    string    `json:"content"`
	Version    int32     `json:"-"`
}

func ValidateComment(v *validator.Validator, comment *Comment) {
	v.Check(comment.Content != "", "content", "must be provided")
}
