package data

import (
	"time"

	"github.com/emzola/bibliotheca/internal/validator"
)

// Rating defines the ratings for a book review.
type Rating struct {
	FiveStars  int64   `json:"fivestars"`
	FourStars  int64   `json:"fourstars"`
	ThreeStars int64   `json:"threestars"`
	TwoStars   int64   `json:"twostars"`
	OneStar    int64   `json:"onestar"`
	Average    float64 `json:"average"`
	Total      int64   `json:"total"`
}

// Review defines a book review.
type Review struct {
	ID        int64     `json:"id"`
	BookID    int64     `json:"book_id"`
	UserID    int64     `json:"user_id"`
	UserName  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	Rating    int8      `json:"rating"`
	Comment   string    `json:"comment"`
	Version   int32     `json:"-"`
}

func ValidateReview(v *validator.Validator, review *Review) {
	v.Check(review.Rating != 0, "rating", "must be provided")
	v.Check(review.Rating <= 5, "rating", "must not be greater than five")
}
