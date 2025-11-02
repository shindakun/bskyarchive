package integration

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/shindakun/bskyarchive/internal/config"
	"github.com/shindakun/bskyarchive/internal/storage"
	"github.com/shindakun/bskyarchive/internal/web/handlers"
	webmiddleware "github.com/shindakun/bskyarchive/internal/web/middleware"
)

// setupTestServerWithSecurityHeaders creates a test server with security headers enabled
func setupTestServerWithSecurityHeaders(t *testing.T, isHTTPS bool) (*httptest.Server, *sql.DB, func()) {
	// Create temporary database
	db, err := storage.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Create mock config
	cfg := &config.Config{
		Server: config.ServerConfig{
			BaseURL: "http://localhost:8080", // Default to HTTP
			Security: config.SecurityConfig{
				Headers: config.SecurityHeadersConfig{
					XFrameOptions:           "DENY",
					XContentTypeOptions:     "nosniff",
					XXSSProtection:          "1; mode=block",
					ReferrerPolicy:          "strict-origin-when-cross-origin",
					ContentSecurityPolicy:   "default-src 'self'",
					StrictTransportSecurity: "max-age=31536000; includeSubDomains",
				},
			},
		},
	}

	// Set HTTPS if requested
	if isHTTPS {
		cfg.Server.BaseURL = "https://example.com"
	}

	// Create router with security headers middleware
	r := chi.NewRouter()
	r.Use(webmiddleware.SecurityHeaders(cfg))

	// Define test endpoints
	r.Get("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	r.Get("/notfound", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	})

	r.Get("/error", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	})

	// Create test server
	ts := httptest.NewServer(r)

	cleanup := func() {
		ts.Close()
		db.Close()
	}

	return ts, db, cleanup
}

// TestSecurityHeadersOn200OK verifies security headers are present on 200 OK responses
// Test Case: TC-SH-001
func TestSecurityHeadersOn200OK(t *testing.T) {
	ts, _, cleanup := setupTestServerWithSecurityHeaders(t, false)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/ok")
	if err != nil {
		t.Fatalf("Failed to GET /ok: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify all security headers are present
	headers := map[string]string{
		"X-Frame-Options":        "DENY",
		"X-Content-Type-Options": "nosniff",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
		"Content-Security-Policy": "default-src 'self'",
	}

	for headerName, expectedValue := range headers {
		actualValue := resp.Header.Get(headerName)
		if actualValue != expectedValue {
			t.Errorf("Header %s: expected %q, got %q", headerName, expectedValue, actualValue)
		}
	}

	// HSTS should NOT be present (HTTP server)
	if hsts := resp.Header.Get("Strict-Transport-Security"); hsts != "" {
		t.Errorf("HSTS header should not be present on HTTP server, got: %s", hsts)
	}
}

// TestSecurityHeadersOn404NotFound verifies security headers are present on 404 responses
// Test Case: TC-SH-002
func TestSecurityHeadersOn404NotFound(t *testing.T) {
	ts, _, cleanup := setupTestServerWithSecurityHeaders(t, false)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/notfound")
	if err != nil {
		t.Fatalf("Failed to GET /notfound: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}

	// Verify security headers are still present on error responses
	requiredHeaders := []string{
		"X-Frame-Options",
		"X-Content-Type-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Content-Security-Policy",
	}

	for _, headerName := range requiredHeaders {
		if value := resp.Header.Get(headerName); value == "" {
			t.Errorf("Header %s should be present on 404 response", headerName)
		}
	}
}

// TestSecurityHeadersOn500Error verifies security headers are present on 500 responses
// Test Case: TC-SH-003
func TestSecurityHeadersOn500Error(t *testing.T) {
	ts, _, cleanup := setupTestServerWithSecurityHeaders(t, false)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/error")
	if err != nil {
		t.Fatalf("Failed to GET /error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", resp.StatusCode)
	}

	// Verify security headers are still present on error responses
	requiredHeaders := []string{
		"X-Frame-Options",
		"X-Content-Type-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Content-Security-Policy",
	}

	for _, headerName := range requiredHeaders {
		if value := resp.Header.Get(headerName); value == "" {
			t.Errorf("Header %s should be present on 500 response", headerName)
		}
	}
}

// TestSecurityHeadersHSTSWhenHTTPS verifies HSTS header is present when TLS enabled
// Test Case: TC-SH-004
func TestSecurityHeadersHSTSWhenHTTPS(t *testing.T) {
	ts, _, cleanup := setupTestServerWithSecurityHeaders(t, true) // HTTPS enabled
	defer cleanup()

	resp, err := http.Get(ts.URL + "/ok")
	if err != nil {
		t.Fatalf("Failed to GET /ok: %v", err)
	}
	defer resp.Body.Close()

	// HSTS header SHOULD be present when HTTPS is enabled
	hsts := resp.Header.Get("Strict-Transport-Security")
	if hsts == "" {
		t.Error("HSTS header should be present when HTTPS is enabled")
	}

	expectedHSTS := "max-age=31536000; includeSubDomains"
	if hsts != expectedHSTS {
		t.Errorf("HSTS header: expected %q, got %q", expectedHSTS, hsts)
	}

	t.Logf("HSTS header correctly set: %s", hsts)
}

// TestSecurityHeadersHSTSWhenHTTP verifies HSTS header is absent when TLS disabled
// Test Case: TC-SH-005
func TestSecurityHeadersHSTSWhenHTTP(t *testing.T) {
	ts, _, cleanup := setupTestServerWithSecurityHeaders(t, false) // HTTP only
	defer cleanup()

	resp, err := http.Get(ts.URL + "/ok")
	if err != nil {
		t.Fatalf("Failed to GET /ok: %v", err)
	}
	defer resp.Body.Close()

	// HSTS header should NOT be present when HTTP
	hsts := resp.Header.Get("Strict-Transport-Security")
	if hsts != "" {
		t.Errorf("HSTS header should not be present on HTTP server, got: %s", hsts)
	}

	t.Log("HSTS header correctly absent for HTTP server")
}

// TestSecurityHeadersCSPCompatibility verifies CSP is compatible with app requirements
// Test Case: TC-SH-006
func TestSecurityHeadersCSPCompatibility(t *testing.T) {
	ts, _, cleanup := setupTestServerWithSecurityHeaders(t, false)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/ok")
	if err != nil {
		t.Fatalf("Failed to GET /ok: %v", err)
	}
	defer resp.Body.Close()

	csp := resp.Header.Get("Content-Security-Policy")
	if csp == "" {
		t.Fatal("CSP header should be present")
	}

	// Verify CSP contains 'self' directive (required for Pico.css and HTMX)
	if !contains(csp, "'self'") {
		t.Errorf("CSP should contain 'self' directive for local resources, got: %s", csp)
	}

	t.Logf("CSP policy: %s", csp)
}

// TestSecurityHeadersAllStatuses verifies headers on multiple status codes
func TestSecurityHeadersAllStatuses(t *testing.T) {
	ts, _, cleanup := setupTestServerWithSecurityHeaders(t, false)
	defer cleanup()

	testCases := []struct {
		path           string
		expectedStatus int
	}{
		{"/ok", http.StatusOK},
		{"/notfound", http.StatusNotFound},
		{"/error", http.StatusInternalServerError},
	}

	requiredHeaders := []string{
		"X-Frame-Options",
		"X-Content-Type-Options",
		"X-XSS-Protection",
		"Referrer-Policy",
		"Content-Security-Policy",
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			resp, err := http.Get(ts.URL + tc.path)
			if err != nil {
				t.Fatalf("Failed to GET %s: %v", tc.path, err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, resp.StatusCode)
			}

			// Verify all required headers are present
			for _, headerName := range requiredHeaders {
				if value := resp.Header.Get(headerName); value == "" {
					t.Errorf("Header %s should be present on %s (status %d)", headerName, tc.path, tc.expectedStatus)
				}
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || contains(s[1:], substr)))
}

// TestRequestSizeUnderLimit verifies requests under the limit proceed normally
// Test Case: TC-MB-001, TC-MB-002
func TestRequestSizeUnderLimit(t *testing.T) {
	// Create test server with 10MB limit
	cfg := &config.Config{
		Server: config.ServerConfig{
			Security: config.SecurityConfig{
				MaxRequestBytes: 10 * 1024 * 1024, // 10MB
			},
		},
	}

	r := chi.NewRouter()
	r.Use(webmiddleware.MaxBytesMiddleware(cfg.Server.Security.MaxRequestBytes))
	r.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
		// Read body to verify it's accessible
		body := make([]byte, 1024)
		n, _ := r.Body.Read(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Received " + string(rune(n)) + " bytes"))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test 1: Small request (1KB) - well under limit
	t.Run("1KB request succeeds", func(t *testing.T) {
		payload := make([]byte, 1024) // 1KB
		resp, err := http.Post(ts.URL+"/upload", "application/octet-stream", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("Failed to POST: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})

	// Test 2: Request at exactly maxBytes
	t.Run("10MB request at limit succeeds", func(t *testing.T) {
		payload := make([]byte, 10*1024*1024) // Exactly 10MB
		resp, err := http.Post(ts.URL+"/upload", "application/octet-stream", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("Failed to POST: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
	})
}

// TestRequestSizeOverLimit verifies requests over the limit return 413
// Test Case: TC-MB-003, TC-MB-004
func TestRequestSizeOverLimit(t *testing.T) {
	// Create test server with 10MB limit
	cfg := &config.Config{
		Server: config.ServerConfig{
			Security: config.SecurityConfig{
				MaxRequestBytes: 10 * 1024 * 1024, // 10MB
			},
		},
	}

	r := chi.NewRouter()
	r.Use(webmiddleware.MaxBytesMiddleware(cfg.Server.Security.MaxRequestBytes))
	r.Post("/upload", func(w http.ResponseWriter, r *http.Request) {
		// Try to read body - should fail for oversized requests
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test 1: Request over limit (11MB)
	t.Run("11MB request rejected with 413", func(t *testing.T) {
		payload := make([]byte, 11*1024*1024) // 11MB
		resp, err := http.Post(ts.URL+"/upload", "application/octet-stream", bytes.NewReader(payload))
		if err != nil {
			t.Fatalf("Failed to POST: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", resp.StatusCode)
		}
	})

	// Test 2: Much larger request (100MB)
	t.Run("100MB request rejected with 413", func(t *testing.T) {
		// Create a pipe to stream a large payload without allocating all memory
		pr, pw := io.Pipe()
		go func() {
			// Write 100MB in chunks
			chunk := make([]byte, 1024*1024) // 1MB chunks
			for i := 0; i < 100; i++ {
				_, err := pw.Write(chunk)
				if err != nil {
					pw.CloseWithError(err)
					return
				}
			}
			pw.Close()
		}()

		resp, err := http.Post(ts.URL+"/upload", "application/octet-stream", pr)
		if err != nil {
			t.Fatalf("Failed to POST: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413, got %d", resp.StatusCode)
		}
	})
}

// TestStreamingRequestLimited verifies streaming requests are properly limited
// Test Case: TC-MB-005
func TestStreamingRequestLimited(t *testing.T) {
	// Create test server with 1MB limit (smaller for testing)
	cfg := &config.Config{
		Server: config.ServerConfig{
			Security: config.SecurityConfig{
				MaxRequestBytes: 1 * 1024 * 1024, // 1MB
			},
		},
	}

	r := chi.NewRouter()
	r.Use(webmiddleware.MaxBytesMiddleware(cfg.Server.Security.MaxRequestBytes))
	r.Post("/stream", func(w http.ResponseWriter, r *http.Request) {
		// Try to read body in chunks (streaming)
		buf := make([]byte, 32*1024) // 32KB chunks
		totalRead := 0
		for {
			n, err := r.Body.Read(buf)
			totalRead += n
			if err == io.EOF {
				break
			}
			if err != nil {
				http.Error(w, "Error reading stream", http.StatusRequestEntityTooLarge)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf("Received %d bytes", totalRead)))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Test: Streaming 2MB should be rejected (limit is 1MB)
	t.Run("2MB streaming request rejected", func(t *testing.T) {
		pr, pw := io.Pipe()
		go func() {
			// Stream 2MB in small chunks
			chunk := make([]byte, 32*1024) // 32KB chunks
			for i := 0; i < 64; i++ { // 64 * 32KB = 2MB
				_, err := pw.Write(chunk)
				if err != nil {
					pw.CloseWithError(err)
					return
				}
			}
			pw.Close()
		}()

		resp, err := http.Post(ts.URL+"/stream", "application/octet-stream", pr)
		if err != nil {
			t.Fatalf("Failed to POST: %v", err)
		}
		defer resp.Body.Close()

		// Should get 413 because stream exceeded limit
		if resp.StatusCode != http.StatusRequestEntityTooLarge {
			t.Errorf("Expected status 413 for oversized stream, got %d", resp.StatusCode)
		}
	})
}

// setupTestServerWithStaticFiles creates a test server with static file serving
func setupTestServerWithStaticFiles(t *testing.T) (*httptest.Server, func()) {
	// Create temporary test static directory
	tmpDir := t.TempDir()

	// Create internal/web/static structure
	staticDir := filepath.Join(tmpDir, "internal", "web", "static")
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static directory: %v", err)
	}

	// Create a test file in the static directory
	testFile := filepath.Join(staticDir, "test.css")
	err = os.WriteFile(testFile, []byte("body { color: red; }"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create a file outside static directory (for traversal test)
	secretFile := filepath.Join(tmpDir, "secret.txt")
	err = os.WriteFile(secretFile, []byte("secret data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create secret file: %v", err)
	}

	// Change to temp directory for relative path resolution
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Create test database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	logger := log.New(os.Stdout, "", log.LstdFlags)
	h := handlers.New(db, nil, nil, nil, logger)

	// Setup router with static file serving
	r := chi.NewRouter()
	r.Get("/static/*", h.ServeStatic)

	ts := httptest.NewServer(r)

	cleanup := func() {
		ts.Close()
		db.Close()
		os.Chdir(origDir)
	}

	return ts, cleanup
}

// TestPathTraversalNormalPath verifies normal paths serve files correctly
func TestPathTraversalNormalPath(t *testing.T) {
	ts, cleanup := setupTestServerWithStaticFiles(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/static/test.css")
	if err != nil {
		t.Fatalf("Failed to GET /static/test.css: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for normal path, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	expected := "body { color: red; }"
	if string(body) != expected {
		t.Errorf("Expected body %q, got %q", expected, string(body))
	}
}

// TestPathTraversalSingleDotDot verifies ../ path returns 404
func TestPathTraversalSingleDotDot(t *testing.T) {
	ts, cleanup := setupTestServerWithStaticFiles(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/static/../secret.txt")
	if err != nil {
		t.Fatalf("Failed to GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for ../ traversal, got %d", resp.StatusCode)
	}
}

// TestPathTraversalMultipleDotDot verifies ../../ path returns 404
func TestPathTraversalMultipleDotDot(t *testing.T) {
	ts, cleanup := setupTestServerWithStaticFiles(t)
	defer cleanup()

	resp, err := http.Get(ts.URL + "/static/../../secret.txt")
	if err != nil {
		t.Fatalf("Failed to GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for ../../ traversal, got %d", resp.StatusCode)
	}
}

// TestPathTraversalURLEncoded verifies URL-encoded traversal is blocked
func TestPathTraversalURLEncoded(t *testing.T) {
	ts, cleanup := setupTestServerWithStaticFiles(t)
	defer cleanup()

	// URL-encoded ../ is %2e%2e%2f
	resp, err := http.Get(ts.URL + "/static/%2e%2e%2fsecret.txt")
	if err != nil {
		t.Fatalf("Failed to GET: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404 for URL-encoded traversal, got %d", resp.StatusCode)
	}
}

// TestPathTraversalLogging verifies path traversal attempts are logged
func TestPathTraversalLogging(t *testing.T) {
	// Create temporary test static directory
	tmpDir := t.TempDir()

	// Create internal/web/static structure
	staticDir := filepath.Join(tmpDir, "internal", "web", "static")
	err := os.MkdirAll(staticDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create static directory: %v", err)
	}

	// Create a file outside static directory
	secretFile := filepath.Join(tmpDir, "secret.txt")
	err = os.WriteFile(secretFile, []byte("secret data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create secret file: %v", err)
	}

	// Change to temp directory
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	err = os.Chdir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}
	defer os.Chdir(origDir)

	// Create test database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer db.Close()

	// Create a buffer to capture log output
	var logBuf bytes.Buffer
	logger := log.New(&logBuf, "", 0)

	h := handlers.New(db, nil, nil, nil, logger)

	// Setup router
	r := chi.NewRouter()
	r.Get("/static/*", h.ServeStatic)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Attempt path traversal
	resp, err := http.Get(ts.URL + "/static/../secret.txt")
	if err != nil {
		t.Fatalf("Failed to GET: %v", err)
	}
	defer resp.Body.Close()

	// Check that security log was written
	logOutput := logBuf.String()
	if !strings.Contains(logOutput, "Security:") || !strings.Contains(logOutput, "Path traversal") {
		t.Errorf("Expected security log for path traversal, got: %s", logOutput)
	}
}
