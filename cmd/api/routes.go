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
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId/download", app.requireActivatedUser(app.deleteBookFromDownloadsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/books/:bookId/favourite", app.requireActivatedUser(app.addFavouriteBookHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId/favourite", app.requireActivatedUser(app.removeFavouriteBookHandler))

	router.HandlerFunc(http.MethodGet, "/v1/categories", app.requireActivatedUser(app.listCategoriesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/categories/:categoryId", app.requireActivatedUser(app.showCategoryHandler))

	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId/reviews", app.requireActivatedUser(app.listReviewsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/books/:bookId/reviews", app.requireActivatedUser(app.createReviewHandler))
	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId/reviews/:reviewId", app.requireActivatedUser(app.showReviewHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/books/:bookId/reviews/:reviewId", app.requireReviewOwnerPermission(app.updateReviewHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId/reviews/:reviewId", app.requireReviewOwnerPermission(app.deleteReviewHandler))

	router.HandlerFunc(http.MethodGet, "/v1/booklists", app.requireActivatedUser(app.listBooklistsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists", app.requireActivatedUser(app.createBooklistHandler))
	router.HandlerFunc(http.MethodGet, "/v1/booklists/:booklistId", app.requireActivatedUser(app.showBooklistHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/booklists/:booklistId", app.requireBooklistOwnerPermission(app.updateBooklistHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId", app.requireBooklistOwnerPermission(app.deleteBooklistHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/books/:bookId", app.requireBooklistOwnerPermission(app.addToBooklistHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId/books/:bookId", app.requireBooklistOwnerPermission(app.deleteFromBooklistHandler))
	router.HandlerFunc(http.MethodGet, "/v1/booklists/:booklistId/books/:bookId", app.requireActivatedUser(app.showBookInBooklistHandler))
	router.HandlerFunc(http.MethodGet, "/v1/booklists/:booklistId/books", app.requireActivatedUser(app.findBooksForBooklistHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/favourite", app.requireActivatedUser(app.addFavouriteBooklistHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId/favourite", app.requireActivatedUser(app.removeFavouriteBooklistHandler))
	router.HandlerFunc(http.MethodGet, "/v1/booklists/:booklistId/comments", app.requireActivatedUser(app.listCommentsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/comments", app.requireActivatedUser(app.createCommentHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/booklists/:booklistId/comments", app.requireCommentOwnerPermission(app.updateCommentHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId/comments", app.requireCommentOwnerPermission(app.deleteCommentHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/comments/:commentId", app.requireActivatedUser(app.createCommentReplyHandler))

	router.HandlerFunc(http.MethodGet, "/v1/requests", app.requireActivatedUser(app.listRequestsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/requests", app.requireActivatedUser(app.createRequestHandler))
	router.HandlerFunc(http.MethodGet, "/v1/requests/:requestId", app.requireActivatedUser(app.showRequestHandler))
	router.HandlerFunc(http.MethodPost, "/v1/requests/:requestId/subscribe", app.requireActivatedUser(app.subscribeRequestHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/requests/:requestId/unsubscribe", app.requireActivatedUser(app.unsubscribeRequestHandler))

	router.HandlerFunc(http.MethodGet, "/v1/users/profile", app.requireActivatedUser(app.showUserHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/users/profile", app.requireActivatedUser(app.updateUserHandler))
	router.HandlerFunc(http.MethodPut, "/v1/users/profile", app.requireActivatedUser(app.updateUserPasswordHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/users/profile", app.requireActivatedUser(app.deleteUserHandler))

	router.HandlerFunc(http.MethodGet, "/v1/users/books", app.requireActivatedUser(app.listUsersBooksHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/books/favourite", app.requireActivatedUser(app.listFavouriteBooksHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/booklists", app.requireActivatedUser(app.listUserBooklistsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/booklists/favourite", app.requireActivatedUser(app.listFavouriteBooklistsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/downloads", app.requireActivatedUser(app.listUserDownloadsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/requests", app.requireActivatedUser(app.listUserRequestsHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", app.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", app.activateUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/password", app.resetUserPasswordHandler)

	router.HandlerFunc(http.MethodPost, "/v1/tokens/activation", app.createActivationTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/password-reset", app.createPasswordResetTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", app.createAuthenticationTokenHandler)

	return app.recoverPanic(app.authenticate(router))
}
