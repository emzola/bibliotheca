package main

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

	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/felixge/httpsnoop"
	"github.com/jellydator/ttlcache/v3"
	"golang.org/x/time/rate"
)

// recoverPanic middleware recovers from panics and will always be run in the event of a panic.
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// rateLimit middleware implements IP-based rate limiting to prevent clients from making too many requests
// too quickly, and putting excessive strain on the server.
func (app *application) rateLimit(next http.Handler) http.Handler {
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
		if app.config.limiter.enabled {
			// Extract the client's IP address from the request
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				app.serverErrorResponse(w, r, err)
				return
			}
			// Lock the mutex to prevent this code from being executed concurrently.
			mu.Lock()
			// Check to see if the IP address already exists in the map. If it doesn't, then
			// initialize a new rate limiter and add the IP address and limiter to the map.
			if _, found := clients[ip]; !found {
				clients[ip] = &client{
					limiter: rate.NewLimiter(rate.Limit(app.config.limiter.rps), app.config.limiter.burst),
				}
			}
			// Update the last seen time for the client
			clients[ip].lastSeen = time.Now()
			// Call the Allow() method on the rate limiter for the current IP address. If
			// the request isn't allowed, unlock the mutex and send a 429 Too Many Requests
			// response, just like before.
			if !clients[ip].limiter.Allow() {
				mu.Unlock()
				app.rateLimitExceededResponse(w, r)
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
func (app *application) enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Origin")
		w.Header().Add("Vary", "Access-Control-Request-Method")
		origin := r.Header.Get("Origin")
		if origin != "" {
			for i := range app.config.cors.trustedOrigins {
				if origin == app.config.cors.trustedOrigins[i] {
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
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Vary", "Authorization")
		authorizationHeader := r.Header.Get("Authorization")
		headerParts := strings.Split(authorizationHeader, " ")
		if authorizationHeader == "" || headerParts[0] == "Basic" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}
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

// requireCommentOwnerPermission middleware checks that a user is authenticated, activated and is the owner of the comment.
func (app *application) requireCommentOwnerPermission(next http.HandlerFunc) http.HandlerFunc {
	fn := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Get user from request context
		user := app.contextGetUser(r)
		// Check whether comment's UserID field is found in cache
		cache := app.cache
		commentUserID := cache.Get("commentUserID")
		if commentUserID == nil {
			// If comment's UserID field is not found, fetch it from the database and set to cache
			id, err := app.readIDParam(r, "commentId")
			if err != nil || id < 1 {
				app.notFoundResponse(w, r)
				return
			}
			comment, err := app.models.Comments.Get(id)
			if err != nil {
				switch {
				case errors.Is(err, data.ErrRecordNotFound):
					app.notFoundResponse(w, r)
				default:
					app.serverErrorResponse(w, r, err)
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
			app.notPermittedResponse(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return app.requireActivatedUser(fn)
}

// metrics middleware exposes the application's request-level metrics
func (app *application) metrics(next http.Handler) http.Handler {
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

// basicAuth middleware implements basic authentication for the /debug/vars endpoint.
func (app *application) basicAuth(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			usernameHash := sha256.Sum256([]byte(username))
			passwordHash := sha256.Sum256([]byte(password))
			expectedUsernameHash := sha256.Sum256([]byte(app.config.basicAuth.username))
			expectedPasswordHash := sha256.Sum256([]byte(app.config.basicAuth.password))
			usernameMatch := (subtle.ConstantTimeCompare(usernameHash[:], expectedUsernameHash[:]) == 1)
			passwordMatch := (subtle.ConstantTimeCompare(passwordHash[:], expectedPasswordHash[:]) == 1)
			if usernameMatch && passwordMatch {
				next.ServeHTTP(w, r)
				return
			}
		}
		w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
		app.invalidCredentialsResponse(w, r)
	})
}
