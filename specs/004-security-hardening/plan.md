# Implementation Plan: Security Hardening

**Branch**: `004-security-hardening` | **Date**: 2025-11-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/004-security-hardening/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

This feature implements critical security controls identified in the 2025-11-01 security audit. Primary requirements include HTTPS/TLS support, CSRF protection, secure cookie configuration, HTTP security headers, request size limits, and path traversal protection. The implementation adds security middleware to the existing HTTP server, upgrades cookie settings, and hardens file serving endpoints to protect against common web vulnerabilities (CSRF, XSS, clickjacking, path traversal, DoS).

## Technical Context

**Language/Version**: Go 1.25.3 (existing project standard)
**Primary Dependencies**:
- `crypto/tls` (Go stdlib) - TLS configuration
- `github.com/gorilla/csrf v1.7.3` - CSRF protection (already imported)
- `github.com/go-chi/chi/v5` - HTTP router (existing)
**Storage**: SQLite (existing - no changes needed)
**Testing**: Go testing package, integration tests
**Target Platform**: Linux/macOS/Windows servers (existing)
**Project Type**: Single web application
**Performance Goals**:
- Security middleware overhead <5ms per request
- TLS handshake using standard TLS 1.3 performance
- No impact on existing endpoint latency
**Constraints**:
- Must maintain HTTP mode for development
- Must not break existing OAuth flow
- HTMX compatibility required for CSRF
- Backward compatible configuration
**Scale/Scope**:
- 7 new/modified files in middleware layer
- 3 configuration file updates
- 8-12 integration tests
- ~500 lines of new security code

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Security & Privacy Principles

✅ **PASS**: "OAuth 2.0 flow using bskyoauth library for secure authentication"
- This feature enhances the existing OAuth implementation with CSRF protection

✅ **PASS**: "Secure session management with token refresh handling"
- This feature upgrades session cookies to use Secure flag and SameSite=Strict

✅ **PASS**: "Encrypt sensitive data at rest; use system keyring where possible"
- This feature adds transport encryption via TLS (data-in-transit)

✅ **PASS**: "CSRF protection on all state-changing operations"
- This feature implements CSRF protection using gorilla/csrf

✅ **PASS**: "Rate limiting on API endpoints to prevent abuse"
- Request size limits added; endpoint rate limiting deferred to future feature

✅ **PASS**: "No credential storage in plaintext"
- No changes to credential storage; TLS certificates stored separately

### Development Standards

✅ **PASS**: "Go 1.21+ with standard library practices"
- Uses crypto/tls from Go stdlib, gorilla/csrf (established library)

✅ **PASS**: "Clear separation of concerns"
- Security controls isolated in internal/web/middleware/

✅ **PASS**: "Comprehensive error handling with retry logic"
- TLS errors properly handled, CSRF validation with clear error messages

✅ **PASS**: "Configuration management via YAML/JSON files"
- All security settings configurable via config.yaml

### Testing Requirements

✅ **PASS**: "Integration tests for AT Protocol interactions"
- OAuth flow tested with new security controls

✅ **PASS**: "Contract tests for API endpoints"
- CSRF protection validated on POST endpoints

✅ **PASS**: "Performance benchmarks for search and export operations"
- Middleware performance benchmarked (<5ms target)

### Documentation Requirements

✅ **PASS**: "User-facing documentation for CLI commands"
- Configuration examples for TLS and security settings

✅ **PASS**: "Configuration examples and best practices"
- Security hardening guide with deployment recommendations

**GATE RESULT**: ✅ ALL CHECKS PASSED - Proceed to Phase 0

## Project Structure

### Documentation (this feature)

```text
specs/004-security-hardening/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command) - N/A for security feature
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command) - Security middleware contracts
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── config/
│   └── config.go                   # Modified: Add TLS configuration fields
├── auth/
│   └── session.go                  # Modified: Secure cookie configuration
└── web/
    ├── middleware/
    │   ├── auth.go                 # Existing
    │   ├── errors.go               # Existing
    │   ├── logging.go              # Existing
    │   ├── security.go             # NEW: Security headers middleware
    │   ├── csrf.go                 # NEW: CSRF middleware wrapper
    │   └── maxbytes.go             # NEW: Request size limit middleware
    └── handlers/
        ├── handlers.go             # Modified: Fix static file serving path traversal
        └── export.go               # Modified: Per-user export directory isolation

cmd/bskyarchive/
└── main.go                         # Modified: Add middleware stack, TLS config

config.yaml                          # Modified: Add security configuration section

tests/
├── integration/
│   ├── security_test.go            # NEW: Security integration tests
│   ├── csrf_test.go                # NEW: CSRF protection tests
│   └── tls_test.go                 # NEW: TLS configuration tests
└── unit/
    └── middleware_test.go          # NEW: Middleware unit tests
```

**Structure Decision**: Single project structure (existing). Security controls implemented as middleware in `internal/web/middleware/`. Configuration extended in `internal/config/`. No new top-level directories or services required. All changes integrate into existing HTTP server architecture.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

N/A - No constitution violations. All requirements align with existing security principles.

## Phase 0: Research & Technology Decisions

### Research Questions

1. **TLS Configuration**: Best practices for Go crypto/tls configuration?
   - Minimum TLS version (1.2 vs 1.3)
   - Cipher suite selection
   - Certificate management approaches (Let's Encrypt vs manual)

2. **CSRF Integration**: How to integrate gorilla/csrf with HTMX?
   - Token passing in HTMX requests
   - Error handling for invalid tokens
   - Compatibility with existing forms

3. **Cookie SameSite**: Can OAuth callback work with SameSite=Strict?
   - OAuth redirect flow implications
   - Fallback to SameSite=Lax if needed
   - Testing strategy

4. **Content Security Policy**: What CSP directives are needed for Pico.css + HTMX?
   - Inline script requirements
   - External resource loading
   - Nonce-based CSP vs allowlist

5. **Request Size Limits**: What's appropriate limit for POST requests?
   - Form data size estimation
   - File upload considerations (not currently supported)
   - Memory impact analysis

### Technology Decisions

*To be filled during research phase*

## Phase 1: Design

### Middleware Architecture

```
HTTP Request
    ↓
[RequestID] → assigns unique ID
    ↓
[RealIP] → extracts client IP
    ↓
[SecurityHeaders] → sets X-Frame-Options, CSP, etc. (NEW)
    ↓
[MaxBytesMiddleware] → limits request body size (NEW)
    ↓
[CSRF] → validates tokens on POST/PUT/DELETE (NEW)
    ↓
[Logging] → logs request details
    ↓
[Recoverer] → panic recovery
    ↓
[Timeout] → request timeout
    ↓
[Router] → route to handler
```

### Configuration Schema

```yaml
server:
  port: 8080
  host: "localhost"
  base_url: "https://your-domain.com"

  # NEW: TLS configuration
  tls:
    enabled: true                    # Enable HTTPS
    cert_file: "./certs/cert.pem"    # Path to TLS certificate
    key_file: "./certs/key.pem"      # Path to TLS private key
    min_version: "1.3"               # Minimum TLS version (1.2 or 1.3)
    # auto_cert: false               # Future: Let's Encrypt support

  # NEW: Security configuration
  security:
    csrf_enabled: true               # Enable CSRF protection
    csrf_field_name: "csrf_token"    # Form field name for CSRF token
    max_request_bytes: 10485760      # 10MB request size limit

    # Security headers
    headers:
      x_frame_options: "DENY"
      x_content_type_options: "nosniff"
      x_xss_protection: "1; mode=block"
      referrer_policy: "strict-origin-when-cross-origin"
      content_security_policy: "default-src 'self'"
      # HSTS only when TLS enabled
      strict_transport_security: "max-age=31536000; includeSubDomains"

  read_timeout: 15s
  write_timeout: 15s
  idle_timeout: 60s
  shutdown_timeout: 30s

oauth:
  scopes: ["atproto", "transition:generic", "transition:chat.bsky"]
  session_secret: "${SESSION_SECRET}"
  session_max_age: 604800
  # NEW: Cookie security (auto-configured based on TLS)
  cookie_secure: "auto"              # auto/true/false
  cookie_samesite: "strict"          # strict/lax/none
```

### API Contracts

*Middleware behavior contracts to be documented in contracts/*

## Phase 2: Implementation Plan

### Phase 2.1: TLS Support (CRITICAL)
**Files**: `cmd/bskyarchive/main.go`, `internal/config/config.go`

1. Add TLS configuration struct to config
2. Load TLS certificates from config paths
3. Configure TLS with minimum version and cipher suites
4. Add environment detection (dev/prod)
5. Implement graceful fallback for missing certificates
6. Add startup logging for TLS status

### Phase 2.2: Security Middleware (CRITICAL)
**Files**: `internal/web/middleware/security.go`, `internal/web/middleware/maxbytes.go`, `internal/web/middleware/csrf.go`

1. Create SecurityHeaders middleware
2. Create MaxBytesMiddleware
3. Create CSRF middleware wrapper for gorilla/csrf
4. Add middleware to router stack in main.go
5. Configure middleware from config.yaml

### Phase 2.3: Secure Cookies (CRITICAL)
**Files**: `internal/auth/session.go`, `internal/config/config.go`

1. Add cookie security configuration to config
2. Implement environment detection helper
3. Set Secure flag based on TLS configuration
4. Upgrade SameSite from Lax to Strict
5. Test OAuth flow with new settings

### Phase 2.4: Path Traversal Protection (HIGH)
**Files**: `internal/web/handlers/handlers.go`

1. Fix ServeStatic method (line 406)
2. Add absolute path resolution
3. Validate resolved path is within static directory
4. Add similar protection to ServeMedia endpoint
5. Test with path traversal attempts

### Phase 2.5: Export Directory Isolation (MEDIUM)
**Files**: `internal/web/handlers/export.go`, `internal/exporter/exporter.go`

1. Update CreateExportDirectory to create per-DID subdirectories
2. Add ownership verification in ExportProgress handler
3. Update export job structure with DID field
4. Test cross-user export access

### Phase 2.6: Testing & Documentation (HIGH)
**Files**: `tests/integration/security_test.go`, `tests/integration/csrf_test.go`, `tests/unit/middleware_test.go`

1. TLS connection tests
2. CSRF protection tests
3. Security header verification tests
4. Path traversal attempt tests
5. Export isolation tests
6. Middleware performance benchmarks
7. Configuration documentation
8. Deployment guide updates

## Open Questions

1. **Let's Encrypt Support**: Should Phase 1 include automatic certificate management?
   - **Decision Needed Before**: Phase 2.1 implementation
   - **Impact**: Complexity of TLS configuration
   - **Recommendation**: Defer to future feature; manual certificates sufficient for v1

2. **HSTS Preload**: Should we support HSTS preload list submission?
   - **Decision Needed Before**: Phase 2.2 implementation
   - **Impact**: Header configuration
   - **Recommendation**: No; users can enable manually if needed

3. **CSP Nonce**: Use nonce-based CSP or allowlist?
   - **Decision Needed Before**: Phase 2.2 implementation
   - **Impact**: Template rendering, HTMX compatibility
   - **Recommendation**: Start with allowlist; upgrade to nonces if needed

## Dependencies

### Internal Dependencies
- Existing HTTP server architecture (`cmd/bskyarchive/main.go`)
- Router middleware chain (`github.com/go-chi/chi/v5`)
- Configuration system (`internal/config/`)
- Authentication system (`internal/auth/`)

### External Dependencies
- `crypto/tls` (Go stdlib) - already available
- `github.com/gorilla/csrf v1.7.3` - already imported
- No new dependencies required

### Dependency Risks
- **Low Risk**: All dependencies are stable, well-maintained libraries
- **CSRF Library**: gorilla/csrf is industry-standard Go CSRF library
- **TLS**: Using Go stdlib reduces supply chain risk

## Success Metrics

1. **Security Controls Active**: All security middleware deployed and active in production
2. **Zero CSRF Vulnerabilities**: All POST endpoints protected with valid token validation
3. **TLS Configured**: Server accepts TLS 1.3 connections with secure ciphers
4. **Headers Present**: Security headers present on 100% of responses
5. **Path Traversal Blocked**: Traversal attempts return 404/403
6. **Performance Target Met**: Middleware adds <5ms per request
7. **Zero Regressions**: All existing tests pass
8. **OAuth Flow Intact**: OAuth authentication works with new security controls

## Next Steps

1. Run `/speckit.plan` command to generate:
   - `research.md` (Phase 0 research findings)
   - `contracts/` (Middleware behavior contracts)
   - `quickstart.md` (Security configuration quick start)

2. After planning complete, run `/speckit.tasks` to generate:
   - `tasks.md` (Detailed implementation checklist)

3. Implement in order:
   - Phase 2.1: TLS Support
   - Phase 2.2: Security Middleware
   - Phase 2.3: Secure Cookies
   - Phase 2.4: Path Traversal Protection
   - Phase 2.5: Export Directory Isolation
   - Phase 2.6: Testing & Documentation

---

**Planning Status**: Phase 1 complete, ready for research phase
**Next Command**: Continue with research.md generation
