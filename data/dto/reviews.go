package dto

import "github.com/emzola/bibliotheca/data"

// CreateReviewRequestBody defines a request body for CreateReview service.
type CreateReviewRequestBody struct {
	Rating  int8   `json:"rating"`
	Comment string `json:"comment"`
}

// UpdateReviewRequestBody defines a request body for UpdateReview service.
type UpdateReviewRequestBody struct {
	Rating  *int8   `json:"rating"`
	Comment *string `json:"comment"`
}

// QsListReviews defines the query strings used for listing reviews.
type QsListReviews struct {
	Filters data.Filters
}
