# Security Middleware Contracts

**Feature**: 004-security-hardening
**Date**: 2025-11-01

## Overview

This document defines the behavioral contracts for security middleware components. Each middleware must satisfy these contracts to ensure consistent security posture across the application.

---

## 1. SecurityHeaders Middleware

### Purpose
Set HTTP security headers on all responses to protect against XSS, clickjacking, MIME sniffing, and other browser-based attacks.

### Contract

**Signature**:
```go
func SecurityHeaders(cfg *config.ServerConfig) func(http.Handler) http.Handler
```

**Input**:
- `cfg`: Server configuration containing security header settings

**Behavior**:
1. MUST set headers on every response, regardless of status code
2. MUST read header values from configuration
3. MUST set X-Frame-Options header (default: "DENY")
4. MUST set X-Content-Type-Options header (default: "nosniff")
5. MUST set X-XSS-Protection header (default: "1; mode=block")
6. MUST set Referrer-Policy header (default: "strict-origin-when-cross-origin")
7. MUST set Content-Security-Policy header (default: "default-src 'self'")
8. MUST set Strict-Transport-Security header when TLS enabled (default: "max-age=31536000; includeSubDomains")
9. MUST NOT set HSTS header when TLS disabled
10. MUST pass request to next handler unchanged

**Output**:
- HTTP response with security headers added
- No modification to request or response body

**Error Handling**:
- MUST NOT fail or block request if header configuration invalid
- SHOULD log warning for invalid header values
- MUST use default values if configuration missing

**Performance**:
- MUST complete in <1ms per request
- MUST NOT allocate significant memory

### Example Usage

```go
// In main.go
r.Use(middleware.SecurityHeaders(cfg.Server))
```

**Expected Headers**:
```
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'
Strict-Transport-Security: max-age=31536000; includeSubDomains (only when TLS enabled)
```

### Test Cases

1. **TC-SH-001**: Headers present on 200 OK response
2. **TC-SH-002**: Headers present on 404 Not Found response
3. **TC-SH-003**: Headers present on 500 Internal Server Error response
4. **TC-SH-004**: HSTS header present when TLS enabled
5. **TC-SH-005**: HSTS header absent when TLS disabled
6. **TC-SH-006**: Custom CSP from configuration applied
7. **TC-SH-007**: Default headers used when configuration missing

---

## 2. MaxBytesMiddleware

### Purpose
Limit request body size to prevent denial-of-service attacks via large payloads.

### Contract

**Signature**:
```go
func MaxBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler
```

**Input**:
- `maxBytes`: Maximum allowed request body size in bytes

**Behavior**:
1. MUST wrap request body with `http.MaxBytesReader`
2. MUST apply limit to all HTTP methods (GET, POST, PUT, DELETE, etc.)
3. MUST apply limit before next handler executes
4. MUST NOT read or modify request body
5. MUST pass request to next handler with wrapped body
6. Handler reading body will receive error if limit exceeded
7. Error MUST be `http.ErrBodyTooLarge` or similar

**Output**:
- Request with body wrapped in MaxBytesReader
- No modification to response

**Error Handling**:
- MUST NOT handle errors directly (delegate to handler)
- Handler MUST check for body too large error
- Handler SHOULD return 413 Request Entity Too Large

**Performance**:
- MUST complete in <0.1ms per request
- MUST NOT allocate significant memory
- MUST support streaming (not buffer entire body)

### Example Usage

```go
// In main.go
r.Use(middleware.MaxBytesMiddleware(10 * 1024 * 1024)) // 10MB
```

**Handler Error Handling**:
```go
func (h *Handlers) StartExport(w http.ResponseWriter, r *http.Request) {
    if err := r.ParseForm(); err != nil {
        if strings.Contains(err.Error(), "request body too large") {
            http.Error(w, "Request too large. Maximum size is 10MB.", http.StatusRequestEntityTooLarge)
            return
        }
        http.Error(w, "Invalid form data", http.StatusBadRequest)
        return
    }
    // ... normal handling
}
```

### Test Cases

1. **TC-MB-001**: Request under limit proceeds normally
2. **TC-MB-002**: Request at exactly maxBytes proceeds normally
3. **TC-MB-003**: Request over limit returns 413 error
4. **TC-MB-004**: Large POST body (11MB) rejected with 413
5. **TC-MB-005**: Streaming request properly limited
6. **TC-MB-006**: GET request with no body not affected

---

## 3. CSRF Middleware

### Purpose
Protect state-changing operations (POST, PUT, DELETE) from cross-site request forgery attacks.

### Contract

**Signature**:
```go
func CSRFMiddleware(cfg *config.ServerConfig) func(http.Handler) http.Handler
```

**Input**:
- `cfg`: Server configuration containing CSRF settings

**Behavior**:
1. MUST generate unique token per session
2. MUST validate token on POST, PUT, DELETE, PATCH requests
3. MUST NOT validate token on GET, HEAD, OPTIONS requests
4. MUST accept token in form field (name from config)
5. MUST accept token in request header (X-CSRF-Token)
6. MUST return 403 Forbidden for invalid/missing token
7. MUST provide token to handlers via `csrf.Token(r)`
8. MUST respect Secure flag based on TLS configuration
9. MUST use session secret for token generation
10. MUST call custom failure handler if provided

**Output**:
- Token validation result (pass/fail)
- 403 response for failed validation
- Token available to handler via helper function

**Error Handling**:
- Invalid token: 403 Forbidden with clear error message
- Missing token: 403 Forbidden with clear error message
- Custom failure handler: Execute custom handler
- HTMX request: Return HTML fragment error message

**Performance**:
- MUST complete in <2ms per request
- Token generation: <1ms per session
- Token validation: <1ms per request

### Example Usage

```go
// In main.go
csrfMiddleware := csrf.Protect(
    []byte(cfg.OAuth.SessionSecret),
    csrf.Secure(cfg.Server.TLS.Enabled),
    csrf.FieldName("csrf_token"),
    csrf.RequestHeader("X-CSRF-Token"),
    csrf.ErrorHandler(http.HandlerFunc(middleware.CSRFFailureHandler)),
)
r.Use(csrfMiddleware)
```

**Template Integration**:
```html
<head>
    <meta name="csrf-token" content="{{ .CSRFToken }}">
</head>

<body hx-headers='{"X-CSRF-Token": "{{ .CSRFToken }}"}'>
    <form method="POST" action="/export/start">
        <input type="hidden" name="csrf_token" value="{{ .CSRFToken }}">
        <!-- form fields -->
        <button type="submit">Export</button>
    </form>
</body>
```

**Handler Token Access**:
```go
func (h *Handlers) renderTemplate(w http.ResponseWriter, r *http.Request, name string, data TemplateData) error {
    data.CSRFToken = csrf.Token(r)
    return tmpl.ExecuteTemplate(w, name, data)
}
```

### Test Cases

1. **TC-CSRF-001**: GET request proceeds without token
2. **TC-CSRF-002**: POST with valid token succeeds
3. **TC-CSRF-003**: POST without token returns 403
4. **TC-CSRF-004**: POST with invalid token returns 403
5. **TC-CSRF-005**: Token in form field validated correctly
6. **TC-CSRF-006**: Token in X-CSRF-Token header validated correctly
7. **TC-CSRF-007**: HTMX request failure returns HTML fragment
8. **TC-CSRF-008**: Regular request failure returns error page
9. **TC-CSRF-009**: Token changes per session
10. **TC-CSRF-010**: Token survives session refresh

---

## 4. Path Traversal Protection

### Purpose
Prevent unauthorized file access via path traversal attacks (../, ../../, etc.)

### Contract

**Function**: `ServeStatic` and `ServeMedia` handlers

**Behavior**:
1. MUST clean user-provided path with `filepath.Clean`
2. MUST resolve to absolute path
3. MUST validate resolved path is within allowed directory
4. MUST use string prefix check for validation
5. MUST return 404 Not Found for invalid paths
6. MUST NOT disclose file system structure in errors
7. MUST log path traversal attempts

**Input Validation**:
```go
func (h *Handlers) ServeStatic(w http.ResponseWriter, r *http.Request) {
    // Extract path from URL
    path := chi.URLParam(r, "*")

    // Clean path (removes ../)
    cleanPath := filepath.Clean(path)

    // Get absolute path to static directory
    staticDir, _ := filepath.Abs("internal/web/static")

    // Construct full path
    fullPath := filepath.Join(staticDir, cleanPath)

    // Validate path is within static directory
    if !strings.HasPrefix(fullPath, staticDir) {
        http.NotFound(w, r)
        h.logger.Printf("Path traversal attempt blocked: %s", path)
        return
    }

    // Serve file
    http.ServeFile(w, r, fullPath)
}
```

**Output**:
- File content if path valid
- 404 Not Found if path invalid or file doesn't exist
- Log entry for security audit

**Error Handling**:
- MUST return generic 404 for all failures
- MUST NOT disclose if file exists or path is invalid
- MUST log suspicious patterns for monitoring

### Test Cases

1. **TC-PT-001**: Normal path `/static/css/style.css` serves file
2. **TC-PT-002**: Path `/../../../etc/passwd` returns 404
3. **TC-PT-003**: Path `/static/../../../etc/passwd` returns 404
4. **TC-PT-004**: Path `/static/%2e%2e/%2e%2e/etc/passwd` returns 404 (URL encoded)
5. **TC-PT-005**: Path traversal attempt logged
6. **TC-PT-006**: Non-existent file returns 404 (same as traversal)
7. **TC-PT-007**: Symlink outside static directory blocked

---

## 5. Export Directory Isolation

### Purpose
Ensure users can only access their own export files, preventing unauthorized access to other users' data.

### Contract

**Functions**: `CreateExportDirectory`, `ExportProgress`

**Behavior**:
1. MUST create exports in per-user subdirectories: `exports/{did}/`
2. MUST include DID in export job structure
3. MUST verify job ownership before serving progress/files
4. MUST return 403 Forbidden for unauthorized access
5. MUST NOT disclose existence of other users' exports

**CreateExportDirectory**:
```go
func CreateExportDirectory(baseDir string, did string) (string, error) {
    timestamp := time.Now().Format("2006-01-02_15-04-05")

    // Create per-user directory
    userDir := filepath.Join(baseDir, did)
    exportDir := filepath.Join(userDir, timestamp)

    if err := os.MkdirAll(exportDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create export directory: %w", err)
    }

    return exportDir, nil
}
```

**ExportProgress Authorization**:
```go
func (h *Handlers) ExportProgress(w http.ResponseWriter, r *http.Request) {
    session, _ := auth.GetSessionFromContext(r.Context())
    jobID := chi.URLParam(r, "job_id")

    // Get job
    job, exists := exportJobs[jobID]
    if !exists {
        http.NotFound(w, r)
        return
    }

    // Verify ownership
    if job.Options.DID != session.DID {
        http.Error(w, "Forbidden", http.StatusForbidden)
        h.logger.Printf("Export access denied: user=%s job_did=%s", session.DID, job.Options.DID)
        return
    }

    // ... serve progress
}
```

### Test Cases

1. **TC-ED-001**: User can access their own exports
2. **TC-ED-002**: User cannot access other user's exports (403)
3. **TC-ED-003**: Export created in correct per-user directory
4. **TC-ED-004**: Job ID guess returns 404 if not owned
5. **TC-ED-005**: Directory listing blocked
6. **TC-ED-006**: Unauthorized access logged

---

## Middleware Chain Order

The middleware MUST be applied in this specific order to ensure security controls are properly enforced:

```
1. RequestID        - Assign unique ID for logging
2. RealIP           - Extract client IP
3. SecurityHeaders  - Set security headers (affects all responses)
4. MaxBytesMiddleware - Limit request body size
5. CSRF             - Validate CSRF tokens (POST/PUT/DELETE only)
6. Logging          - Log request details (after security checks)
7. Recoverer        - Panic recovery
8. Timeout          - Request timeout
9. Router           - Route to handler
```

**Rationale**:
- SecurityHeaders first ensures all responses (including errors) have security headers
- MaxBytesMiddleware before CSRF prevents large payloads before token validation
- CSRF before logging ensures failed CSRF attempts are logged
- Logging after security checks captures full security context

---

## Contract Verification

Each middleware MUST pass the following verification checks:

### Security Checklist
- [ ] Does not log sensitive data (tokens, passwords, session IDs)
- [ ] Does not expose internal paths or configuration in errors
- [ ] Uses constant-time comparison for security-sensitive checks
- [ ] Properly handles edge cases (empty body, missing headers, etc.)
- [ ] Thread-safe (no shared mutable state)

### Performance Checklist
- [ ] Completes within specified time limit
- [ ] Does not allocate excessive memory
- [ ] Supports concurrent requests efficiently
- [ ] Does not block on I/O operations

### Compatibility Checklist
- [ ] Works with existing authentication system
- [ ] Compatible with HTMX requests
- [ ] Handles both HTML and JSON responses
- [ ] Backwards compatible with existing endpoints

---

## Compliance

These contracts align with:
- **OWASP Top 10 (2021)**: A01 (Access Control), A03 (Injection), A04 (Insecure Design), A05 (Security Misconfiguration)
- **NIST Cybersecurity Framework**: PR.AC (Access Control), PR.DS (Data Security)
- **Constitution Principles**: Security & Privacy (section II), Development Standards (section III)

**Contract Status**: ✅ DEFINED - Ready for implementation
