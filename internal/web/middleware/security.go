package middleware

import (
	"net/http"

	"github.com/shindakun/bskyarchive/internal/config"
)

// SecurityHeaders creates middleware that adds HTTP security headers to all responses
func SecurityHeaders(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set security headers from configuration
			headers := cfg.Server.Security.Headers

			// X-Frame-Options: Prevents clickjacking
			if headers.XFrameOptions != "" {
				w.Header().Set("X-Frame-Options", headers.XFrameOptions)
			}

			// X-Content-Type-Options: Prevents MIME sniffing
			if headers.XContentTypeOptions != "" {
				w.Header().Set("X-Content-Type-Options", headers.XContentTypeOptions)
			}

			// X-XSS-Protection: Enables browser XSS filtering
			if headers.XXSSProtection != "" {
				w.Header().Set("X-XSS-Protection", headers.XXSSProtection)
			}

			// Referrer-Policy: Controls referrer information
			if headers.ReferrerPolicy != "" {
				w.Header().Set("Referrer-Policy", headers.ReferrerPolicy)
			}

			// Content-Security-Policy: Prevents XSS and other code injection attacks
			if headers.ContentSecurityPolicy != "" {
				w.Header().Set("Content-Security-Policy", headers.ContentSecurityPolicy)
			}

			// Strict-Transport-Security: Forces HTTPS (only when TLS enabled)
			// Check if BASE_URL uses HTTPS to determine if we should set HSTS
			if cfg.IsHTTPS() && headers.StrictTransportSecurity != "" {
				w.Header().Set("Strict-Transport-Security", headers.StrictTransportSecurity)
			}

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}
