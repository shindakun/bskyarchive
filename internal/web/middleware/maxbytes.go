package middleware

import (
	"net/http"
)

// MaxBytesMiddleware creates middleware that limits request body size
func MaxBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Wrap request body with MaxBytesReader to enforce size limit
			r.Body = http.MaxBytesReader(w, r.Body, maxBytes)

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}
