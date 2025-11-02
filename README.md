# Bluesky Personal Archive Tool

A local-first archival solution for Bluesky. Archive and search your Bluesky content on your own machine.

## Features

- **Privacy-first**: All data stored locally on your machine
- **Full-text search**: Find any post instantly with SQLite FTS5
- **Complete archive**: Posts, media, profiles, and engagement metrics
- **Fast & efficient**: Incremental updates and rate-limited operations
- **Export your data**: Export to JSON or CSV formats with optional media files and date filtering

## Export Your Archive

Export your archived posts to portable formats for backup, analysis, or migration.

### Supported Formats

**JSON Export** (Complete metadata)
- Full post data with all metadata, engagement metrics, and relationships
- Preserves nested structures (embeds, replies, quotes)
- Includes `manifest.json` with export metadata
- Best for: Complete backups, data portability, programmatic access

**CSV Export** (Spreadsheet-compatible)
- RFC 4180 compliant format with UTF-8 BOM for Excel compatibility
- 15 columns: URI, CID, DID, Text, CreatedAt, engagement metrics, reply data, media info
- Media hashes as semicolon-separated list
- Best for: Spreadsheet analysis, Excel/Google Sheets, data visualization

### Export Options

**Media Files** (optional)
- Copies media files to `/media` subdirectory in export
- Organized by content hash for deduplication
- Includes images from posts, link previews, and quoted posts

**Date Range Filtering** (optional)
- Filter posts by creation date
- Supports start date, end date, or both
- Useful for exporting specific time periods

### Using the Export Feature

1. Navigate to the **Export** page in the web interface
2. Choose your format (JSON or CSV)
3. Select whether to include media files
4. Optionally set a date range filter
5. Click "Start Export"
6. Monitor progress in real-time
7. Find your export in `./exports/YYYY-MM-DD_HH-MM-SS/`

### Export Directory Structure

```
exports/
└── 2025-01-31_14-30-00/
    ├── manifest.json       # Export metadata
    ├── posts.json         # (JSON format) or posts.csv (CSV format)
    └── media/             # (if media included)
        ├── bafkreiabc123.jpeg
        └── bafkreixyz789.png
```

### Troubleshooting

**"An export is already in progress"**
- Only one export can run at a time per user
- Wait for the current export to complete before starting a new one

**"No posts found in your archive"**
- Archive some posts first by syncing your Bluesky account
- Check your date range filter - you may need to adjust or remove it

**Missing media files**
- Media files that weren't downloaded during archival will be skipped
- Check the logs for warnings about missing files
- The export will still complete successfully with available media

## Requirements

- Go 1.21 or higher
- SQLite (included via modernc.org/sqlite)

## Quick Start

### 1. Clone the repository

```bash
git clone https://github.com/shindakun/bskyarchive.git
cd bskyarchive
```

### 2. Set up environment variables

**Required:**
```bash
# Generate a secure random session secret (must be at least 32 characters)
export SESSION_SECRET=$(openssl rand -base64 32)
```

**Optional (for OAuth to work):**
```bash
# Set your public base URL (required for OAuth with Bluesky)
export BASE_URL="https://your-domain.com"
```

**Important**:
- The `SESSION_SECRET` is used to encrypt session cookies. Keep it secret and don't commit it to version control.
- The `BASE_URL` overrides the config file and is required for OAuth to work with Bluesky (localhost won't work)

### 3. Build and run

```bash
go build -o bskyarchive ./cmd/bskyarchive
./bskyarchive
```

The server will start on `http://localhost:8080`

### 4. First-time setup

1. Visit `http://localhost:8080`
2. Click "Sign in with Bluesky"
3. Enter your Bluesky handle (e.g., `user.bsky.social`)
4. Complete the OAuth flow
5. You'll be redirected to your dashboard

## Configuration

The application uses `config.yaml` for configuration. Default settings:

```yaml
server:
  port: 8080
  host: "localhost"
  # base_url: "https://your-domain.com"  # Required for OAuth to work with Bluesky

archive:
  db_path: "./data/archive.db"
  media_path: "./data/media"

oauth:
  scopes:
    - "atproto"
    - "transition:generic"
    - "transition:chat.bsky"
  session_max_age: 604800  # 7 days
```

### Setting the Base URL

**Important**: For OAuth to work with Bluesky, you need to set a publicly accessible `base_url`.

**Option 1: Environment Variable (Recommended)**
```bash
export BASE_URL="https://your-domain.com"
./bskyarchive
```

**Option 2: Config File**
```yaml
server:
  base_url: "https://your-domain.com"
```

**For local development with a tunnel** (e.g., ngrok, cloudflared):

```bash
# Start ngrok tunnel
ngrok http 8080

# In another terminal, set BASE_URL and start app
export SESSION_SECRET=$(openssl rand -base64 32)
export BASE_URL="https://abc123.ngrok.io"  # Use your ngrok URL
./bskyarchive
```

**Note**:
- Environment variable `BASE_URL` overrides the config file setting
- Bluesky OAuth requires a publicly accessible URL - `localhost` won't work for the OAuth callback
- The base URL is logged on startup: `OAuth manager initialized with base URL: https://...`

You can override the config file location:

```bash
CONFIG_PATH=/path/to/config.yaml ./bskyarchive
```

## OAuth Flow

This tool uses Bluesky's OAuth 2.0 with PKCE flow via the [bskyoauth](https://github.com/shindakun/bskyoauth) library.

**No client ID or client secret is required** - the OAuth flow uses your application's base URL (`https://{{ngrok.url}}`) as the client identifier. This is a simpler, more secure approach than traditional OAuth.

## Architecture

- **Language**: Go 1.21+
- **Database**: SQLite with FTS5 full-text search
- **Web**: chi router, html/template, HTMX, Pico CSS
- **Auth**: bskyoauth (Bluesky OAuth 2.0 with PKCE)
- **AT Protocol**: indigo SDK

## Project Structure

```
bskyarchive/
├── cmd/bskyarchive/      # Main application
├── internal/
│   ├── auth/             # OAuth & session management
│   ├── config/           # Configuration loading
│   ├── models/           # Data models
│   ├── storage/          # Database operations
│   └── web/
│       ├── handlers/     # HTTP handlers
│       ├── middleware/   # HTTP middleware
│       ├── templates/    # HTML templates
│       └── static/       # CSS, JS, images
├── specs/                # Feature specifications
└── config.yaml           # Configuration file
```

## Development

### Building

```bash
go build -o bskyarchive ./cmd/bskyarchive
```

### Testing

```bash
go test ./...
```

### Code Style

Follow standard Go conventions. Use `gofmt` for formatting.

## Security

### Overview

This application implements comprehensive security hardening for production deployment with ngrok HTTPS. All security features are enabled by default and work automatically when deployed with HTTPS.

### Security Features

**CSRF Protection (Cross-Site Request Forgery)**
- All POST/PUT/DELETE endpoints require valid CSRF tokens
- Automatic token injection in HTML forms and HTMX requests
- OAuth login endpoints are exempt (OAuth has its own protection)
- Invalid tokens return HTTP 403 Forbidden

**Secure Session Cookies**
- HTTP-only cookies prevent XSS access to session tokens
- Secure flag automatically set when deployed with HTTPS (ngrok)
- SameSite=Lax for OAuth compatibility
- 7-day expiration (configurable)
- Signed with SESSION_SECRET for integrity

**Security Headers**
- `X-Frame-Options: DENY` - Prevents clickjacking attacks
- `X-Content-Type-Options: nosniff` - Prevents MIME-sniffing
- `X-XSS-Protection: 1; mode=block` - Legacy XSS protection
- `Referrer-Policy: strict-origin-when-cross-origin` - Controls referrer info
- `Content-Security-Policy` - Restricts resource loading to same origin
- `Strict-Transport-Security` - Forces HTTPS (auto-enabled with ngrok)

**Request Size Limits**
- Maximum request body size: 10MB (configurable)
- Requests exceeding limit receive HTTP 413 (Payload Too Large)
- Protects against denial-of-service attacks

**Path Traversal Protection**
- All file serving operations validate paths
- Prevents access to files outside designated directories
- Blocks `../` and URL-encoded traversal attempts
- Security logging for blocked attempts

**Export Directory Isolation**
- Per-user export directories: `exports/{did}/timestamp/`
- Users can only access their own exports
- Job ownership verification on all export operations
- Security logging for unauthorized access attempts

### Configuration

Security settings are in `config.yaml`. See the file for detailed comments on each option:

```yaml
server:
  security:
    # CSRF Protection
    csrf_enabled: true
    csrf_field_name: "csrf_token"

    # Request Size Limit (10MB)
    max_request_bytes: 10485760

    # Security Headers (see config.yaml for full details)
    headers:
      x_frame_options: "DENY"
      x_content_type_options: "nosniff"
      # ... additional headers

oauth:
  # Cookie Security
  cookie_secure: "auto"      # Auto-detect HTTPS from BASE_URL
  cookie_samesite: "lax"     # Required for OAuth compatibility
```

### Production Deployment with ngrok

**Recommended Setup:**

1. **Generate session secret**
   ```bash
   export SESSION_SECRET=$(openssl rand -base64 32)
   ```

2. **Start ngrok tunnel**
   ```bash
   ngrok http 8080
   ```

3. **Set BASE_URL and start application**
   ```bash
   export BASE_URL="https://your-subdomain.ngrok.app"
   ./bskyarchive
   ```

**Security Checklist:**
- ✓ ngrok provides HTTPS/TLS 1.3 termination
- ✓ BASE_URL configured with https:// URL
- ✓ SESSION_SECRET set to strong random value (32+ chars)
- ✓ All security features enabled in config.yaml (default)
- ✓ Application logs show "Cookie security enabled: true"

**Architecture:**
- ngrok handles: HTTPS termination, TLS certificates, public endpoint
- Application handles: CSRF, secure cookies, security headers, authorization
- Application runs: HTTP on localhost:8080 (behind ngrok proxy)

### Additional Security Practices

- **Secrets Management**: Never commit SESSION_SECRET to version control
- **Database Security**: SQLite uses WAL mode for safe concurrent access
- **OAuth Security**: PKCE (Proof Key for Code Exchange) prevents authorization code interception
- **Local-First**: All archived data stays on your machine
- **No Telemetry**: Application doesn't phone home or send analytics

### Security Monitoring

The application logs security-relevant events:
- CSRF token validation failures
- Path traversal attempts (blocked)
- Unauthorized export access attempts
- Cookie security configuration on startup

Check logs regularly for suspicious activity.

## Core Principles

1. **Data Privacy & Local-First**: All data stays on your machine
2. **Comprehensive & Accurate**: Complete archive of your content
3. **Multiple Export Formats**: JSON and CSV exports with media and date filtering
4. **Fast & Efficient**: Full-text search with SQLite FTS5
5. **Incremental Operations**: Only fetch new content on updates

## License

See LICENSE file for details.

## Author

Steve Layton
- GitHub: [@shindakun](https://github.com/shindakun)
- Bluesky: [@shindakun.net](https://bsky.app/profile/shindakun.net)
