package handler

import (
	"context"
	"net/http"

	"github.com/emzola/bibliotheca/data"
)

// Type contextKey is a custom contextKey type, with the underlying type string.
// This is necessary to prevent name collisions with external packages.
type contextKey string

// Convert the string "user" to a contextKey type and assign it to the userContextKey
// constant. This constant is used as the key for getting and setting user information
// in the request context.
const userContextKey = contextKey("user")

// contextSetUser returns a new copy of the request with the provided User struct
// added to the context. Note that we use our userContextKey constant as the key.
func (h *Handler) contextSetUser(r *http.Request, user *data.User) *http.Request {
	ctx := context.WithValue(r.Context(), userContextKey, user)
	return r.WithContext(ctx)
}

// contextGetUser retrieves the User struct from the request context. The only
// time that we'll use this helper is when we logically expect there to be User struct
// value in the context, and if it doesn't exist it will firmly be an 'unexpected' error.
func (h *Handler) contextGetUser(r *http.Request) *data.User {
	user, ok := r.Context().Value(userContextKey).(*data.User)
	if !ok {
		panic("missing user value in request context")
	}
	return user
}
