package main

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func (app *application) routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(app.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(app.methodNotAllowed)

	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", app.healthcheckHandler)

	router.HandlerFunc(http.MethodGet, "/v1/books", app.requireActivatedUser(app.listBooksHandler))
	router.HandlerFunc(http.MethodPost, "/v1/books", app.requireActivatedUser(app.createBookHandler))
	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId", app.requireActivatedUser(app.showBookHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/books/:bookId", app.requireBookOwnerPermission(app.updateBookHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId", app.requireBookOwnerPermission(app.deleteBookHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/books/:bookId/cover", app.requireBookOwnerPermission(app.updateBookCoverHandler))
	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId/download", app.requireActivatedUser(app.downloadBookHandler))
	router.HandlerFunc(http.MethodPost, "/v1/books/:bookId/favourite", app.requireActivatedUser(app.addFavouriteBookHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId/favourite", app.requireActivatedUser(app.removeFavouriteBookHandler))

	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId/reviews", app.requireActivatedUser(app.listReviewsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/books/:bookId/reviews", app.requireActivatedUser(app.createReviewHandler))
	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId/reviews/:reviewId", app.requireActivatedUser(app.showReviewHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/books/:bookId/reviews/:reviewId", app.requireReviewOwnerPermission(app.updateReviewHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/books/:bookId/reviews/:reviewId/upvote", app.requireActivatedUser(app.upvoteReviewHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/books/:bookId/reviews/:reviewId/downvote", app.requireActivatedUser(app.downvoteReviewHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId/reviews/:reviewId", app.requireReviewOwnerPermission(app.deleteReviewHandler))

	// router.HandlerFunc(http.MethodGet, "/v1/booklists", app.requireActivatedUser(app.listBooklistsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists", app.requireActivatedUser(app.createBooklistHandler))
	router.HandlerFunc(http.MethodGet, "/v1/booklists/:booklistId", app.requireActivatedUser(app.showBooklistHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/booklists/:booklistId", app.requireBooklistOwnerPermission(app.updateBooklistHandler))
	// router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/book", app.requireBooklistOwnerPermission(app.addBookToBooklistHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId", app.requireBooklistOwnerPermission(app.deleteBooklistHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/favourite", app.requireActivatedUser(app.addFavouriteBooklistHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId/favourite", app.requireActivatedUser(app.removeFavouriteBooklistHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/password", app.updateUserPasswordHandler)

	router.HandlerFunc(http.MethodGet, "/v1/users/favourite-books", app.requireActivatedUser(app.listFavouriteBooksHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/favourite-booklists", app.requireActivatedUser(app.listFavouriteBooklistsHandler))

	router.HandlerFunc(http.MethodPost, "/v1/tokens/activation", app.createActivationTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/password-reset", app.createPasswordResetTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return app.authenticate(router)
}
