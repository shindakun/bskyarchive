# Security Troubleshooting Guide

This guide helps diagnose and resolve common security-related issues with the bskyarchive application.

## Table of Contents

1. [CSRF Protection Issues](#csrf-protection-issues)
2. [Cookie Security Problems](#cookie-security-problems)
3. [OAuth Login Failures](#oauth-login-failures)
4. [Security Headers Not Applied](#security-headers-not-applied)
5. [Request Size Limit Errors](#request-size-limit-errors)
6. [Path Traversal Blocks](#path-traversal-blocks)
7. [Export Access Denied](#export-access-denied)
8. [HTTPS/TLS Issues](#httpstls-issues)

---

## CSRF Protection Issues

### Symptom: "403 Forbidden" on form submissions

**Error Message:**
```
Forbidden - CSRF token invalid
```

**Common Causes:**
1. CSRF token missing from form
2. CSRF token expired (session expired)
3. Session cookie not being sent
4. CSRF protection misconfigured

**Solutions:**

**Check 1: Verify CSRF token in forms**
```bash
# View page source and look for CSRF token
curl -c cookies.txt https://your-subdomain.ngrok.app/some-page
grep csrf_token cookies.txt
```

Look for hidden input field:
```html
<input type="hidden" name="csrf_token" value="...">
```

**Check 2: Verify CSRF configuration**
```yaml
# In config.yaml
server:
  security:
    csrf_enabled: true
    csrf_field_name: "csrf_token"
```

**Check 3: Check browser console**
- Open browser DevTools (F12)
- Look for CSRF-related errors
- Check if CSRF meta tag exists: `<meta name="csrf-token" content="...">`

**Check 4: Verify HTMX CSRF headers**
```html
<!-- In base.html template -->
<body hx-headers='{"X-CSRF-Token": "{{ .CSRFToken }}"}'>
```

**Workaround:**
If OAuth login is blocked, verify that `/auth/login` is exempt:
```go
// In internal/web/middleware/csrf.go
if r.Method == http.MethodPost && r.URL.Path == "/auth/login" {
    next.ServeHTTP(w, r)
    return
}
```

---

## Cookie Security Problems

### Symptom: Session not persisting / logged out immediately

**Error Message:**
```
Session cookie rejected by browser
```

**Common Causes:**
1. Secure flag set but accessing over HTTP
2. BASE_URL misconfigured
3. Browser blocking cookies
4. SameSite policy too strict

**Solutions:**

**Check 1: Verify BASE_URL configuration**
```bash
# Check environment variable
echo $BASE_URL

# Should be https:// for production
# https://your-subdomain.ngrok.app (correct)
# http://localhost:8080 (only for local dev)
```

**Check 2: Check startup logs**
```bash
./bskyarchive 2>&1 | grep -i cookie

# Should see:
# "Cookie security enabled: true"  (for HTTPS)
# "Cookie security enabled: false" (for HTTP)
```

**Check 3: Verify cookie configuration**
```yaml
# In config.yaml
oauth:
  cookie_secure: "auto"      # Recommended
  cookie_samesite: "lax"     # Required for OAuth
```

**Check 4: Inspect cookie in browser**
- Open DevTools → Application → Cookies
- Look for session cookie
- Verify attributes:
  - `Secure`: Should be `true` for HTTPS
  - `HttpOnly`: Should be `true`
  - `SameSite`: Should be `Lax`

**Workaround for development:**
```yaml
oauth:
  cookie_secure: "false"  # Only for local HTTP testing
```

---

## OAuth Login Failures

### Symptom: OAuth redirect fails or infinite loop

**Error Message:**
```
OAuth callback failed: state mismatch
Invalid redirect URI
```

**Common Causes:**
1. BASE_URL doesn't match OAuth callback URL
2. CSRF blocking OAuth endpoints
3. Cookie not persisting OAuth state
4. ngrok URL changed

**Solutions:**

**Check 1: Verify BASE_URL matches ngrok**
```bash
# Check ngrok status
curl http://127.0.0.1:4040/api/tunnels

# Check BASE_URL
echo $BASE_URL

# They must match exactly
```

**Check 2: Verify OAuth callback URL**
```bash
# Should be: https://your-subdomain.ngrok.app/auth/callback
# Check application logs for OAuth initialization
grep "OAuth manager initialized" logs/app.log
```

**Check 3: Test OAuth state cookie**
```bash
# Start OAuth flow
curl -c cookies.txt -L https://your-subdomain.ngrok.app/auth/login

# Check for oauth_state cookie
cat cookies.txt | grep oauth_state
```

**Check 4: Verify CSRF exemption**
OAuth login should be exempt from CSRF validation. Check:
```go
// In internal/web/middleware/csrf.go
if r.Method == http.MethodPost && r.URL.Path == "/auth/login" {
    next.ServeHTTP(w, r)  // Bypass CSRF for OAuth
    return
}
```

**Workaround:**
Use a stable ngrok subdomain (requires ngrok account):
```bash
ngrok http 8080 --subdomain=your-subdomain
```

---

## Security Headers Not Applied

### Symptom: Security headers missing in responses

**Error Message:**
```
Security scan shows missing headers
```

**Common Causes:**
1. Middleware not registered in router
2. Headers misconfigured in config.yaml
3. Reverse proxy stripping headers

**Solutions:**

**Check 1: Verify headers in response**
```bash
curl -I https://your-subdomain.ngrok.app | grep -E "X-Frame|X-Content|X-XSS"

# Should see:
# X-Frame-Options: DENY
# X-Content-Type-Options: nosniff
# X-XSS-Protection: 1; mode=block
```

**Check 2: Verify middleware registration**
```go
// In cmd/bskyarchive/main.go
router.Use(middleware.SecurityHeaders(cfg))
```

**Check 3: Check configuration**
```yaml
# In config.yaml
server:
  security:
    headers:
      x_frame_options: "DENY"
      x_content_type_options: "nosniff"
      # ... other headers
```

**Check 4: Verify HSTS is conditional**
HSTS should only appear over HTTPS:
```bash
# Over HTTPS (ngrok) - should have HSTS
curl -I https://your-subdomain.ngrok.app | grep Strict-Transport

# Over HTTP (local) - should NOT have HSTS
curl -I http://localhost:8080 | grep Strict-Transport
```

**Debugging:**
Add logging to SecurityHeaders middleware:
```go
log.Printf("Security headers applied for: %s", r.URL.Path)
```

---

## Request Size Limit Errors

### Symptom: Large uploads fail with 413 error

**Error Message:**
```
413 Payload Too Large
http: request body too large
```

**Common Causes:**
1. Request exceeds 10MB default limit
2. Streaming upload not chunked
3. Limit set too low for use case

**Solutions:**

**Check 1: Verify request size**
```bash
# Check size of file being uploaded
ls -lh file.json

# Test with curl
curl -X POST --data-binary @file.json https://your-subdomain.ngrok.app/endpoint
```

**Check 2: Adjust limit if needed**
```yaml
# In config.yaml
server:
  security:
    max_request_bytes: 10485760  # 10MB default
    # max_request_bytes: 52428800  # 50MB if needed
```

**Check 3: Verify middleware is working**
```bash
# Send exactly 10MB (should succeed)
dd if=/dev/zero bs=1M count=10 | curl -X POST --data-binary @- https://your-subdomain.ngrok.app/test

# Send 11MB (should fail with 413)
dd if=/dev/zero bs=1M count=11 | curl -X POST --data-binary @- https://your-subdomain.ngrok.app/test
```

**Check 4: Check for double limiting**
ngrok may have its own limits. Check ngrok logs:
```bash
curl http://127.0.0.1:4040/api/requests/http
```

**Workaround:**
For legitimate large uploads, increase limit:
```yaml
max_request_bytes: 104857600  # 100MB
```

---

## Path Traversal Blocks

### Symptom: Legitimate file requests blocked

**Error Message:**
```
404 Not Found
Security: Path traversal attempt blocked
```

**Common Causes:**
1. URL contains `..` legitimately (e.g., in filename)
2. Path validation too strict
3. Symlinks in file path

**Solutions:**

**Check 1: Verify path structure**
```bash
# Check what path is being requested
grep "Path traversal" logs/app.log

# Example log:
# Security: Path traversal attempt blocked - requested: ../../../etc/passwd
```

**Check 2: Test specific path**
```bash
# Test static file access
curl -I https://your-subdomain.ngrok.app/static/css/style.css

# Should work (200 OK)

# Test traversal (should fail)
curl -I https://your-subdomain.ngrok.app/static/../config.yaml

# Should return 404
```

**Check 3: Verify file exists**
```bash
# Check if file actually exists
ls -la internal/web/static/css/style.css
```

**Check 4: Review path validation logic**
```go
// In internal/web/handlers/handlers.go
cleanPath := filepath.Clean(path)
absFullPath, _ := filepath.Abs(fullPath)

// Verify path is within allowed directory
if !strings.HasPrefix(absFullPath, absStaticDir+string(filepath.Separator)) {
    // Blocked
}
```

**False Positive Fix:**
If legitimate files are blocked, check for:
- Symlinks in path
- Mixed path separators (/ vs \)
- URL encoding issues

---

## Export Access Denied

### Symptom: Cannot access own exports

**Error Message:**
```
403 Forbidden
Security: Unauthorized export access attempt
```

**Common Causes:**
1. DID mismatch (session vs export owner)
2. Export job not found
3. Session expired

**Solutions:**

**Check 1: Verify session**
```bash
# Check browser console for session info
# DevTools → Application → Cookies
# Look for session cookie
```

**Check 2: Check export ownership**
```bash
# Exports are stored in per-user directories
ls -la exports/

# Should see:
# exports/did:plc:abc123/2025-11-01_12-00-00/
```

**Check 3: Verify DID in logs**
```bash
grep "Unauthorized export access" logs/app.log

# Example:
# user did:plc:xxx attempted to access job owned by did:plc:yyy
```

**Check 4: Test export creation**
```bash
# Create new export and note the job ID
# Access should work immediately
curl https://your-subdomain.ngrok.app/export/progress/{job_id}
```

**Session Refresh:**
If session expired:
1. Log out
2. Log back in
3. Try export again

---

## HTTPS/TLS Issues

### Symptom: Certificate errors or HTTPS not working

**Error Message:**
```
ERR_CERT_AUTHORITY_INVALID
SSL certificate problem
```

**Common Causes:**
1. ngrok tunnel not running
2. Using HTTP URL instead of HTTPS
3. Browser cache issues
4. ngrok certificate not trusted

**Solutions:**

**Check 1: Verify ngrok tunnel**
```bash
# Check ngrok status
curl http://127.0.0.1:4040/api/tunnels | jq .

# Should show active HTTPS tunnel
```

**Check 2: Verify HTTPS URL**
```bash
# Test HTTPS endpoint
curl -v https://your-subdomain.ngrok.app 2>&1 | grep -E "SSL|TLS"

# Should see TLS 1.3
```

**Check 3: Test TLS connection**
```bash
openssl s_client -connect your-subdomain.ngrok.app:443 -tls1_3

# Should successfully connect
# Certificate should be issued by "ngrok"
```

**Check 4: Verify HSTS header**
```bash
curl -I https://your-subdomain.ngrok.app | grep Strict-Transport

# Should see:
# Strict-Transport-Security: max-age=31536000; includeSubDomains
```

**Browser Cache Issue:**
Clear browser data:
1. Open DevTools (F12)
2. Right-click reload button
3. Select "Empty Cache and Hard Reload"

**Workaround for development:**
Accept ngrok certificate:
1. Visit HTTPS URL in browser
2. Click "Advanced"
3. Click "Proceed to site"

---

## General Debugging Tips

### Enable Debug Logging

Add detailed logging to middleware:
```go
// In internal/web/middleware/*.go
log.Printf("DEBUG: %s %s - %v", r.Method, r.URL.Path, someValue)
```

### Check Application Logs

```bash
# Tail logs in real-time
tail -f logs/application.log

# Filter for security events
grep -i "security" logs/application.log
```

### Test with curl

```bash
# Full verbose output
curl -v -c cookies.txt -b cookies.txt https://your-subdomain.ngrok.app

# Follow redirects
curl -L https://your-subdomain.ngrok.app

# Include headers
curl -H "X-Custom-Header: value" https://your-subdomain.ngrok.app
```

### Browser DevTools

1. Open DevTools (F12)
2. **Network tab**: See all requests/responses
3. **Console tab**: JavaScript errors
4. **Application tab**: Cookies, storage
5. **Security tab**: Certificate info

### Test Security Headers

Use online tools:
- https://securityheaders.com/
- https://observatory.mozilla.org/

---

## Getting Help

If you're still stuck:

1. **Check the logs** - Most issues show up in logs
2. **Review the checklist** - `SECURITY-CHECKLIST.md`
3. **Read the spec** - `specs/004-security-hardening/spec.md`
4. **Check configuration** - `config.yaml` with detailed comments
5. **Run tests** - `go test ./...` to verify functionality

### Report an Issue

If you found a bug:
1. Gather logs and error messages
2. Note your configuration (redact secrets!)
3. Describe steps to reproduce
4. Open an issue on GitHub

---

**Last Updated**: 2025-11-01
**Feature**: 004-security-hardening
