# Security Configuration Quick Start

**Feature**: 004-security-hardening
**Date**: 2025-11-01
**Audience**: Developers and system administrators deploying bskyarchive

## Overview

This guide provides quick start instructions for configuring security features in bskyarchive. Follow these steps to enable HTTPS, CSRF protection, security headers, and other security controls.

---

## Prerequisites

- bskyarchive v0.5.0+ (with security hardening features)
- TLS certificate and private key (for HTTPS)
- `SESSION_SECRET` environment variable set (32+ characters)

---

## Quick Start: Secure Production Deployment

### Step 1: Generate TLS Certificates

**Option A: Self-Signed Certificate (Development/Testing)**

```bash
# Generate self-signed certificate (valid for 365 days)
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
    -subj "/CN=localhost"

# Move certificates to project directory
mkdir -p ./certs
mv cert.pem key.pem ./certs/
```

**Option B: Let's Encrypt Certificate (Production)**

```bash
# Install certbot
sudo apt-get install certbot  # Debian/Ubuntu
# OR
brew install certbot          # macOS

# Obtain certificate (requires port 80 access and public domain)
sudo certbot certonly --standalone -d yourdomain.com

# Certificates located at:
# - /etc/letsencrypt/live/yourdomain.com/fullchain.pem (cert)
# - /etc/letsencrypt/live/yourdomain.com/privkey.pem (key)

# Link certificates to project directory
mkdir -p ./certs
ln -s /etc/letsencrypt/live/yourdomain.com/fullchain.pem ./certs/cert.pem
ln -s /etc/letsencrypt/live/yourdomain.com/privkey.pem ./certs/key.pem
```

**Option C: Certificate from CA (Production)**

```bash
# Copy certificates from your CA to project directory
mkdir -p ./certs
cp /path/to/your/certificate.crt ./certs/cert.pem
cp /path/to/your/private.key ./certs/key.pem

# Secure private key
chmod 600 ./certs/key.pem
```

### Step 2: Configure Security Settings

Edit `config.yaml`:

```yaml
server:
  port: 8443  # HTTPS standard port (or 443 for production)
  host: "0.0.0.0"  # Listen on all interfaces
  base_url: "https://yourdomain.com"  # Your public URL

  # TLS Configuration
  tls:
    enabled: true
    cert_file: "./certs/cert.pem"
    key_file: "./certs/key.pem"
    min_version: "1.3"  # TLS 1.3 (recommended) or "1.2" for legacy clients

  # Security Configuration
  security:
    csrf_enabled: true
    csrf_field_name: "csrf_token"
    max_request_bytes: 10485760  # 10MB limit

    # Security Headers
    headers:
      x_frame_options: "DENY"
      x_content_type_options: "nosniff"
      x_xss_protection: "1; mode=block"
      referrer_policy: "strict-origin-when-cross-origin"
      content_security_policy: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'"
      strict_transport_security: "max-age=31536000; includeSubDomains"

  # Timeouts
  read_timeout: 15s
  write_timeout: 15s
  idle_timeout: 60s
  shutdown_timeout: 30s

# Archive Configuration (existing)
archive:
  db_path: "./data/archive.db"
  media_path: "./data/media"
  max_archive_size_gb: 10
  worker_count: 3
  batch_size: 100

# OAuth Configuration
oauth:
  scopes:
    - "atproto"
    - "transition:generic"
    - "transition:chat.bsky"
  session_secret: "${SESSION_SECRET}"
  session_max_age: 604800  # 7 days

  # Cookie Security (auto-configured based on TLS)
  cookie_secure: "auto"  # auto/true/false
  cookie_samesite: "lax"  # lax (recommended for OAuth compatibility)

# Rate Limiting (existing)
rate_limit:
  requests_per_window: 300
  window_duration: 5m
  burst: 10
```

### Step 3: Set Environment Variables

```bash
# Generate secure session secret (32+ characters)
export SESSION_SECRET=$(openssl rand -base64 32)

# Optional: Set base URL
export BASE_URL="https://yourdomain.com"
```

**For Production**: Store `SESSION_SECRET` securely (environment file, secrets manager, etc.)

### Step 4: Start the Server

```bash
# Run bskyarchive with HTTPS enabled
./bskyarchive

# Expected output:
# [bskyarchive] 2025/11/01 12:00:00 Starting Bluesky Archive Tool...
# [bskyarchive] 2025/11/01 12:00:00 Configuration loaded successfully
# [bskyarchive] 2025/11/01 12:00:00 Database initialized successfully
# [bskyarchive] 2025/11/01 12:00:00 TLS enabled with minimum version TLS 1.3
# [bskyarchive] 2025/11/01 12:00:00 CSRF protection enabled
# [bskyarchive] 2025/11/01 12:00:00 Security headers configured
# [bskyarchive] 2025/11/01 12:00:00 Server starting on https://0.0.0.0:8443
```

### Step 5: Verify Security Configuration

**Check TLS Configuration**:
```bash
# Test TLS connection
curl -v https://localhost:8443 2>&1 | grep "SSL connection"

# Check TLS version
openssl s_client -connect localhost:8443 -tls1_3 2>&1 | grep "Protocol"
```

**Check Security Headers**:
```bash
# Verify security headers
curl -I https://localhost:8443

# Expected headers:
# X-Frame-Options: DENY
# X-Content-Type-Options: nosniff
# X-XSS-Protection: 1; mode=block
# Content-Security-Policy: default-src 'self'; ...
# Strict-Transport-Security: max-age=31536000; includeSubDomains
```

**Check CSRF Protection**:
```bash
# Attempt POST without CSRF token (should fail with 403)
curl -X POST https://localhost:8443/export/start -d "format=json"

# Expected: 403 Forbidden
```

---

## Development Mode Configuration

For local development without HTTPS:

```yaml
server:
  port: 8080
  host: "localhost"
  base_url: "http://localhost:8080"

  # TLS Disabled for Development
  tls:
    enabled: false  # HTTP only

  # Security Configuration (still active)
  security:
    csrf_enabled: true
    max_request_bytes: 10485760

    # Security Headers (without HSTS)
    headers:
      x_frame_options: "DENY"
      x_content_type_options: "nosniff"
      x_xss_protection: "1; mode=block"
      referrer_policy: "strict-origin-when-cross-origin"
      content_security_policy: "default-src 'self'"
      # HSTS omitted when TLS disabled

oauth:
  cookie_secure: "false"  # Allow cookies over HTTP (dev only!)
  cookie_samesite: "lax"
```

**Warning**: Development mode with `cookie_secure: false` should NEVER be used in production!

---

## Configuration Reference

### TLS Settings

| Setting | Values | Default | Description |
|---------|--------|---------|-------------|
| `tls.enabled` | true/false | false | Enable HTTPS |
| `tls.cert_file` | path | - | Path to TLS certificate |
| `tls.key_file` | path | - | Path to TLS private key |
| `tls.min_version` | "1.2"/"1.3" | "1.3" | Minimum TLS version |

### Security Settings

| Setting | Values | Default | Description |
|---------|--------|---------|-------------|
| `security.csrf_enabled` | true/false | true | Enable CSRF protection |
| `security.csrf_field_name` | string | "csrf_token" | Form field name for CSRF token |
| `security.max_request_bytes` | integer | 10485760 | Maximum request body size (bytes) |

### Security Headers

| Header | Default Value | Purpose |
|--------|---------------|---------|
| `x_frame_options` | DENY | Prevent clickjacking |
| `x_content_type_options` | nosniff | Prevent MIME sniffing |
| `x_xss_protection` | 1; mode=block | Enable XSS filter |
| `referrer_policy` | strict-origin-when-cross-origin | Control referrer information |
| `content_security_policy` | default-src 'self' | Prevent XSS/injection |
| `strict_transport_security` | max-age=31536000 | Force HTTPS (when TLS enabled) |

### Cookie Settings

| Setting | Values | Default | Description |
|---------|--------|---------|-------------|
| `oauth.cookie_secure` | auto/true/false | auto | Set Secure flag (auto=based on TLS) |
| `oauth.cookie_samesite` | strict/lax/none | lax | SameSite cookie attribute |

---

## Troubleshooting

### Issue: "TLS certificate not found"

**Solution**:
1. Verify certificate files exist at configured paths
2. Check file permissions (readable by application)
3. Ensure correct paths in config.yaml

```bash
# Check certificate files
ls -la ./certs/cert.pem ./certs/key.pem

# Verify certificate validity
openssl x509 -in ./certs/cert.pem -text -noout
```

### Issue: "CSRF token invalid"

**Solution**:
1. Verify `SESSION_SECRET` is set and consistent across restarts
2. Clear browser cookies and retry
3. Check that CSRF token is present in form/header

```bash
# Verify SESSION_SECRET is set
echo $SESSION_SECRET

# Check CSRF token in HTML
curl https://localhost:8443 | grep csrf-token
```

### Issue: "OAuth callback fails with Secure cookie"

**Solution**:
1. Ensure `base_url` uses HTTPS in config
2. Verify TLS is enabled and working
3. Check OAuth provider configuration matches base URL

### Issue: "CSP blocks inline scripts"

**Solution**:
1. Review Content-Security-Policy header
2. Use external scripts instead of inline
3. Update CSP directives in config if needed

```yaml
# Allow specific inline scripts (not recommended)
content_security_policy: "default-src 'self'; script-src 'self' 'unsafe-inline'"
```

### Issue: "Request too large (413)"

**Solution**:
1. Check `max_request_bytes` configuration
2. Increase limit if legitimate large requests needed
3. Verify request size is reasonable

```yaml
# Increase limit to 50MB
max_request_bytes: 52428800  # 50MB
```

---

## Security Checklist

Before deploying to production, verify:

- [ ] TLS 1.3 enabled with valid certificates
- [ ] `SESSION_SECRET` is strong (32+ random characters)
- [ ] `cookie_secure: auto` or `true` (not `false`)
- [ ] All security headers configured
- [ ] CSRF protection enabled
- [ ] Request size limits appropriate
- [ ] Base URL uses HTTPS
- [ ] Firewall rules configured (allow 443, block 80 or redirect)
- [ ] Certificate auto-renewal configured (Let's Encrypt)
- [ ] Security headers verified with browser developer tools
- [ ] CSRF protection tested with POST requests
- [ ] TLS configuration tested with SSL Labs (https://www.ssllabs.com/ssltest/)

---

## Additional Resources

- **Mozilla Security Headers**: https://infosec.mozilla.org/guidelines/web_security
- **OWASP CSRF Guide**: https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
- **Let's Encrypt**: https://letsencrypt.org/getting-started/
- **SSL Labs Server Test**: https://www.ssllabs.com/ssltest/
- **CSP Evaluator**: https://csp-evaluator.withgoogle.com/

---

## Support

For security issues or questions:
1. Review this guide and configuration reference
2. Check troubleshooting section
3. Consult feature specification: `specs/004-security-hardening/spec.md`
4. Review security audit report for rationale

**Security Configuration**: ✅ READY - Follow this guide for secure deployment
