package handler

import (
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"expvar"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emzola/bibliotheca/data"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/emzola/bibliotheca/service"
	"github.com/felixge/httpsnoop"
	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/time/rate"
)

// recoverPanic middleware recovers from panics and will always be run in the event of a panic.
func (h *Handler) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				h.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// rateLimit middleware implements IP-based rate limiting to prevent clients from making too many requests
// too quickly, and putting excessive strain on the server.
func (h *Handler) rateLimit(next http.Handler) http.Handler {
	// Define a client struct to hold the rate limiter and last seen time for each
	// client.
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	// Declare a mutex and a map to hold the clients' IP addresses and rate limiters.
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)
	// Launch a background goroutine which removes old entries from the clients map once
	// every minute.
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			// Loop through all clients. If they haven't been seen within the last three
			// minutes, delete the corresponding entry from the map.
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only carry out rate limiting check if rate limiting is enabled
		if h.config.Limiter.Enabled {
			// Extract the client's IP address from the request
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				h.serverErrorResponse(w, r, err)
				return
			}
			// Lock the mutex to prevent this code from being executed concurrently.
			mu.Lock()
			// Check to see if the IP address already exists in the map. If it doesn't, then
			// initialize a new rate limiter and add the IP address and limiter to the map.
			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(h.config.Limiter.RPS), h.config.Limiter.Burst),
				}
			}
			// Update the last seen time for the client
			clients[ip].lastSeen = time.Now()
			// Call the Allow() method on the rate limiter for the current IP address. If
			// the request isn't allowed, unlock the mutex and send a 429 Too Many Requests
			// response, just like before.
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				h.rateLimitExceededResponse(w, r)
				return
			}
			// Very importantly, unlock the mutex before calling the next handler in the
			// chain. Notice that we DON'T use defer to unlock the mutex, as that would mean
			// that the mutex isn't unlocked until all the handlers downstream of this
			// middleware have also returned.
			mu.Unlock()
		}
		next.ServeHTTP(w, r)
	})
}

// enableCORS middleware relaxes the same-origin policy.
func (h *Handler) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		origin := r.Header.Get("Origin")
		if origin != "" {
			for i := range h.config.Cors.TrustedOrigins {
				if origin == h.config.Cors.TrustedOrigins[i] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
						w.Header().Set("Access-Control-Allow-Methods", "OPTIONS, PUT, PATCH, DELETE")
						w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
						w.WriteHeader(http.StatusOK)
						return
					}
					break
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

// authenticate middleware authenticates users. It returns an authenticated or anonymous user.
func (h *Handler) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")
		authorizationHeader := r.Header.Get("Authorization")
		headerParts := strings.Split(authorizationHeader, " ")
		if authorizationHeader == "" || headerParts[0] == "Basic" {
			r = h.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}
		if len(headerParts) != 2 || headerParts[0] != "Bearer" {
			h.invalidAuthenticationTokenResponse(w, r)
			return
		}
		token := headerParts[1]
		v := validator.New()
		if data.ValidateTokenPlaintext(v, token); !v.Valid() {
			h.invalidAuthenticationTokenResponse(w, r)
			return
		}
		user, err := h.service.GetUserForToken(data.ScopeAuthentication, token)
		if err != nil {
			switch {
			case errors.Is(err, service.ErrFailedValidation):
				h.invalidAuthenticationTokenResponse(w, r)
			default:
				h.serverErrorResponse(w, r, err)
			}
			return
		}
		r = h.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// requireAuthenticatedUser middleware checks that a user is not anonymous.
func (h *Handler) requireAuthenticatedUser(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := h.contextGetUser(r)
		if user.IsAnonymous() {
			h.authenticationRequiredResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireActivatedUser middleware checks that a user is both authenticated and activated.
func (h *Handler) requireActivatedUser(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := h.contextGetUser(r)
		if !user.Activated {
			h.inactiveAccountResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return h.requireAuthenticatedUser(fn)
}

// requireBookOwnerPermission middleware checks that a user is authenticated, activated and is the owner of the book.
func (h *Handler) requireBookOwnerPermission(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from request context
		user := h.contextGetUser(r)
		// Check whether book's UserID field is found in cache
		cache := h.cache
		bookUserID := cache.Get("bookUserID")
		if bookUserID == nil {
			// If book's UserID field is not found, fetch it from the database and set to cache
			bookID, err := h.readIDParam(r, "bookId")
			if err != nil {
				h.notFoundResponse(w, r)
				return
			}
			book, err := h.service.GetBook(bookID)
			if err != nil {
				switch {
				case errors.Is(err, service.ErrRecordNotFound):
					h.notFoundResponse(w, r)
				default:
					h.serverErrorResponse(w, r, err)
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
			h.notPermittedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return h.requireActivatedUser(fn)
}

// requireReviewOwnerPermission middleware checks that a user is authenticated, activated and is the owner of the review.
func (h *Handler) requireReviewOwnerPermission(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from request context
		user := h.contextGetUser(r)
		// Check whether review's UserID field is found in cache
		cache := h.cache
		reviewUserID := cache.Get("reviewUserID")
		if reviewUserID == nil {
			// If review's UserID field is not found, fetch it from the database and set to cache
			reviewID, err := h.readIDParam(r, "reviewId")
			if err != nil {
				h.notFoundResponse(w, r)
				return
			}
			review, err := h.service.GetReview(reviewID)
			if err != nil {
				switch {
				case errors.Is(err, service.ErrRecordNotFound):
					h.notFoundResponse(w, r)
				default:
					h.serverErrorResponse(w, r, err)
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
			h.notPermittedResponse(w, r)
		}
		next.ServeHTTP(w, r)
	})
	return h.requireActivatedUser(fn)
}

// requireBooklistOwnerPermission middleware checks that a user is authenticated, activated and is the owner of the booklist.
func (h *Handler) requireBooklistOwnerPermission(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from request context
		user := h.contextGetUser(r)
		// Check whether booklist's UserID field is found in cache
		cache := h.cache
		booklistUserID := cache.Get("booklistUserID")
		if booklistUserID == nil {
			// If booklist's UserID field is not found, fetch it from the database and set to cache
			booklistID, err := h.readIDParam(r, "booklistId")
			if err != nil {
				h.notFoundResponse(w, r)
				return
			}
			booklist, err := h.service.GetBooklist(booklistID, data.Filters{})
			if err != nil {
				switch {
				case errors.Is(err, service.ErrRecordNotFound):
					h.notFoundResponse(w, r)
				default:
					h.serverErrorResponse(w, r, err)
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
			h.notPermittedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return h.requireActivatedUser(fn)
}

// requireCommentOwnerPermission middleware checks that a user is authenticated, activated and is the owner of the comment.
func (h *Handler) requireCommentOwnerPermission(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from request context
		user := h.contextGetUser(r)
		// Check whether comment's UserID field is found in cache
		cache := h.cache
		commentUserID := cache.Get("commentUserID")
		if commentUserID == nil {
			// If comment's UserID field is not found, fetch it from the database and set to cache
			commentID, err := h.readIDParam(r, "commentId")
			if err != nil {
				h.notFoundResponse(w, r)
				return
			}
			comment, err := h.service.GetComment(commentID)
			if err != nil {
				switch {
				case errors.Is(err, service.ErrRecordNotFound):
					h.notFoundResponse(w, r)
				default:
					h.serverErrorResponse(w, r, err)
				}
				return
			}
			cache.Set("commentUserID", comment.UserID, ttlcache.DefaultTTL)
			// Retrieve comment's UserID field from the cache that has just been set
			commentUserID = cache.Get("commentUserID")
		}
		// Compare user's ID and comment's UserID field in cache. If they aren't the same,
		// forbid further action
		if user.ID != commentUserID.Value() {
			h.notPermittedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return h.requireActivatedUser(fn)
}

// metrics middleware exposes request-level metrics.
func (h *Handler) metrics(next http.Handler) http.Handler {
	if h.config.Metrics.Enabled {
		totalRequestsReceived := expvar.NewInt("total_requests_received")
		totalResponsesSent := expvar.NewInt("total_responses_sent")
		totalProcessingTimeMicrosecond := expvar.NewInt("total_processing_time_μs")
		totalResponsesSentBystatus := expvar.NewMap("total_responses_sent_by_status")
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			totalRequestsReceived.Add(1)
			metrics := httpsnoop.CaptureMetrics(next, w, r)
			totalResponsesSent.Add(1)
			totalProcessingTimeMicrosecond.Add(metrics.Duration.Microseconds())
			totalResponsesSentBystatus.Add(strconv.Itoa(metrics.Code), 1)
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

// basicAuth middleware implements basic authentication for the /debug/vars endpoint.
func (h *Handler) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(h.config.BasicAuth.Username))
			expectedPasswordHash := sha256.Sum256([]byte(h.config.BasicAuth.Password))
			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)
			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		h.invalidCredentialsResponse(w, r)
	})
}
