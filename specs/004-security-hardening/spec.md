# Feature Specification: Security Hardening

**Feature ID**: 004-security-hardening
**Created**: 2025-11-01
**Status**: Planning
**Priority**: HIGH

## Problem Statement

A comprehensive security audit (2025-11-01) identified critical security vulnerabilities that must be addressed before production deployment. The application currently lacks essential security controls including HTTPS/TLS support, CSRF protection, secure cookie configuration, and HTTP security headers. These deficiencies expose users to session hijacking, cross-site request forgery, clickjacking, and man-in-the-middle attacks.

## User Impact

**Who**: All users of the bskyarchive application
**Current Pain**: Users' OAuth tokens and session data are transmitted unencrypted, making them vulnerable to network interception. State-changing operations can be exploited via CSRF attacks. Browser-based attacks (XSS, clickjacking) are not mitigated.

**Value Delivered**:
- Encrypted communication via HTTPS/TLS protects OAuth tokens and session cookies
- CSRF protection prevents unauthorized state changes
- Secure cookie configuration prevents session hijacking
- HTTP security headers provide defense-in-depth against browser attacks
- Request size limits prevent denial-of-service attacks
- Path traversal protections prevent unauthorized file access

## Requirements

### Functional Requirements

**FR1: HTTPS/TLS Support (CRITICAL)**
- Server must support TLS 1.3 with secure cipher suites
- Automatic certificate management via Let's Encrypt or manual certificate loading
- HTTP-to-HTTPS redirect capability
- Environment-based TLS configuration (dev vs production)

**FR2: CSRF Protection (CRITICAL)**
- Implement CSRF tokens for all state-changing operations (POST, PUT, DELETE)
- Use gorilla/csrf library (already in dependencies)
- Token validation on:
  - `/archive/start` (POST)
  - `/export/start` (POST)
  - Any future state-changing endpoints
- HTMX compatibility with CSRF tokens

**FR3: Secure Cookie Configuration (CRITICAL)**
- Set `Secure: true` flag on cookies when HTTPS is enabled
- Upgrade `SameSite` from `Lax` to `Strict` mode
- Add environment detection for automatic secure flag setting

**FR4: HTTP Security Headers (HIGH)**
- Add security headers middleware to set:
  - `X-Frame-Options: DENY` (clickjacking protection)
  - `X-Content-Type-Options: nosniff` (MIME sniffing protection)
  - `X-XSS-Protection: 1; mode=block` (legacy XSS protection)
  - `Content-Security-Policy` (XSS/injection protection)
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Strict-Transport-Security` (HSTS, when HTTPS enabled)

**FR5: Request Size Limits (HIGH)**
- Implement `MaxBytesReader` middleware
- Default limit: 10MB per request
- Configurable via config.yaml
- Proper error handling for oversized requests

**FR6: Path Traversal Protection (HIGH)**
- Fix static file serving in `/static/*` handler
- Validate resolved paths are within allowed directory
- Add absolute path resolution with prefix checking
- Apply same protection to media serving endpoint

**FR7: Export Directory Isolation (MEDIUM)**
- Create per-user export subdirectories: `./exports/{did}/`
- Add ownership verification in export progress endpoint
- Prevent users from accessing other users' exports

### Non-Functional Requirements

**NFR1: Performance**
- Security headers middleware adds <1ms per request
- CSRF token validation adds <2ms per request
- TLS handshake overhead acceptable (standard TLS 1.3 performance)

**NFR2: Backward Compatibility**
- HTTP mode still available for development/testing
- Graceful degradation if TLS certificates not configured
- Existing database schema unchanged

**NFR3: Configuration**
- All security settings configurable via config.yaml
- Environment variable overrides supported
- Sensible defaults for production security

**NFR4: Observability**
- Log security-related events (failed CSRF validation, path traversal attempts)
- Startup warnings for insecure configurations
- Clear error messages for configuration issues

## Success Criteria

1. **HTTPS/TLS**: Server accepts TLS 1.3 connections with secure ciphers
2. **CSRF**: All POST endpoints require valid CSRF tokens; invalid tokens return 403
3. **Cookies**: Session cookies set with Secure=true and SameSite=Strict in production
4. **Headers**: All security headers present on every response
5. **Size Limits**: Requests >10MB rejected with 413 status
6. **Path Security**: Requests to `/../` or other traversal attempts blocked
7. **Isolation**: Users can only access their own exports
8. **Tests**: Security controls verified with integration tests

## Out of Scope

- Rate limiting on authentication endpoints (defer to future feature)
- Advanced CSP directives (start with basic policy)
- Automated security scanning/SAST integration
- OAuth token encryption at rest (requires separate key management feature)
- Search query length limits (low priority)
- Session secret entropy validation (low priority)
- Security audit logging infrastructure

## Dependencies

**Internal**:
- Existing authentication system (`internal/auth/`)
- Session management (`internal/auth/session.go`)
- HTTP handlers (`internal/web/handlers/`)
- Configuration system (`internal/config/`)

**External**:
- `github.com/gorilla/csrf` v1.7.3 (already imported)
- Go standard library `crypto/tls`
- Go standard library `net/http`

## Technical Approach

### Architecture Changes

1. **Middleware Stack** (in `cmd/bskyarchive/main.go`):
   ```
   Router
   ├── RequestID
   ├── RealIP
   ├── SecurityHeaders (NEW)
   ├── MaxBytesMiddleware (NEW)
   ├── CSRF (NEW)
   ├── Logging
   ├── Recoverer
   └── Timeout
   ```

2. **TLS Configuration**:
   - Add `TLSConfig` struct to `ServerConfig` in config
   - Support both automatic (Let's Encrypt) and manual certificate modes
   - Environment-based selection (dev without TLS, production with TLS)

3. **Cookie Security**:
   - Modify `InitSessions()` in `internal/auth/session.go`
   - Add environment detection helper
   - Set Secure flag based on TLS configuration

### File Modifications

**High Priority**:
- `cmd/bskyarchive/main.go` - Add middleware, TLS config
- `internal/web/middleware/security.go` - NEW: Security headers middleware
- `internal/web/middleware/csrf.go` - NEW: CSRF middleware wrapper
- `internal/config/config.go` - Add TLS configuration fields
- `internal/auth/session.go` - Secure cookie configuration
- `internal/web/handlers/handlers.go` - Fix static file serving (line 406)
- `config.yaml` - Add security configuration section

**Medium Priority**:
- `internal/web/handlers/export.go` - Per-user export directories
- `internal/exporter/exporter.go` - Update export directory creation

### Testing Strategy

**Unit Tests**:
- CSRF token generation and validation
- Security header presence
- Cookie flag settings based on environment

**Integration Tests**:
- End-to-end TLS connections
- CSRF protection on POST endpoints
- Path traversal attempt blocking
- Export directory isolation

**Security Tests**:
- Attempt CSRF attack without token
- Path traversal with `/../` sequences
- Access other user's exports
- Oversized request handling

## Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| TLS certificate management complexity | HIGH | Support both manual and automatic (Let's Encrypt) modes; provide clear documentation |
| Breaking existing development workflows | MEDIUM | Keep HTTP mode available for local development; environment-based configuration |
| CSRF breaks HTMX requests | MEDIUM | Ensure CSRF tokens properly integrated with HTMX; test thoroughly |
| Performance impact of middleware | LOW | Benchmark middleware overhead; optimize if >5ms per request |
| Cookie SameSite=Strict breaks OAuth callback | MEDIUM | Test OAuth flow with Strict mode; revert to Lax if incompatible |

## Open Questions

1. **Certificate Management**: Use Let's Encrypt automatic mode or require manual certificates?
   - **Recommendation**: Support both; default to manual for simplicity

2. **HTTP-to-HTTPS Redirect**: Automatic redirect or separate HTTP port?
   - **Recommendation**: Optional redirect; document in configuration

3. **CSP Policy**: Start strict or permissive?
   - **Recommendation**: Start with `default-src 'self'`; refine based on testing

4. **HSTS max-age**: How long to cache HSTS policy?
   - **Recommendation**: 1 year (31536000 seconds) for production

## Implementation Phases

### Phase 1: Critical Security Controls (HIGH)
- HTTPS/TLS support
- Secure cookie configuration
- CSRF protection
- HTTP security headers

### Phase 2: Access Controls (HIGH)
- Path traversal protection
- Request size limits
- Export directory isolation

### Phase 3: Testing & Documentation (MEDIUM)
- Security integration tests
- Configuration documentation
- Deployment guide updates

### Phase 4: Hardening (MEDIUM)
- Security event logging
- Configuration validation
- Startup security checks

## References

- Security Audit Report (2025-11-01)
- OWASP Top 10 (2021): https://owasp.org/Top10/
- Mozilla Security Headers: https://infosec.mozilla.org/guidelines/web_security
- Go TLS Configuration: https://go.dev/src/crypto/tls/
- gorilla/csrf Documentation: https://github.com/gorilla/csrf

## Approvals

- [ ] Security requirements reviewed
- [ ] Constitution principles verified
- [ ] Technical approach validated
- [ ] Implementation phases agreed
