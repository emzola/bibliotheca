package dto

import "github.com/emzola/bibliotheca/data"

// RegisterUserRequestBody defines a request body for RegisterUser service.
type RegisterUserRequestBody struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// activateUserRequestBody defines a request body for ActivateUser service.
type ActivateUserRequestBody struct {
	TokenPlaintext string `json:"token"`
}

// ResetUserPasswordRequestBody defines a request body for ResetUserPassword service.
type ResetUserPasswordRequestBody struct {
	Password       string `json:"password"`
	TokenPlaintext string `json:"token"`
}

// UpdateUserRequestBody defines a request body for UpdateUser service.
type UpdateUserRequestBody struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

// UpdateUserPasswordRequestBody defines a request body for UpdateUserPassword service.
type UpdateUserPasswordRequestBody struct {
	OldPassword        string `json:"old_password"`
	NewPassword        string `json:"new_password"`
	ConfirmNewPassword string `json:"confirm_new_password"`
}

// QsListUserRequests defines query strings for QsListUserRequests service.
type QsListUserRequests struct {
	Status  string
	Filters data.Filters
}

// QsListUserFavouriteBooklists defines query strings for ListUserFavouriteBooklist service.
type QsListUserFavouriteBooklists struct {
	Filters data.Filters
}

// QsListUserBooklists defines query strings for ListUserBooklists service.
type QsListUserBooklists struct {
	Filters data.Filters
}

// QsListUserFavouriteBooks defines query strings for ListUserFavouriteBooks service.
type QsListUserFavouriteBooks struct {
	Filters data.Filters
}

// QsListUserDownloads defines query strings for ListUserDownloads service.
type QsListUserDownloads struct {
	FromDate string
	ToDate   string
	Filters  data.Filters
}
