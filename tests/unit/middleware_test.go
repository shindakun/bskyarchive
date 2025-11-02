package unit

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/csrf"
	"github.com/shindakun/bskyarchive/internal/config"
	webmiddleware "github.com/shindakun/bskyarchive/internal/web/middleware"
)

// TestCSRFTokenGeneration verifies that CSRF middleware generates valid tokens
// Test Case: TC-CSRF-001, TC-CSRF-009
func TestCSRFTokenGeneration(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")

	// Create CSRF middleware
	csrfMiddleware := webmiddleware.CSRFProtection(secret, false)

	// Create a test handler that captures the CSRF token
	var token1, token2 string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := csrf.Token(r)
		if token == "" {
			t.Error("CSRF token is empty")
		}
		if token1 == "" {
			token1 = token
		} else {
			token2 = token
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap handler with CSRF middleware
	protectedHandler := csrfMiddleware(handler)

	// Test 1: Token is generated for first request
	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	protectedHandler.ServeHTTP(w1, req1)

	if token1 == "" {
		t.Fatal("Expected CSRF token to be generated, got empty string")
	}

	// Verify token has reasonable length (gorilla/csrf tokens are base64 encoded)
	if len(token1) < 20 {
		t.Errorf("CSRF token too short: got %d bytes, expected at least 20", len(token1))
	}

	// Test 2: Token is different for second request (different session)
	req2 := httptest.NewRequest("GET", "/test", nil)
	w2 := httptest.NewRecorder()
	protectedHandler.ServeHTTP(w2, req2)

	if token2 == "" {
		t.Fatal("Expected CSRF token to be generated for second request, got empty string")
	}

	if token1 == token2 {
		t.Error("Expected different tokens for different sessions, got same token")
	}

	t.Logf("Token 1: %s", token1[:10]+"...")
	t.Logf("Token 2: %s", token2[:10]+"...")
}

// TestCSRFMiddlewareWrapping verifies middleware can be applied without panics
func TestCSRFMiddlewareWrapping(t *testing.T) {
	secret := []byte("test-secret-key-32-bytes-long!!")

	// Create CSRF middleware
	csrfMiddleware := webmiddleware.CSRFProtection(secret, false)

	// Create simple handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Wrap handler - should not panic
	protectedHandler := csrfMiddleware(handler)

	// Make request - should not panic
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CSRF middleware panicked: %v", r)
		}
	}()

	protectedHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}
}

// TestCSRFFailureHandler verifies custom failure handler is called
func TestCSRFFailureHandler(t *testing.T) {
	t.Run("HTMX request gets HTML fragment", func(t *testing.T) {
		// Create request with HX-Request header
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("HX-Request", "true")
		w := httptest.NewRecorder()

		// Call failure handler directly
		webmiddleware.CSRFFailureHandler(w, req)

		// Verify response
		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}

		body := w.Body.String()
		if body == "" {
			t.Error("Expected HTML error message, got empty body")
		}

		// Should contain HTML
		if !contains(body, "<div") {
			t.Errorf("Expected HTML fragment, got: %s", body)
		}

		// Should mention security or CSRF
		if !contains(body, "Security Error") && !contains(body, "security") {
			t.Errorf("Expected security error message, got: %s", body)
		}
	})

	t.Run("Regular request gets plain error", func(t *testing.T) {
		// Create regular request without HX-Request header
		req := httptest.NewRequest("POST", "/test", nil)
		w := httptest.NewRecorder()

		// Call failure handler directly
		webmiddleware.CSRFFailureHandler(w, req)

		// Verify response
		if w.Code != http.StatusForbidden {
			t.Errorf("Expected status 403, got %d", w.Code)
		}

		body := w.Body.String()
		if body == "" {
			t.Error("Expected error message, got empty body")
		}

		// Should mention CSRF
		if !contains(body, "CSRF") {
			t.Errorf("Expected CSRF in error message, got: %s", body)
		}
	})
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || contains(s[1:], substr)))
}

// BenchmarkSecurityHeaders measures the performance overhead of SecurityHeaders middleware
// Test Case: TC-SH-007 (Performance requirement: <1ms per request)
func BenchmarkSecurityHeaders(b *testing.B) {
	// Create mock config
	cfg := &config.Config{
		Server: config.ServerConfig{
			BaseURL: "https://example.com",
			Security: config.SecurityConfig{
				Headers: config.SecurityHeadersConfig{
					XFrameOptions:           "DENY",
					XContentTypeOptions:     "nosniff",
					XXSSProtection:          "1; mode=block",
					ReferrerPolicy:          "strict-origin-when-cross-origin",
					ContentSecurityPolicy:   "default-src 'self'",
					StrictTransportSecurity: "max-age=31536000",
				},
			},
		},
	}

	// Create handler with security headers middleware
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := webmiddleware.SecurityHeaders(cfg)
	wrappedHandler := middleware(handler)

	// Create test request
	req := httptest.NewRequest("GET", "/test", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		wrappedHandler.ServeHTTP(w, req)
	}
}
