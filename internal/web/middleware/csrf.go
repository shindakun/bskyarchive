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
	// The OAuth login form doesn't include CSRF tokens (simple HTML form)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Exempt /auth/login from CSRF validation entirely
			// OAuth flow uses its own state parameter for protection
			if r.URL.Path == "/auth/login" {
				next.ServeHTTP(w, r)
				return
			}

			// For all other requests, apply CSRF protection
			csrfMiddleware(next).ServeHTTP(w, r)
		})
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
