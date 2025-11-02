# Security Checklist for Production Deployment

This checklist ensures all security controls are properly configured before deploying to production with ngrok HTTPS.

## Pre-Deployment Checklist

### Environment Setup

- [ ] **Generate SESSION_SECRET**
  ```bash
  export SESSION_SECRET=$(openssl rand -base64 32)
  ```
  - Must be at least 32 characters
  - Keep secret - never commit to version control
  - Store securely (e.g., password manager, secrets manager)

- [ ] **Configure BASE_URL**
  ```bash
  export BASE_URL="https://your-subdomain.ngrok.app"
  ```
  - Must use `https://` protocol for secure cookies
  - Should match your ngrok domain
  - Required for OAuth to function correctly

- [ ] **Verify ngrok installation**
  ```bash
  ngrok version
  ```
  - ngrok v3+ recommended
  - Account authenticated (for stable subdomains)

### Configuration Review

- [ ] **Review config.yaml security settings**
  - `server.security.csrf_enabled: true` (CSRF protection enabled)
  - `server.security.max_request_bytes: 10485760` (10MB limit)
  - `oauth.cookie_secure: "auto"` (auto-detect HTTPS)
  - `oauth.cookie_samesite: "lax"` (OAuth compatibility)
  - All security headers configured

- [ ] **Verify session configuration**
  - `oauth.session_secret: "${SESSION_SECRET}"` (environment variable)
  - `oauth.session_max_age: 604800` (7 days, or your preference)

- [ ] **Check file paths**
  - `archive.db_path` points to writable directory
  - `archive.media_path` has sufficient disk space
  - Export directory is writable

### Build & Test

- [ ] **Run all tests**
  ```bash
  go test ./...
  ```
  - All tests should pass
  - No skipped security tests

- [ ] **Build application**
  ```bash
  go build -o bskyarchive ./cmd/bskyarchive
  ```
  - No build errors
  - Binary created successfully

- [ ] **Test local startup**
  ```bash
  export SESSION_SECRET=$(openssl rand -base64 32)
  export BASE_URL="http://localhost:8080"
  ./bskyarchive
  ```
  - Application starts without errors
  - Check logs for "Cookie security enabled: false" (HTTP)

## Deployment Checklist

### ngrok Setup

- [ ] **Start ngrok tunnel**
  ```bash
  ngrok http 8080
  ```
  - Note the HTTPS URL (e.g., `https://abc123.ngrok.app`)
  - Verify tunnel is active and accessible

- [ ] **Update BASE_URL**
  ```bash
  export BASE_URL="https://abc123.ngrok.app"
  ```
  - Use the exact ngrok URL
  - Must include `https://` protocol

### Application Startup

- [ ] **Start application with production settings**
  ```bash
  export SESSION_SECRET="<your-secret>"
  export BASE_URL="https://your-subdomain.ngrok.app"
  ./bskyarchive
  ```

- [ ] **Verify startup logs**
  - ✓ "OAuth manager initialized with base URL: https://..."
  - ✓ "Cookie security enabled: true"
  - ✓ "CSRF protection enabled"
  - ✓ "Starting server on localhost:8080"

### Security Verification

- [ ] **Test HTTPS access**
  ```bash
  curl -I https://your-subdomain.ngrok.app
  ```
  - Returns HTTP 200 or redirects correctly
  - No TLS/SSL errors

- [ ] **Verify security headers**
  ```bash
  curl -I https://your-subdomain.ngrok.app | grep -E "X-Frame|X-Content|X-XSS|Referrer|Content-Security|Strict-Transport"
  ```
  - ✓ `X-Frame-Options: DENY`
  - ✓ `X-Content-Type-Options: nosniff`
  - ✓ `X-XSS-Protection: 1; mode=block`
  - ✓ `Referrer-Policy: strict-origin-when-cross-origin`
  - ✓ `Content-Security-Policy: default-src 'self'...`
  - ✓ `Strict-Transport-Security: max-age=31536000; includeSubDomains`

- [ ] **Test TLS configuration (via ngrok)**
  ```bash
  openssl s_client -connect your-subdomain.ngrok.app:443 -tls1_3
  ```
  - TLS 1.3 supported
  - Valid certificate chain
  - Strong cipher suites

- [ ] **Test OAuth flow**
  1. Visit `https://your-subdomain.ngrok.app`
  2. Click "Sign in with Bluesky"
  3. Enter Bluesky handle
  4. Complete OAuth authorization
  5. Verify successful redirect and login
  - ✓ No certificate errors
  - ✓ OAuth callback works
  - ✓ Session cookie created with Secure flag

- [ ] **Test CSRF protection**
  ```bash
  # Attempt POST without CSRF token (should fail)
  curl -X POST https://your-subdomain.ngrok.app/some-endpoint
  ```
  - Should return HTTP 403 Forbidden
  - OAuth login endpoint should still work

- [ ] **Test request size limits**
  ```bash
  # Attempt to send >10MB request (should fail)
  dd if=/dev/zero bs=1M count=11 | curl -X POST --data-binary @- https://your-subdomain.ngrok.app/some-endpoint
  ```
  - Should return HTTP 413 Payload Too Large

- [ ] **Test path traversal protection**
  ```bash
  curl -I https://your-subdomain.ngrok.app/static/../../../etc/passwd
  ```
  - Should return HTTP 404 Not Found
  - Check logs for security warning

## Post-Deployment Monitoring

### Initial Checks

- [ ] **Monitor application logs**
  - No unexpected errors
  - Security events logged appropriately
  - No suspicious path traversal attempts

- [ ] **Test user workflows**
  - Login/logout works
  - Archive viewing works
  - Export functionality works
  - Media serving works

- [ ] **Verify export isolation**
  - Create export as user A
  - Try to access user A's export as user B
  - Should receive HTTP 403 Forbidden

### Ongoing Monitoring

- [ ] **Regular log review**
  - Weekly review of security logs
  - Monitor for CSRF failures
  - Monitor for path traversal attempts
  - Monitor for unauthorized export access

- [ ] **Session secret rotation**
  - Plan to rotate SESSION_SECRET periodically
  - Will invalidate all existing sessions

- [ ] **Security updates**
  - Keep Go runtime updated
  - Update dependencies regularly
  - Monitor security advisories

## Incident Response

### Security Event Response

If you observe suspicious activity:

1. **Review logs immediately**
   ```bash
   grep -i "security" logs/application.log
   ```

2. **Common issues to investigate**
   - Multiple CSRF failures from same IP
   - Path traversal attempts
   - Unauthorized export access attempts
   - Unusual request patterns

3. **Mitigation steps**
   - Block suspicious IPs at ngrok level
   - Rotate SESSION_SECRET if sessions compromised
   - Review and update security configurations
   - Consider additional rate limiting

### Emergency Shutdown

If critical security issue detected:

1. **Stop application immediately**
   ```bash
   pkill bskyarchive
   ```

2. **Stop ngrok tunnel**
   ```bash
   pkill ngrok
   ```

3. **Review and fix issue**
4. **Re-run security checklist before redeployment**

## Additional Resources

- **Security Hardening Spec**: `specs/004-security-hardening/spec.md`
- **Configuration Guide**: `README.md` - Security section
- **Troubleshooting**: `specs/004-security-hardening/TROUBLESHOOTING.md`
- **ngrok Documentation**: https://ngrok.com/docs
- **OWASP Top 10**: https://owasp.org/www-project-top-ten/

## Compliance Notes

This checklist addresses common web application vulnerabilities:

- ✓ **A01:2021 - Broken Access Control**: Export isolation, path traversal protection
- ✓ **A02:2021 - Cryptographic Failures**: Secure cookies, HTTPS, session secrets
- ✓ **A03:2021 - Injection**: CSP, input validation
- ✓ **A04:2021 - Insecure Design**: CSRF protection, security headers
- ✓ **A05:2021 - Security Misconfiguration**: Default secure settings, comprehensive config
- ✓ **A07:2021 - Identification/Authentication Failures**: OAuth with PKCE, secure sessions

---

**Last Updated**: 2025-11-01
**Feature**: 004-security-hardening
**Status**: Production Ready
