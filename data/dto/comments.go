package dto

// CreateCommentRequestBody defines the request body for CreateComment service.
type CreateCommentRequestBody struct {
	Content string `json:"content"`
}

// UpdateCommentRequestBody defines the request body for UpdateComment service.
type UpdateCommentRequestBody struct {
	Content *string `json:"content"`
}

// CreateCommentReplyRequestBody defines the request body for CreateCommentReply service.
type CreateCommentReplyRequestBody struct {
	Content string `json:"content"`
}
