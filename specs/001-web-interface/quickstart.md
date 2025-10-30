# Quickstart Guide: Web Interface Development

**Feature**: Web Interface (001-web-interface)
**Date**: 2025-10-30
**For**: Developers implementing this feature

## Overview

This guide provides step-by-step instructions for developing the web interface feature for the Bluesky Personal Archive Tool. Follow this guide to set up your development environment, understand the architecture, and implement the feature according to the plan.

## Prerequisites

- **Go**: Version 1.21 or higher
- **Git**: For version control
- **Code Editor**: VS Code, GoLand, or your preferred editor
- **Browser**: Modern browser for testing (Chrome, Firefox, Safari, or Edge)
- **Bluesky Account**: For testing OAuth flow

## Project Structure

```
bskyarchive/
├── cmd/
│   └── web/
│       └── main.go                # Web server entry point
├── internal/
│   ├── web/
│   │   ├── handlers/              # HTTP handlers
│   │   ├── middleware/            # Auth, session, CSRF middleware
│   │   ├── templates/             # HTML templates
│   │   ├── static/                # CSS, JS, images
│   │   └── server.go              # Server setup
│   ├── auth/
│   │   ├── oauth.go               # bskyoauth integration
│   │   └── session.go             # Session management
│   └── archive/
│       └── client.go              # Archive service client
├── tests/
│   ├── integration/
│   └── unit/
├── specs/
│   └── 001-web-interface/
│       ├── spec.md                # Feature specification
│       ├── plan.md                # Implementation plan
│       ├── research.md            # Technology research
│       ├── data-model.md          # Data models
│       ├── contracts/             # API contracts
│       └── quickstart.md          # This file
├── go.mod
├── go.sum
└── config.yaml                    # Configuration file
```

## Setup Steps

### 1. Install Dependencies

```bash
# Navigate to project root
cd /Users/steve/go/src/github.com/shindakun/bskyarchive

# Install Go dependencies
go get github.com/shindakun/bskyoauth
go get github.com/go-chi/chi/v5
go get github.com/gorilla/sessions
go get github.com/gorilla/csrf

# Download HTMX and Pico CSS (place in internal/web/static/)
# HTMX: https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js
# Pico CSS: https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css
```

### 2. Create Configuration File

Create `config.yaml` in project root:

```yaml
server:
  host: "localhost"
  port: 8080
  read_timeout: 15s
  write_timeout: 15s
  shutdown_timeout: 30s

session:
  secret_key: ""  # Auto-generated if empty
  max_age_days: 7
  cookie_name: "bsky_session"

oauth:
  client_id: "YOUR_BLUESKY_CLIENT_ID"
  client_secret: "YOUR_BLUESKY_CLIENT_SECRET"
  redirect_url: "http://localhost:8080/auth/callback"
  scopes:
    - "atproto"

archive:
  data_path: "./archive"
  sqlite_path: "./archive/index.db"

logging:
  level: "info"  # debug, info, warn, error
  format: "json"  # json or text
```

### 3. Set Up Directory Structure

```bash
# Create directories
mkdir -p cmd/web
mkdir -p internal/web/{handlers,middleware,templates/{layouts,pages,partials},static/{css,js,images}}
mkdir -p internal/auth
mkdir -p internal/archive
mkdir -p tests/{unit,integration}

# Create empty files to start with
touch cmd/web/main.go
touch internal/web/server.go
touch internal/auth/oauth.go
touch internal/auth/session.go
touch internal/archive/client.go
```

### 4. Download Static Assets

```bash
# Download HTMX
curl -o internal/web/static/js/htmx.min.js \
  https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js

# Download Pico CSS
curl -o internal/web/static/css/pico.min.css \
  https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css

# Create custom styles file
touch internal/web/static/css/styles.css
touch internal/web/static/js/app.js
```

## Implementation Order

Follow this order to build the feature incrementally:

### Phase 1: Foundation (User Story 1 - P1)

**Goal**: Get basic server running with landing page and OAuth flow

1. **Set up HTTP server** (`cmd/web/main.go`, `internal/web/server.go`)
   - Load configuration
   - Initialize chi router
   - Set up middleware stack
   - Graceful shutdown

2. **Create base template** (`internal/web/templates/layouts/base.html`)
   - HTML5 boilerplate
   - Link to Pico CSS and custom styles
   - Include HTMX script
   - CSRF meta tag

3. **Implement session management** (`internal/auth/session.go`)
   - Configure gorilla/sessions with encryption
   - Session creation/destruction helpers
   - 7-day expiration logic

4. **Implement OAuth flow** (`internal/auth/oauth.go`)
   - Initialize bskyoauth client
   - Login handler (initiate OAuth)
   - Callback handler (exchange code for tokens)
   - Store tokens in session

5. **Create landing page** (`internal/web/handlers/landing.go`, `internal/web/templates/pages/landing.html`)
   - Display project description
   - Login button (POST to /auth/login)
   - Show error messages if OAuth fails
   - Redirect to dashboard if already authenticated

6. **Implement authentication middleware** (`internal/web/middleware/auth.go`)
   - Check session validity
   - Redirect to landing if unauthorized
   - Extend session expiration on each request (rolling window)

**Checkpoint**: You should be able to:
- Visit http://localhost:8080
- See the landing page
- Click login and complete OAuth flow
- Be redirected to /dashboard (stub page for now)
- See session cookie in browser dev tools

### Phase 2: Archive Management (User Story 2 - P2)

**Goal**: Display archive status and allow users to initiate syncs

1. **Create archive service client** (`internal/archive/client.go`)
   - Define ArchiveService interface
   - Implement methods to communicate with archival backend
   - Mock implementation for testing

2. **Create dashboard page** (`internal/web/handlers/dashboard.go`, `internal/web/templates/pages/dashboard.html`)
   - Fetch archive status from backend
   - Display stats (total posts, media, last sync)
   - Show 5 most recent posts
   - Show active operation progress (if any)

3. **Create archive management page** (`internal/web/handlers/archive.go`, `internal/web/templates/pages/archive.html`)
   - Display detailed archive status
   - Form to initiate sync (full or incremental)
   - Show progress bar if sync is running
   - Disable form while sync is in progress

4. **Implement sync initiation endpoint** (`internal/web/handlers/archive.go` - POST /api/archive/sync)
   - Validate CSRF token
   - Parse sync type (full/incremental)
   - Call archive service to start sync
   - Return HTML fragment with progress indicator

5. **Implement progress polling endpoint** (`internal/web/handlers/archive.go` - GET /api/progress/:id)
   - Fetch operation status from archive service
   - Return HTML fragment with progress bar
   - Update status (queued, running, completed, failed)
   - Use HTMX polling (`hx-trigger="every 2s"`)

6. **Create archive browse page** (`internal/web/handlers/browse.go`, `internal/web/templates/pages/browse.html`)
   - Fetch paginated posts from archive service
   - Display posts in cards (text, metadata, engagement stats)
   - Implement pagination (previous/next links)
   - Use HTMX for infinite scroll or click-to-load-more

**Checkpoint**: You should be able to:
- Log in and see dashboard with archive stats
- Navigate to archive management page
- Initiate a sync operation (full or incremental)
- See real-time progress updates
- Browse archived posts with pagination

### Phase 3: About Page (User Story 3 - P3)

**Goal**: Add informational about page

1. **Create about page** (`internal/web/handlers/about.go`, `internal/web/templates/pages/about.html`)
   - Project description
   - Version number
   - Link to GitHub repository (opens in new tab)
   - Link to author's Bluesky profile (opens in new tab)
   - Consistent styling with other pages

**Checkpoint**: You should be able to:
- Navigate to /about
- See project information
- Click links to GitHub and Bluesky (open in new tabs)

### Phase 4: Styling & Polish

**Goal**: Apply dark theme and responsive design

1. **Create custom dark theme** (`internal/web/static/css/styles.css`)
   - Override Pico CSS variables for dark theme
   - Define color palette (Bluesky blue accent)
   - Ensure WCAG AA contrast compliance
   - Add custom styles for post cards, progress bars, etc.

2. **Add progressive enhancements** (`internal/web/static/js/app.js`)
   - Client-side form validation
   - Keyboard shortcuts (Escape to close modals)
   - Smooth scroll behavior
   - HTMX error handling

3. **Test responsive design**
   - Test all pages at mobile, tablet, and desktop breakpoints
   - Adjust navigation for mobile (stack or hamburger)
   - Ensure post cards reflow properly
   - Test on actual devices or browser dev tools

4. **Add navigation partial** (`internal/web/templates/partials/nav.html`)
   - Links to Dashboard, Archive, Browse, About
   - Show user's display name
   - Logout button
   - Responsive menu

**Checkpoint**: You should be able to:
- See consistent dark theme across all pages
- Use the app on mobile, tablet, and desktop
- Navigate between pages easily
- Log out successfully

### Phase 5: Error Handling & Edge Cases

**Goal**: Handle errors gracefully

1. **Implement error pages** (404, 500)
   - Create error templates
   - User-friendly error messages
   - Link back to safe pages

2. **Handle OAuth errors**
   - User denies authorization
   - Token exchange fails
   - Network errors during OAuth

3. **Handle archive operation errors**
   - Operation fails mid-sync
   - Network interruption
   - Display error messages with retry option

4. **Handle session expiration**
   - Detect expired sessions
   - Redirect to landing with message
   - Clear expired session cookie

**Checkpoint**: All edge cases handled gracefully.

### Phase 6: Testing

**Goal**: Ensure feature works correctly

1. **Write unit tests** (`tests/unit/`)
   - Test each handler with table-driven tests
   - Mock archive service
   - Test middleware (auth, session, CSRF)

2. **Write integration tests** (`tests/integration/`)
   - Test full OAuth flow
   - Test sync initiation and progress polling
   - Test session expiration

3. **Manual testing**
   - Test in multiple browsers
   - Test responsive behavior
   - Test accessibility with screen reader
   - Verify WCAG AA contrast

**Checkpoint**: All tests pass, feature fully functional.

## Development Tips

### Running the Server

```bash
# Run with default config
go run cmd/web/main.go

# Run with custom config
go run cmd/web/main.go --config custom-config.yaml

# Run with environment variable overrides
BSKY_CLIENT_ID=xxx BSKY_CLIENT_SECRET=yyy go run cmd/web/main.go
```

### Hot Reloading (Optional)

Use `air` for live reloading during development:

```bash
# Install air
go install github.com/cosmtrek/air@latest

# Create .air.toml config
air init

# Run with air
air
```

### Template Development

Templates are parsed once at startup. To reload templates without restarting:

```go
// In development mode, re-parse templates on each request
if os.Getenv("ENV") == "development" {
    templates = template.Must(template.ParseGlob("internal/web/templates/**/*.html"))
}
```

### Testing OAuth Locally

If bskyoauth requires HTTPS for callbacks:

```bash
# Generate self-signed cert
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Update config.yaml
oauth:
  redirect_url: "https://localhost:8080/auth/callback"

# Run server with TLS
# Update main.go to use http.ListenAndServeTLS
```

### Debugging HTMX

Enable HTMX debug logging in templates:

```html
<script>
  htmx.logAll();  // Log all HTMX events to console
</script>
```

### Inspecting Sessions

Use browser dev tools:
- **Application tab** (Chrome/Edge) or **Storage tab** (Firefox)
- Look for `bsky_session` cookie
- Note: Cookie value is encrypted, so you'll only see ciphertext

## Common Issues & Solutions

### Issue: OAuth redirect fails

**Solution**: Check that `redirect_url` in config matches the one registered with Bluesky OAuth.

### Issue: CSRF token mismatch

**Solution**: Ensure forms include `{{.CSRFToken}}` hidden field or HTMX requests include `X-CSRF-Token` header.

### Issue: Session expires immediately

**Solution**: Check that `session.max_age_days` is set correctly and server time is accurate.

### Issue: Templates not updating

**Solution**: Make sure you're re-parsing templates (see Hot Reloading section).

### Issue: HTMX not working

**Solution**: Check browser console for errors. Ensure `htmx.min.js` is loaded and HTMX attributes are correct.

### Issue: Dark theme not applying

**Solution**: Ensure `styles.css` is loaded after `pico.min.css` so overrides take effect.

## Configuration Examples

### Development Config

```yaml
server:
  host: "localhost"
  port: 8080

session:
  secret_key: "dev-secret-key-not-for-production"
  max_age_days: 7

oauth:
  client_id: "dev-client-id"
  client_secret: "dev-client-secret"
  redirect_url: "http://localhost:8080/auth/callback"

logging:
  level: "debug"
  format: "text"
```

### Production Config

```yaml
server:
  host: "localhost"
  port: 8080
  read_timeout: 30s
  write_timeout: 30s

session:
  secret_key: ""  # Auto-generated 32-byte key
  max_age_days: 7

oauth:
  client_id: "${BSKY_CLIENT_ID}"
  client_secret: "${BSKY_CLIENT_SECRET}"
  redirect_url: "http://localhost:8080/auth/callback"

logging:
  level: "info"
  format: "json"
```

## Next Steps

After completing this quickstart:

1. **Run `/speckit.tasks`** to generate detailed task list
2. **Implement Phase 1** (Authentication & Landing)
3. **Test Phase 1** before moving to Phase 2
4. **Iterate** through each phase
5. **Commit frequently** with clear commit messages
6. **Push to both remotes** (origin and tangled)

## Resources

- **Spec**: [spec.md](spec.md)
- **Plan**: [plan.md](plan.md)
- **Research**: [research.md](research.md)
- **Data Model**: [data-model.md](data-model.md)
- **API Contracts**: [contracts/http-api.md](contracts/http-api.md)

**External**:
- [Go net/http docs](https://pkg.go.dev/net/http)
- [chi router docs](https://go-chi.io/)
- [HTMX docs](https://htmx.org/docs/)
- [Pico CSS docs](https://picocss.com/)
- [gorilla/sessions docs](https://github.com/gorilla/sessions)
- [bskyoauth docs](https://github.com/shindakun/bskyoauth)

## Summary

This quickstart guide provides:
- Project structure overview
- Step-by-step setup instructions
- Implementation order (3 phases aligned with user stories)
- Development tips and common issues
- Configuration examples

Follow this guide to build the web interface feature according to the specification and plan. Good luck! 🚀
