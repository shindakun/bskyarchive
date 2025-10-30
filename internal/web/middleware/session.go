package middleware

import (
	"net/http"

	"github.com/gorilla/sessions"
)

const sessionName = "bskyarchive-session"

// SessionStore wraps gorilla sessions store
type SessionStore struct {
	store *sessions.CookieStore
}

// NewSessionStore creates a new session store with a default secret
// TODO: Replace with config-based secret in Phase 3
func NewSessionStore() *SessionStore {
	// Temporary secret - will be replaced with config value
	secret := []byte("temporary-secret-key-replace-in-phase-3-minimum-32-chars")
	return &SessionStore{
		store: sessions.NewCookieStore(secret),
	}
}

// SessionMiddleware adds session support to the request context
func SessionMiddleware(store *SessionStore) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Session handling will be implemented in Phase 3
			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuth is a middleware that requires authentication
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authentication check will be implemented in Phase 3
		// For now, redirect to login
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
	})
}
