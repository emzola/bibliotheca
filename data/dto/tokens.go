package dto

// CreateActivationTokenRequestBody defines a request body for CreateActivationToken service.
type CreateActivationTokenRequestBody struct {
	Email string `json:"email"`
}

// CreatePasswordResetTokenRequestBody defines a request body for CreatePasswordResetToken service.
type CreatePasswordResetTokenRequestBody struct {
	Email string `json:"email"`
}

// createAuthenticationTokenRequestBody defines a request body for CreatePasswordResetToken service.
type CreateAuthenticationTokenRequestBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
