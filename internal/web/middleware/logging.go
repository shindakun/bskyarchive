package middleware

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/shindakun/bskyarchive/internal/auth"
)

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += n
	return n, err
}

// LoggingMiddleware logs HTTP requests with method, path, status, duration, and DID
func LoggingMiddleware(logger *log.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK, // Default status
			}

			// Process request
			next.ServeHTTP(rw, r)

			// Calculate duration
			duration := time.Since(start)

			// Try to get DID from session
			did := "-"
			if session, ok := auth.GetSessionFromContext(r.Context()); ok && session != nil {
				did = session.DID
			}

			// Log request
			logger.Printf(
				"method=%s path=%s status=%d duration=%s did=%s bytes=%d",
				r.Method,
				r.URL.Path,
				rw.statusCode,
				duration.Round(time.Millisecond),
				did,
				rw.written,
			)
		})
	}
}

// FormatLogEntry formats a consistent log entry
func FormatLogEntry(method, path string, status int, duration time.Duration, did string) string {
	return fmt.Sprintf(
		"method=%s path=%s status=%d duration=%s did=%s",
		method,
		path,
		status,
		duration.Round(time.Millisecond),
		did,
	)
}
