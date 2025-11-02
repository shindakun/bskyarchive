# Research: Security Hardening

**Feature**: 004-security-hardening
**Date**: 2025-11-01
**Status**: Complete

## Overview

This document captures research findings for security hardening implementation. Research focused on Go TLS best practices, CSRF integration with HTMX, OAuth compatibility with strict cookies, Content Security Policy for Pico.css/HTMX, and request size limits.

---

## Research Question 1: TLS Configuration Best Practices

### Question
What are the best practices for Go crypto/tls configuration?
- Minimum TLS version (1.2 vs 1.3)
- Cipher suite selection
- Certificate management approaches

### Findings

#### TLS Version
**Decision**: Use TLS 1.3 as minimum version

**Rationale**:
- TLS 1.3 eliminates weak cipher suites and improves performance
- TLS 1.2 has known vulnerabilities (BEAST, CRIME, POODLE) mitigated in 1.3
- TLS 1.3 handshake is faster (1-RTT vs 2-RTT)
- All modern browsers support TLS 1.3 (95%+ compatibility)
- Go 1.13+ has excellent TLS 1.3 support

**Configuration**:
```go
TLSConfig: &tls.Config{
    MinVersion: tls.VersionTLS13,
    // No cipher suites needed for TLS 1.3 (Go auto-selects secure ones)
}
```

**Fallback Strategy**: Allow TLS 1.2 configuration via config file for legacy clients:
```yaml
tls:
  min_version: "1.3"  # or "1.2" for legacy support
```

#### Cipher Suites (TLS 1.3)
**Decision**: Use Go default cipher suites for TLS 1.3

**Rationale**:
- TLS 1.3 only supports 5 secure cipher suites
- Go automatically selects the best available: TLS_AES_128_GCM_SHA256, TLS_AES_256_GCM_SHA384, TLS_CHACHA20_POLY1305_SHA256
- Explicit configuration unnecessary and error-prone
- Go's defaults align with IANA/IETF recommendations

**Implementation**: No explicit CipherSuites field in TLSConfig for TLS 1.3

#### Certificate Management
**Decision**: Manual certificate loading for Phase 1

**Options Evaluated**:
1. **Manual Certificate Files** (SELECTED)
   - Pros: Simple, explicit control, works with any CA, no dependencies
   - Cons: Manual renewal, requires file management
   - Implementation: Load from `cert_file` and `key_file` config paths

2. **Let's Encrypt (golang.org/x/crypto/acme/autocert)** (Future Enhancement)
   - Pros: Automatic renewal, free certificates, widely trusted
   - Cons: Requires port 80 access, public domain, added complexity
   - Decision: Defer to feature 005-autocert-support

3. **System Certificate Store** (Not Applicable)
   - For client verification only, not server certificates

**Implementation**:
```go
cert, err := tls.LoadX509KeyPair(cfg.Server.TLS.CertFile, cfg.Server.TLS.KeyFile)
if err != nil {
    return fmt.Errorf("failed to load TLS certificate: %w", err)
}

tlsConfig := &tls.Config{
    MinVersion:   tls.VersionTLS13,
    Certificates: []tls.Certificate{cert},
}

srv := &http.Server{
    Addr:      cfg.GetAddr(),
    Handler:   router,
    TLSConfig: tlsConfig,
}
```

### Alternatives Considered

**TLS 1.2 as Minimum**: Rejected due to security concerns, but available as config option
**Custom Cipher Suites**: Rejected; Go defaults are secure and maintained
**Let's Encrypt**: Deferred to future feature; adds significant complexity

### Impact on Design

- Add `TLSConfig` struct to `ServerConfig` in config package
- Support both TLS 1.2 and 1.3 via string configuration ("1.2", "1.3")
- Certificate paths required in config.yaml when TLS enabled
- Graceful error handling for missing/invalid certificates
- Development mode defaults to HTTP (TLS optional)

---

## Research Question 2: CSRF Integration with HTMX

### Question
How to integrate gorilla/csrf with HTMX?
- Token passing in HTMX requests
- Error handling for invalid tokens
- Compatibility with existing forms

### Findings

#### HTMX CSRF Token Integration
**Decision**: Use meta tag + hx-headers for CSRF tokens

**gorilla/csrf Token Generation**:
```go
// In main.go middleware stack
csrfMiddleware := csrf.Protect(
    []byte(cfg.OAuth.SessionSecret),
    csrf.Secure(cfg.Server.TLS.Enabled), // Only secure=true with HTTPS
    csrf.FieldName("csrf_token"),
    csrf.RequestHeader("X-CSRF-Token"), // HTMX uses this
)
r.Use(csrfMiddleware)
```

**HTML Template Integration**:
```html
<head>
    <meta name="csrf-token" content="{{ .csrfToken }}">
</head>

<body hx-headers='{"X-CSRF-Token": "{{ .csrfToken }}"}'>
    <!-- All HTMX requests in body will include token -->
</body>
```

**Form Integration** (Non-HTMX):
```html
<form method="POST" action="/archive/start">
    <input type="hidden" name="csrf_token" value="{{ .csrfToken }}">
    <!-- form fields -->
</form>
```

#### Error Handling
**gorilla/csrf Error Response**:
- Returns 403 Forbidden for invalid tokens
- Provides FailureHandler option for custom error pages

**HTMX Compatibility**:
```go
// Custom failure handler for better UX
csrfFailure := func(w http.ResponseWriter, r *http.Request) {
    if r.Header.Get("HX-Request") == "true" {
        // HTMX request - return HTML fragment
        w.WriteHeader(http.StatusForbidden)
        w.Write([]byte(`<div class="error">Session expired. Please refresh the page.</div>`))
    } else {
        // Regular request - redirect to error page
        http.Error(w, "CSRF token invalid", http.StatusForbidden)
    }
}

csrfMiddleware := csrf.Protect(
    []byte(secret),
    csrf.ErrorHandler(http.HandlerFunc(csrfFailure)),
)
```

#### Template Helper
**Token Availability in Templates**:
```go
// In handler
func (h *Handlers) renderTemplate(w http.ResponseWriter, name string, data TemplateData) error {
    data.CSRFToken = csrf.Token(r) // gorilla/csrf helper
    return tmpl.ExecuteTemplate(w, name, data)
}

// Template data structure
type TemplateData struct {
    Session    *models.Session
    CSRFToken  string // Available in all templates
    // ... other fields
}
```

### Alternatives Considered

**Double Submit Cookie**: Rejected; gorilla/csrf server-side tokens are more secure
**Custom CSRF Implementation**: Rejected; gorilla/csrf is battle-tested
**Per-Request Token in URL**: Rejected; tokens in URL are logged and cached

### Impact on Design

- Add `CSRFToken` field to `TemplateData` struct
- Middleware wrapper in `internal/web/middleware/csrf.go`
- Base template updated with meta tag and hx-headers
- Custom failure handler for HTMX compatibility
- All POST endpoints automatically protected

---

## Research Question 3: OAuth Callback with SameSite=Strict

### Question
Can OAuth callback work with SameSite=Strict cookies?
- OAuth redirect flow implications
- Fallback to SameSite=Lax if needed

### Findings

#### SameSite Cookie Behavior
**SameSite=Strict**: Cookie NOT sent on cross-site navigation (including OAuth redirects)
**SameSite=Lax**: Cookie sent on top-level GET navigation (OAuth redirect compatible)
**SameSite=None**: Cookie sent on all cross-site requests (requires Secure flag)

#### OAuth Flow with Cookies
1. User clicks "Login" on bskyarchive.com
2. Redirected to Bluesky OAuth provider (bsky.app)
3. User authorizes on bsky.app
4. Redirected back to bskyarchive.com/callback with authorization code
5. **Problem**: Session cookie not sent if SameSite=Strict

#### Testing Results
**SameSite=Strict**: ❌ Session cookie not available in OAuth callback
- OAuth state parameter not verified (requires session)
- Re-login required after OAuth authorization

**SameSite=Lax**: ✅ Session cookie available in OAuth callback
- OAuth state parameter properly verified
- Seamless login experience
- Still protects against most CSRF attacks

**Decision**: Use SameSite=Lax for session cookies

**Rationale**:
- OAuth callback requires session cookie for state verification
- SameSite=Lax provides CSRF protection for POST/PUT/DELETE
- Top-level GET navigation (OAuth redirect) is safe with SameSite=Lax
- Combined with CSRF tokens on POST endpoints provides defense-in-depth

**Additional CSRF Protection**:
- OAuth library (bskyoauth) implements state parameter validation
- CSRF tokens protect all POST endpoints
- Both mechanisms together provide strong CSRF protection

### Alternatives Considered

**SameSite=None**: Rejected; weakens security unnecessarily
**Separate OAuth Cookie**: Rejected; adds complexity, same SameSite issue
**State in URL Parameter**: Rejected; bskyoauth handles state internally

### Impact on Design

- Keep SameSite=Lax (current configuration)
- Document why Strict is not used (OAuth compatibility)
- Rely on CSRF tokens + OAuth state for complete CSRF protection
- Update security audit documentation with justification

---

## Research Question 4: Content Security Policy for Pico.css + HTMX

### Question
What CSP directives are needed for Pico.css + HTMX?
- Inline script requirements
- External resource loading
- Nonce-based CSP vs allowlist

### Findings

#### Current Application Analysis
**Pico.css**: CSS framework, no inline styles or scripts
**HTMX**: JavaScript library with attributes (e.g., hx-get, hx-post)
**Current Usage**: Minimal inline JavaScript, mostly HTMX attributes

#### CSP Directive Requirements
**Script Sources**:
- HTMX loaded from `/static/js/htmx.min.js` (self)
- No inline `<script>` tags found in templates
- **Decision**: `script-src 'self'`

**Style Sources**:
- Pico.css loaded from `/static/css/pico.min.css` (self)
- Minimal inline styles in templates
- **Decision**: `style-src 'self' 'unsafe-inline'` (Pico uses inline styles in some cases)

**Frame Sources**:
- No iframes used
- **Decision**: `frame-ancestors 'none'` (equivalent to X-Frame-Options: DENY)

**Default Sources**:
- All resources served from same origin
- **Decision**: `default-src 'self'`

#### Recommended CSP Policy (Phase 1)
```
Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'
```

**Directive Breakdown**:
- `default-src 'self'`: All resources from same origin by default
- `script-src 'self'`: JavaScript only from same origin
- `style-src 'self' 'unsafe-inline'`: Styles from same origin + inline (Pico.css requirement)
- `img-src 'self' data:`: Images from same origin + data URIs (for inline images)
- `font-src 'self'`: Fonts from same origin
- `frame-ancestors 'none'`: Prevent embedding in iframes
- `base-uri 'self'`: Restrict base tag to same origin
- `form-action 'self'`: Forms can only submit to same origin

#### HTMX Compatibility
**HTMX and CSP**: HTMX works with strict CSP; no inline event handlers
- HTMX uses `hx-*` attributes, not inline JavaScript
- All HTMX scripts loaded from static files
- **Decision**: No special CSP accommodations needed for HTMX

#### Nonce vs Allowlist
**Nonce-based CSP**:
- Pros: Strongest protection, allows inline scripts with nonce
- Cons: Requires template modification, nonce generation per request

**Allowlist CSP**:
- Pros: Simpler, no template changes needed
- Cons: Less protection against inline script injection

**Decision**: Start with allowlist CSP
- Current application has no inline scripts
- Simpler implementation for Phase 1
- Can upgrade to nonce-based CSP in future if inline scripts added

### Alternatives Considered

**No CSP**: Rejected; CSP provides significant XSS protection
**Permissive CSP (unsafe-eval, unsafe-inline)**: Rejected; defeats purpose of CSP
**Nonce-based CSP**: Deferred to future enhancement; allowlist sufficient for current app

### Impact on Design

- Add CSP header in SecurityHeaders middleware
- Configuration field for CSP in config.yaml
- Document CSP policy and rationale
- Test CSP with browser developer tools (CSP violation reports)
- Monitor for CSP violations in production (future: CSP reporting)

---

## Research Question 5: Request Size Limits

### Question
What's an appropriate limit for POST requests?
- Form data size estimation
- File upload considerations
- Memory impact analysis

### Findings

#### Current Application Analysis
**POST Endpoints**:
1. `/auth/login` - Handle input (50-100 bytes)
2. `/archive/start` - Operation type (20 bytes)
3. `/export/start` - Format, date range, options (~200 bytes)

**Maximum Expected Request Size**:
- Form data: ~1KB per request
- No file uploads currently supported
- No large JSON payloads

#### Request Size Limit Recommendations
**Industry Standards**:
- **nginx default**: 1MB (`client_max_body_size`)
- **Apache default**: Unlimited (bad practice)
- **AWS ALB**: 1MB (can't be changed)
- **Cloudflare**: 100MB free tier, 500MB paid

**Web Application Best Practices**:
- **Small forms** (login, settings): 1-10KB
- **Medium forms** (blog post, comment): 100KB-1MB
- **File uploads**: 10MB-100MB+

**Decision**: 10MB limit for bskyarchive

**Rationale**:
- Current application: <1KB per request
- 10MB provides significant margin for future features
- Prevents DoS attacks via large payloads
- Low enough to prevent memory exhaustion
- High enough to not interfere with legitimate use

#### Memory Impact Analysis
**Go http.MaxBytesReader**:
- Reads request body up to limit
- Returns error if limit exceeded
- Does not buffer entire request in memory (streaming)
- **Memory usage**: O(buffer size), not O(request size)

**Concurrent Request Impact**:
- 100 concurrent 10MB requests = 1GB theoretical max
- Actual memory usage much lower (streaming + buffering)
- Go's HTTP server efficiently handles concurrent requests

**Implementation**:
```go
func MaxBytesMiddleware(maxBytes int64) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
            next.ServeHTTP(w, r)
        })
    }
}
```

#### Error Handling
**MaxBytesReader Behavior**:
- Returns `http.ErrBodyTooLarge` when limit exceeded
- Handler must check for this error

**User-Friendly Response**:
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

### Alternatives Considered

**1MB Limit**: Rejected; too restrictive for future features
**100MB Limit**: Rejected; unnecessary, higher DoS risk
**No Limit**: Rejected; vulnerable to DoS attacks
**Per-Endpoint Limits**: Rejected; unnecessary complexity for current app

### Impact on Design

- Add `max_request_bytes` field to server security config
- Default: 10MB (10485760 bytes)
- Configurable via config.yaml
- Middleware applied globally to all routes
- Proper error handling with 413 status code

---

## Technology Decisions Summary

| Decision Point | Selected Option | Rationale |
|----------------|-----------------|-----------|
| **TLS Version** | TLS 1.3 minimum | Modern security, better performance, wide support |
| **Cipher Suites** | Go defaults | Secure, maintained, aligned with standards |
| **Certificate Management** | Manual (Phase 1) | Simple, explicit, works with any CA |
| **CSRF Library** | gorilla/csrf | Industry standard, well-maintained, HTMX compatible |
| **CSRF Token Passing** | Meta tag + hx-headers | Clean, works with HTMX and forms |
| **SameSite Cookie** | Lax | OAuth compatibility, still provides CSRF protection |
| **CSP Policy** | Allowlist-based | Simple, no inline scripts, sufficient for current app |
| **CSP Directives** | `default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; ...` | Strict but compatible with Pico.css + HTMX |
| **Request Size Limit** | 10MB | Balance between security and future flexibility |

---

## Implementation Recommendations

### High Priority
1. **TLS Configuration**: Start with TLS 1.3, allow 1.2 fallback via config
2. **CSRF Protection**: Implement immediately for all POST endpoints
3. **Security Headers**: Deploy CSP and other headers together

### Configuration Strategy
- All security settings in config.yaml under `server.security`
- Environment-based defaults (dev vs prod)
- Clear error messages for misconfiguration
- Startup validation and warnings

### Testing Strategy
1. **TLS**: Connect with TLS 1.3 client, verify cipher suites
2. **CSRF**: Attempt POST without token, verify 403 response
3. **CSP**: Browser developer tools CSP violation check
4. **Size Limits**: Send 11MB request, verify 413 response

### Documentation Needs
1. **Configuration Guide**: TLS setup with example certificates
2. **Security Hardening Guide**: Deployment best practices
3. **CSRF Guide**: How to add CSRF tokens to new forms
4. **CSP Guide**: How to update CSP when adding new resources

---

## Open Questions Resolved

1. ✅ **Let's Encrypt Support**: Defer to future feature
2. ✅ **SameSite=Strict**: Not compatible with OAuth; use Lax
3. ✅ **CSP Nonce**: Start with allowlist; nonces not needed yet
4. ✅ **Request Size**: 10MB provides good balance

---

## Next Steps

1. Proceed to Phase 1: Design
   - Generate contracts for middleware behavior
   - Create quickstart guide for security configuration
   - Update agent context with new security technologies

2. Implement based on research findings:
   - TLS with manual certificate loading
   - gorilla/csrf with HTMX integration
   - Security headers with CSP policy
   - Request size middleware with 10MB limit
   - Path traversal protection for static files

**Research Status**: ✅ COMPLETE - All questions resolved, ready for implementation
