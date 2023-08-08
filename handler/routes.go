package handler

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func (h *Handler) Routes() http.Handler {
	router := httprouter.New()

	router.NotFound = http.HandlerFunc(h.notFoundResponse)
	router.MethodNotAllowed = http.HandlerFunc(h.methodNotAllowed)

	router.HandlerFunc(http.MethodGet, "/v1/books", h.requireActivatedUser(h.listBooksHandler))
	router.HandlerFunc(http.MethodPost, "/v1/books", h.requireActivatedUser(h.createBookHandler))
	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId", h.requireActivatedUser(h.showBookHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/books/:bookId", h.requireBookOwnerPermission(h.updateBookHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId", h.requireBookOwnerPermission(h.deleteBookHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/books/:bookId/cover", h.requireBookOwnerPermission(h.updateBookCoverHandler))
	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId/download", h.requireActivatedUser(h.downloadBookHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId/download", h.requireActivatedUser(h.deleteBookFromDownloadsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/books/:bookId/favourite", h.requireActivatedUser(h.favouriteBookHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId/favourite", h.requireActivatedUser(h.deleteFavouriteBookHandler))

	router.HandlerFunc(http.MethodGet, "/v1/categories", h.requireActivatedUser(h.listCategoriesHandler))
	router.HandlerFunc(http.MethodGet, "/v1/categories/:categoryId", h.requireActivatedUser(h.showCategoryHandler))

	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId/reviews", h.requireActivatedUser(h.listReviewsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/books/:bookId/reviews", h.requireActivatedUser(h.createReviewHandler))
	router.HandlerFunc(http.MethodGet, "/v1/books/:bookId/reviews/:reviewId", h.requireActivatedUser(h.showReviewHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/books/:bookId/reviews/:reviewId", h.requireReviewOwnerPermission(h.updateReviewHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/books/:bookId/reviews/:reviewId", h.requireReviewOwnerPermission(h.deleteReviewHandler))

	router.HandlerFunc(http.MethodGet, "/v1/booklists", h.requireActivatedUser(h.listBooklistsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists", h.requireActivatedUser(h.createBooklistHandler))
	router.HandlerFunc(http.MethodGet, "/v1/booklists/:booklistId", h.requireActivatedUser(h.showBooklistHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/booklists/:booklistId", h.requireBooklistOwnerPermission(h.updateBooklistHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId", h.requireBooklistOwnerPermission(h.deleteBooklistHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/books/:bookId", h.requireBooklistOwnerPermission(h.addBookToBooklistHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId/books/:bookId", h.requireBooklistOwnerPermission(h.deleteBookFromBooklistHandler))
	router.HandlerFunc(http.MethodGet, "/v1/booklists/:booklistId/books/:bookId", h.requireActivatedUser(h.showBookInBooklistHandler))
	router.HandlerFunc(http.MethodGet, "/v1/booklists/:booklistId/books", h.requireActivatedUser(h.findBooksForBooklistHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/favourite", h.requireActivatedUser(h.favouriteBooklistHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId/favourite", h.requireActivatedUser(h.DeleteFavouriteBooklistHandler))

	router.HandlerFunc(http.MethodGet, "/v1/booklists/:booklistId/comments", h.requireActivatedUser(h.listCommentsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/comments", h.requireActivatedUser(h.createCommentHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/booklists/:booklistId/comments", h.requireCommentOwnerPermission(h.updateCommentHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/booklists/:booklistId/comments", h.requireCommentOwnerPermission(h.deleteCommentHandler))
	router.HandlerFunc(http.MethodPost, "/v1/booklists/:booklistId/comments/:commentId", h.requireActivatedUser(h.createCommentReplyHandler))

	router.HandlerFunc(http.MethodGet, "/v1/requests", h.requireActivatedUser(h.listRequestsHandler))
	router.HandlerFunc(http.MethodPost, "/v1/requests", h.requireActivatedUser(h.createRequestHandler))
	router.HandlerFunc(http.MethodGet, "/v1/requests/:requestId", h.requireActivatedUser(h.showRequestHandler))
	router.HandlerFunc(http.MethodPost, "/v1/requests/:requestId/subscribe", h.requireActivatedUser(h.subscribeRequestHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/requests/:requestId/unsubscribe", h.requireActivatedUser(h.unsubscribeRequestHandler))

	router.HandlerFunc(http.MethodPost, "/v1/users", h.registerUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/activated", h.activateUserHandler)
	router.HandlerFunc(http.MethodPut, "/v1/users/password", h.resetUserPasswordHandler)

	router.HandlerFunc(http.MethodGet, "/v1/users/profile", h.requireActivatedUser(h.showUserHandler))
	router.HandlerFunc(http.MethodPatch, "/v1/users/profile", h.requireActivatedUser(h.updateUserHandler))
	router.HandlerFunc(http.MethodPut, "/v1/users/profile", h.requireActivatedUser(h.updateUserPasswordHandler))
	router.HandlerFunc(http.MethodDelete, "/v1/users/profile", h.requireActivatedUser(h.deleteUserHandler))

	router.HandlerFunc(http.MethodGet, "/v1/users/books", h.requireActivatedUser(h.listUserBooksHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/books/favourite", h.requireActivatedUser(h.listUserFavouriteBooksHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/booklists", h.requireActivatedUser(h.listUserBooklistsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/booklists/favourite", h.requireActivatedUser(h.listUserFavouriteBooklistsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/downloads", h.requireActivatedUser(h.listUserDownloadsHandler))
	router.HandlerFunc(http.MethodGet, "/v1/users/requests", h.requireActivatedUser(h.listUserRequestsHandler))

	router.HandlerFunc(http.MethodPost, "/v1/tokens/activation", h.createActivationTokenHandler)
	router.HandlerFunc(http.MethodPost, "/v1/tokens/authentication", h.createAuthenticationTokenHandler)
	router.HandlerFunc(http.MethodDelete, "/v1/tokens/authentication", h.requireAuthenticatedUser(h.deleteAuthenticationTokenHandler))
	router.HandlerFunc(http.MethodPost, "/v1/tokens/password-reset", h.createPasswordResetTokenHandler)

	// router.HandlerFunc(http.MethodGet, "/debug/vars", app.basicAuth(expvar.Handler().ServeHTTP))
	router.HandlerFunc(http.MethodGet, "/v1/healthcheck", h.healthcheckHandler)

	// Swagger routes
	router.HandlerFunc(http.MethodGet, "/spec", h.handleSwaggerFile())
	router.HandlerFunc(http.MethodGet, "/docs/*any", httpSwagger.Handler(httpSwagger.URL("/spec")))

	return h.recoverPanic(h.enableCORS(h.rateLimit(h.authenticate(router))))
}
