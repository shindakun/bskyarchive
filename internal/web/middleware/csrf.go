package middleware

import (
	"net/http"

	"github.com/gorilla/csrf"
)

// CSRFProtection creates a CSRF protection middleware using gorilla/csrf
// with exemption for OAuth login endpoint
func CSRFProtection(secret []byte, secure bool) func(http.Handler) http.Handler {
	// Create the base CSRF middleware
	csrfMiddleware := csrf.Protect(
		secret,
		csrf.Secure(secure),
		csrf.FieldName("csrf_token"),
		csrf.RequestHeader("X-CSRF-Token"), // For HTMX requests
		csrf.ErrorHandler(http.HandlerFunc(CSRFFailureHandler)),
	)

	// Wrap to exempt OAuth login path from CSRF validation
	// Note: We apply the middleware to all requests (for cookie setting)
	// but short-circuit validation for /auth/login POST requests
	return func(next http.Handler) http.Handler {
		// Wrap the next handler with CSRF protection
		csrfProtected := csrfMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// If this is a POST to /auth/login, skip CSRF validation
			// (OAuth flow doesn't include CSRF token in the login form)
			if r.Method == http.MethodPost && r.URL.Path == "/auth/login" {
				next.ServeHTTP(w, r)
				return
			}

			// For all other requests, proceed with CSRF validation
			next.ServeHTTP(w, r)
		}))

		return csrfProtected
	}
}

// CSRFFailureHandler provides HTMX-aware error handling for CSRF failures
func CSRFFailureHandler(w http.ResponseWriter, r *http.Request) {
	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// HTMX request - return HTML fragment with proper status
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`<div class="error" role="alert">
			<strong>Security Error:</strong> Your session has expired or the security token is invalid.
			Please <a href="javascript:window.location.reload()">refresh the page</a> and try again.
		</div>`))
		return
	}

	// Regular request - return standard error
	http.Error(w, "CSRF token validation failed. Please refresh the page and try again.", http.StatusForbidden)
}
