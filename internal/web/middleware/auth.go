package middleware

import (
	"net/http"

	"github.com/shindakun/bskyarchive/internal/auth"
)

// RequireAuth is a middleware that requires authentication
// Redirects to /auth/login if no valid session is found
func RequireAuth(sessionManager *auth.SessionManager) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try to get session
			session, err := sessionManager.GetSession(r)
			if err != nil || session == nil {
				// No valid session, redirect to login
				// Check if this is an HTMX request
				if r.Header.Get("HX-Request") == "true" {
					// For HTMX requests, use HX-Redirect header for client-side redirect
					w.Header().Set("HX-Redirect", "/auth/login")
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				// For regular requests, use standard redirect
				http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
				return
			}

			// Session is valid, add to context and continue
			ctx := auth.SetSessionInContext(r.Context(), session)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
