package main

import (
	"errors"
	"net/http"
	"strings"

	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/jellydator/ttlcache/v3"
)

// authenticate middleware authenticates users. It returns an authenticated or anonymous user.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")
		authorizationHeader := r.Header.Get("Authorization")
		if authorizationHeader == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}
		headerParts := strings.Split(authorizationHeader, " ")
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}
		token := headerParts[1]
		v := validator.New()
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}
		user, err := app.models.Users.GetForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, data.ErrRecordNotFound):
				app.invalidAuthenticationTokenResponse(w, r)
			default:
				app.serverErrorResponse(w, r, err)
			}
			return
		}
		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// requireAuthenticatedUser middleware checks that a user is not anonymous.
func (app *application) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if user.IsAnonymous() {
			app.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser middleware checks that a user is both authenticated and activated.
func (app *application) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if !user.Activated {
			app.inactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return app.requireAuthenticatedUser(fn)
}

// requireBookOwnerPermission middleware checks that a user is authenticated, activated and is the owner of the book.
func (app *application) requireBookOwnerPermission(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from request context
		user := app.contextGetUser(r)
		// Check whether book's UserID field is found in cache
		cache := app.cache
		bookUserID := cache.Get("bookUserID")
		if bookUserID == nil {
			// If book's UserID field is not found, fetch it from the database and set to cache
			id, err := app.readIDParam(r, "bookId")
			if err != nil || id < 1 {
				app.notFoundResponse(w, r)
				return
			}
			book, err := app.models.Books.Get(id)
			if err != nil {
				switch {
				case errors.Is(err, data.ErrRecordNotFound):
					app.notFoundResponse(w, r)
				default:
					app.serverErrorResponse(w, r, err)
				}
				return
			}
			cache.Set("bookUserID", book.UserID, ttlcache.DefaultTTL)
			// Retrieve book's UserID field from the cache that has just been set
			bookUserID = cache.Get("bookUserID")
		}
		// Compare user's ID and book's UserID field in cache. If they aren't the same,
		// forbid further action
		if user.ID != bookUserID.Value() {
			app.notPermittedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return app.requireActivatedUser(fn)
}

// requireReviewOwnerPermission middleware checks that a user is authenticated, activated and is the owner of the review.
func (app *application) requireReviewOwnerPermission(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from request context
		user := app.contextGetUser(r)
		// Check whether review's UserID field is found in cache
		cache := app.cache
		reviewUserID := cache.Get("reviewUserID")
		if reviewUserID == nil {
			// If review's UserID field is not found, fetch it from the database and set to cache
			id, err := app.readIDParam(r, "reviewId")
			if err != nil || id < 1 {
				app.notFoundResponse(w, r)
				return
			}
			review, err := app.models.Reviews.Get(id)
			if err != nil {
				switch {
				case errors.Is(err, data.ErrRecordNotFound):
					app.notFoundResponse(w, r)
				default:
					app.serverErrorResponse(w, r, err)
				}
				return
			}
			cache.Set("reviewUserID", review.UserID, ttlcache.DefaultTTL)
			// Retrieve review's UserID field from the cache that has just been set
			reviewUserID = cache.Get("reviewUserID")
		}
		// Compare user's ID and review's UserID field in cache. If they aren't the same,
		// forbid further action
		if user.ID != reviewUserID.Value() {
			app.notPermittedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return app.requireActivatedUser(fn)
}

// requireBooklistOwnerPermission middleware checks that a user is authenticated, activated and is the owner of the booklist.
func (app *application) requireBooklistOwnerPermission(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from request context
		user := app.contextGetUser(r)
		// Check whether booklist's UserID field is found in cache
		cache := app.cache
		booklistUserID := cache.Get("booklistUserID")
		if booklistUserID == nil {
			// If booklist's UserID field is not found, fetch it from the database and set to cache
			id, err := app.readIDParam(r, "booklistId")
			if err != nil || id < 1 {
				app.notFoundResponse(w, r)
				return
			}
			booklist, err := app.models.Booklists.Get(id)
			if err != nil {
				switch {
				case errors.Is(err, data.ErrRecordNotFound):
					app.notFoundResponse(w, r)
				default:
					app.serverErrorResponse(w, r, err)
				}
				return
			}
			cache.Set("booklistUserID", booklist.UserID, ttlcache.DefaultTTL)
			// Retrieve booklist's UserID field from the cache that has just been set
			booklistUserID = cache.Get("booklistUserID")
		}
		// Compare user's ID and booklist's UserID field in cache. If they aren't the same,
		// forbid further action
		if user.ID != booklistUserID.Value() {
			app.notPermittedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return app.requireActivatedUser(fn)
}
