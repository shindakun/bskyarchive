# Research: Web Interface

**Feature**: Web Interface (001-web-interface)
**Date**: 2025-10-30
**Status**: Complete

## Overview

This document captures research decisions for implementing the web interface layer of the Bluesky Personal Archive Tool. All technology choices align with user requirements (Go, HTML, vanilla JavaScript, HTMX) and the project constitution.

## Technology Stack Decisions

### 1. HTTP Server & Routing

**Decision**: Use Go's standard library `net/http` with a lightweight router (gorilla/mux or chi)

**Rationale**:
- Constitution specifies net/http stdlib
- Proven, stable, and well-documented
- No need for heavy framework overhead
- Lightweight routers like chi or gorilla/mux add minimal complexity while improving routing ergonomics
- Full control over middleware chain

**Alternatives Considered**:
- **Gin/Echo**: Rejected - full frameworks add unnecessary complexity for a localhost-only tool
- **Pure net/http with ServeMux**: Considered but limited routing features (no URL parameters, middleware chaining is verbose)

**Best Practices**:
- Use chi router for clean middleware composition and route grouping
- Implement graceful shutdown
- Use context for request-scoped values
- Table-driven handler tests

### 2. HTML Templating

**Decision**: Use Go's `html/template` package with a layout/partial system

**Rationale**:
- Built into Go standard library
- Auto-escapes HTML to prevent XSS
- Supports template inheritance via `{{define}}` and `{{template}}`
- Familiar to Go developers

**Template Structure**:
```
templates/
├── layouts/base.html      # Common structure (<!DOCTYPE>, head, body wrapper)
├── pages/*.html           # Full page templates
└── partials/*.html        # Reusable components (nav, footer, cards)
```

**Best Practices**:
- Use `template.Must()` to catch template parsing errors at startup
- Pre-compile templates once at server initialization
- Pass data via struct types (not `map[string]interface{}`)
- Use `{{block}}` for layout extension points

### 3. Session Management

**Decision**: Use gorilla/sessions with secure cookie store for 7-day expiring sessions

**Rationale**:
- Industry-standard session library for Go
- Supports encrypted cookie storage (appropriate for localhost single-user)
- Configurable expiration (required: 7 days)
- Flash message support for user feedback

**Security Considerations**:
- HTTP-only cookies to prevent XSS access
- Secure flag for cookies (even on localhost for best practices)
- SameSite=Lax to prevent CSRF
- Rotate session ID on authentication
- Store minimal data in session (user DID, handle, expiration)

**Alternatives Considered**:
- **JWT tokens**: Rejected - more complex, harder to invalidate, unnecessary for single-user localhost
- **Server-side session store (Redis/memcached)**: Rejected - overkill for single-user, adds external dependency

### 4. OAuth Integration

**Decision**: Use existing `github.com/shindakun/bskyoauth` package

**Rationale**:
- Already specified by user as project requirement
- Handles Bluesky OAuth 2.0 flow
- Manages token refresh

**Implementation Pattern**:
1. `/login` → initiate OAuth flow → redirect to Bluesky
2. `/callback` → receive auth code → exchange for tokens → create session → redirect to dashboard
3. Store tokens in session encrypted with gorilla/sessions
4. Middleware checks session validity on protected routes

**Error Handling**:
- OAuth denial: Redirect to landing with friendly message
- Token expiration: Auto-refresh if possible, else redirect to login
- Network errors: Display user-friendly error page with retry option

### 5. AT Protocol Integration & Data Collection

**Decision**: Use `github.com/bluesky-social/indigo` (official AT Protocol SDK for Go) for all Bluesky API interactions

**Rationale**:
- Official SDK from Bluesky team
- Comprehensive AT Protocol support (posts, profiles, media, social graph)
- Handles XRPC (cross-service RPC protocol)
- Built-in type safety with generated lexicons
- Active maintenance and community support

**Architecture**:
```
User triggers sync → Background worker starts → AT Protocol client fetches data → Store in SQLite
```

**Data Collection Strategy**:

1. **Full Sync** (first-time archive):
   - Fetch all posts via `app.bsky.feed.getAuthorFeed` (paginated, 100 per page)
   - Download embedded media (images, videos)
   - Fetch profile snapshots via `app.bsky.actor.getProfile`
   - Fetch social graph (followers/following) via `app.bsky.graph` endpoints
   - Store everything in SQLite with FTS5 index

2. **Incremental Sync** (subsequent syncs):
   - Fetch posts newer than `lastSyncTime`
   - Check for updates to existing posts (edits, deleted posts)
   - Fetch new media only
   - Update profile if changed
   - Much faster than full sync

**Rate Limiting**:
- Bluesky API limit: **300 requests per 5 minutes** (per OAuth token)
- Strategy: Respect `RateLimit-*` headers from API responses
- Implement exponential backoff on 429 (Too Many Requests)
- Show estimated time remaining in progress UI

**Implementation**:
```go
import (
  "github.com/bluesky-social/indigo/api/atproto"
  "github.com/bluesky-social/indigo/api/bsky"
  "github.com/bluesky-social/indigo/xrpc"
)

// Create authenticated client
client := &xrpc.Client{
  Host: "https://bsky.social",
  Auth: &xrpc.AuthInfo{
    AccessJwt:  session.AccessToken,
    RefreshJwt: session.RefreshToken,
    Did:        session.DID,
    Handle:     session.Handle,
  },
}

// Fetch user's posts (paginated)
resp, err := bsky.FeedGetAuthorFeed(ctx, client, did, "", limit, cursor)
```

**Error Handling**:
- **401 Unauthorized**: Token expired → attempt refresh → redirect to login if refresh fails
- **429 Too Many Requests**: Rate limited → wait and retry with exponential backoff
- **Network errors**: Retry with exponential backoff (max 3 attempts)
- **Partial failures**: Continue sync, log errors, show summary at end

**Media Download**:
- Extract media URLs from post embed objects
- Download in parallel (max 5 concurrent)
- Store with content-addressable filenames (hash-based)
- Organize by year/month: `archive/media/2024/10/abc123.jpg`
- Track download progress for real-time UI updates

**Alternatives Considered**:
- **Direct XRPC calls without SDK**: Rejected - reinventing the wheel, prone to errors
- **Unofficial Go clients**: Rejected - indigo is official and most comprehensive
- **Firehose streaming**: Rejected - overkill for personal archive, requires persistent connection

**Best Practices**:
- Use context for cancellation (user can stop sync mid-operation)
- Implement resume capability (store cursor position)
- Transaction-based database writes (atomic commits)
- Validate data before storage (schema validation)

### 6. SQLite Storage & Full-Text Search

**Decision**: Use `modernc.org/sqlite` (pure Go SQLite) with FTS5 for full-text search

**Rationale**:
- Pure Go implementation (no CGO required)
- Cross-platform (Windows, macOS, Linux)
- FTS5 built-in for fast full-text search (<100ms per constitution requirement)
- Single-file database (easy backup)
- ACID compliance (data integrity)

**Schema Design**:

```sql
-- Posts table
CREATE TABLE posts (
  uri TEXT PRIMARY KEY,
  cid TEXT NOT NULL,
  did TEXT NOT NULL,
  text TEXT,
  created_at TIMESTAMP NOT NULL,
  indexed_at TIMESTAMP NOT NULL,
  has_media BOOLEAN DEFAULT 0,
  like_count INTEGER DEFAULT 0,
  repost_count INTEGER DEFAULT 0,
  reply_count INTEGER DEFAULT 0,
  is_reply BOOLEAN DEFAULT 0,
  reply_parent TEXT,
  embed_type TEXT,  -- 'images', 'external', 'record', null
  embed_data JSON,  -- Full embed structure
  labels JSON,      -- Content labels
  archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_posts_did ON posts(did);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);

-- Full-text search virtual table
CREATE VIRTUAL TABLE posts_fts USING fts5(
  uri UNINDEXED,
  text,
  content='posts',
  content_rowid='rowid'
);

-- Trigger to keep FTS in sync
CREATE TRIGGER posts_ai AFTER INSERT ON posts BEGIN
  INSERT INTO posts_fts(rowid, uri, text) VALUES (new.rowid, new.uri, new.text);
END;

-- Media table
CREATE TABLE media (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_uri TEXT NOT NULL,
  media_url TEXT NOT NULL,
  local_path TEXT NOT NULL,
  alt_text TEXT,
  mime_type TEXT,
  size_bytes INTEGER,
  downloaded_at TIMESTAMP,
  FOREIGN KEY (post_uri) REFERENCES posts(uri)
);

-- Profiles table (snapshots over time)
CREATE TABLE profiles (
  did TEXT NOT NULL,
  handle TEXT NOT NULL,
  display_name TEXT,
  description TEXT,
  avatar_url TEXT,
  banner_url TEXT,
  followers_count INTEGER,
  follows_count INTEGER,
  posts_count INTEGER,
  snapshot_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (did, snapshot_at)
);

-- Archive operations tracking
CREATE TABLE operations (
  id TEXT PRIMARY KEY,
  did TEXT NOT NULL,
  type TEXT NOT NULL,  -- 'full' or 'incremental'
  status TEXT NOT NULL,  -- 'queued', 'running', 'completed', 'failed'
  progress REAL DEFAULT 0.0,
  posts_fetched INTEGER DEFAULT 0,
  media_downloaded INTEGER DEFAULT 0,
  started_at TIMESTAMP NOT NULL,
  completed_at TIMESTAMP,
  error_message TEXT,
  cursor TEXT  -- For resume capability
);
```

**Query Examples**:
```sql
-- Full-text search
SELECT p.* FROM posts p
JOIN posts_fts fts ON p.rowid = fts.rowid
WHERE posts_fts MATCH 'bluesky AND protocol'
ORDER BY p.created_at DESC
LIMIT 50;

-- Pagination
SELECT * FROM posts
WHERE did = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- Archive stats
SELECT
  COUNT(*) as total_posts,
  COUNT(CASE WHEN has_media THEN 1 END) as posts_with_media,
  SUM(like_count) as total_likes
FROM posts
WHERE did = ?;
```

**Database Migrations**:
- Use numbered migration files: `001_initial.sql`, `002_add_index.sql`
- Track applied migrations in `schema_migrations` table
- Apply migrations on server startup
- Never modify old migrations (additive only)

**Alternatives Considered**:
- **mattn/go-sqlite3**: Rejected - requires CGO, complicates cross-compilation
- **PostgreSQL**: Rejected - overkill for single-user, requires separate process
- **JSON files**: Rejected - poor query performance, no full-text search

### 7. CSS Framework & Dark Theme

**Decision**: Use Pico CSS v2 (classless CSS framework) with custom dark theme overrides

**Rationale**:
- Minimal, semantic HTML styling (no class soup)
- Built-in dark mode support
- Responsive by default
- Tiny footprint (~10KB gzipped)
- Accessible (WCAG AA compliant)
- Matches "modern and sleek" requirement

**Customization**:
```css
/* custom dark theme overrides in static/css/styles.css */
:root {
  --primary: #1e88e5;       /* Bluesky blue */
  --background-color: #121212;
  --card-background: #1e1e1e;
  --text-color: #e0e0e0;
}
```

**Alternatives Considered**:
- **Tailwind CSS**: Rejected - utility-first approach adds build step and bloat
- **Bootstrap**: Rejected - too heavy, over-engineered for simple 5-page app
- **Custom CSS from scratch**: Considered but reinventing accessibility and responsiveness is inefficient

### 8. HTMX for Dynamic Interactions

**Decision**: Use HTMX v1.9+ for partial page updates and real-time progress

**Rationale**:
- User requirement (specified in plan input)
- Hypermedia-driven approach aligns with server-rendered architecture
- No build step required
- Enables AJAX requests with HTML responses (no JSON serialization)
- Perfect for progress updates, archive status refresh

**Use Cases**:
- **Real-time progress updates**: `hx-get="/api/progress" hx-trigger="every 2s"` polls backend, updates progress bar HTML
- **Archive initiation**: `hx-post="/archive/sync"` submits form without full page reload
- **Archive browsing pagination**: `hx-get="/browse?page=2" hx-target="#posts"` loads next page inline

**Best Practices**:
- Return HTML fragments from endpoints (not JSON)
- Use `hx-swap` strategies for smooth UX (outerHTML, innerHTML, beforeend)
- Implement proper HTTP status codes (200, 4xx, 5xx) for HTMX error handling
- Add fallback: forms still work with JS disabled (progressive enhancement)

### 9. Vanilla JavaScript

**Decision**: Minimal vanilla JS for progressive enhancement only

**Rationale**:
- User requirement
- No build tooling required
- HTMX handles most dynamic behavior
- Use JS only for: client-side form validation, keyboard shortcuts, animations

**Implementation**:
```javascript
// static/js/app.js (~5KB)
document.addEventListener('DOMContentLoaded', function() {
  // Progressive enhancements
  // - Escape key to close modals
  // - Keyboard navigation improvements
  // - Client-side form validation (in addition to server-side)
});
```

**Constraints**:
- Total JS footprint target: <50KB (easily achievable with HTMX + minimal custom JS)
- No bundler (direct `<script>` tags)
- Use ES6+ features (target modern browsers only)

### 10. CSRF Protection

**Decision**: Use Double Submit Cookie pattern with gorilla/csrf middleware

**Rationale**:
- Constitution requires CSRF protection on state-changing operations
- gorilla/csrf integrates seamlessly with gorilla/mux and gorilla/sessions
- Automatic token generation and validation
- Works with HTMX (include token in headers)

**Implementation**:
```go
import "github.com/gorilla/csrf"

csrfMiddleware := csrf.Protect(
  []byte("32-byte-key"),
  csrf.Secure(false), // localhost only
  csrf.Path("/"),
)
```

Templates:
```html
<form method="POST">
  {{ .CSRFField }}
  <!-- form fields -->
</form>
```

HTMX config:
```javascript
document.body.addEventListener('htmx:configRequest', function(evt) {
  evt.detail.headers['X-CSRF-Token'] = document.querySelector('meta[name="csrf-token"]').content;
});
```

### 11. Responsive Design

**Decision**: Mobile-first responsive design using Pico CSS defaults + CSS Grid/Flexbox

**Rationale**:
- User requirement: "site should be responsive"
- Pico CSS is mobile-first by default
- CSS Grid for page layouts, Flexbox for component layouts
- No media query spaghetti

**Breakpoints** (Pico CSS defaults):
- Mobile: <576px
- Tablet: 576px-768px
- Desktop: >768px

**Responsive Patterns**:
- Navigation: Horizontal links on desktop, hamburger/stack on mobile (HTMX-powered toggle)
- Archive cards: CSS Grid with `auto-fit` for fluid columns
- Tables: Responsive table pattern (stack on mobile)

## Archiver & Storage Integration

### Background Worker Architecture

**Decision**: Implement a background worker pattern for long-running sync operations

**Architecture**:
```
Web Handler → Starts Worker (goroutine) → Worker updates progress in DB → HTMX polls progress
```

**Implementation**:
```go
// internal/archiver/worker.go
type Worker struct {
  client   *xrpc.Client
  storage  *storage.Store
  progress *OperationProgress
}

func (w *Worker) Run(ctx context.Context, did string, fullSync bool) error {
  // Update progress: queued → running
  // Fetch posts with pagination
  // For each post: download media, store in DB
  // Update progress periodically (every 10 posts)
  // On completion: queued → completed (or failed)
}
```

**Progress Tracking**:
- Store operation status in `operations` table
- Web handlers read from `operations` table
- Worker writes progress updates atomically
- No in-memory state (survives restarts)

**Concurrency Control**:
- Only one sync operation per user at a time
- Check for existing running operation before starting new one
- Use database row-level locking (`SELECT FOR UPDATE`)

**Rationale**: Decouples long-running archival from HTTP request lifecycle

## Security Considerations

### Token Storage

**Decision**: Store OAuth tokens encrypted in session cookies

**Rationale**:
- Localhost-only reduces attack surface
- gorilla/sessions encrypts cookie values
- Alternative (server-side store) adds complexity without significant benefit for single-user

### Session Expiration

**Decision**: 7-day rolling expiration (per requirement FR-018)

**Implementation**:
- Set MaxAge on cookie: 7 days (604800 seconds)
- Each request resets expiration (rolling window)
- After 7 days of inactivity, session expires → redirect to login

### HTTPS

**Decision**: Optional HTTPS support with self-signed cert for localhost

**Rationale**:
- Localhost generally uses HTTP
- Some OAuth flows require HTTPS callback
- If bskyoauth requires HTTPS: Use Go's `http.ListenAndServeTLS()` with generated self-signed cert

## Testing Strategy

### Unit Tests

**Scope**: Handler logic, middleware, template rendering

**Approach**:
- Table-driven tests with `httptest.ResponseRecorder`
- Mock archive service interface
- Test each handler's happy path + error cases

**Example**:
```go
func TestLandingHandler(t *testing.T) {
  tests := []struct{
    name string
    authenticated bool
    wantStatus int
    wantRedirect string
  }{
    {"unauthenticated shows landing", false, 200, ""},
    {"authenticated redirects to dashboard", true, 302, "/dashboard"},
  }
  // ... table-driven test implementation
}
```

### Integration Tests

**Scope**: OAuth flow end-to-end

**Approach**:
- Spin up test server with bskyoauth in test mode (mock OAuth provider)
- Simulate full login flow: landing → login → callback → dashboard
- Verify session creation and cookie setting

**Tools**:
- `httptest.Server` for test HTTP server
- Mock OAuth provider or bskyoauth test fixtures

### Manual Testing

**Scope**: Visual design, responsive behavior, HTMX interactions

**Approach**:
- Test in multiple browsers (Chrome, Firefox, Safari, Edge)
- Test responsive breakpoints using browser dev tools
- Verify WCAG AA contrast for dark theme (use browser extensions)

## Performance Considerations

### Template Caching

**Decision**: Parse and cache templates once at server startup

**Implementation**:
```go
var templates *template.Template

func init() {
  templates = template.Must(template.ParseGlob("templates/**/*.html"))
}
```

### Static Asset Serving

**Decision**: Serve static assets with caching headers

**Implementation**:
```go
fs := http.FileServer(http.Dir("static"))
http.Handle("/static/", http.StripPrefix("/static/", fs))
```

**Caching**:
- Set `Cache-Control: max-age=31536000` for versioned assets (CSS/JS with hash in filename)
- Set `Cache-Control: no-cache` for HTML

### Archive Browse Pagination

**Decision**: Server-side pagination with configurable page size (default 50 posts)

**Rationale**:
- Supports 10,000+ posts efficiently
- Reduces memory footprint
- Fast page loads

**Implementation**:
- URL param: `/browse?page=2&size=50`
- Backend fetches only requested page from SQLite
- HTMX loads next page on scroll or click

## Deployment & Configuration

### Configuration

**Decision**: YAML config file with reasonable defaults

**Config Schema**:
```yaml
server:
  host: "localhost"
  port: 8080
  read_timeout: 15s
  write_timeout: 15s

session:
  secret_key: "random-32-byte-key"  # auto-generated if not provided
  max_age_days: 7

oauth:
  client_id: "from-bluesky"
  client_secret: "from-bluesky"
  redirect_url: "http://localhost:8080/callback"

archive:
  data_path: "./archive"
```

### Running the Server

**Command**:
```bash
# Build
go build -o bskyarchive-web cmd/web/main.go

# Run
./bskyarchive-web --config config.yaml
```

**Environment Variables** (optional overrides):
```bash
BSKY_CLIENT_ID=xxx
BSKY_CLIENT_SECRET=xxx
BSKY_SESSION_SECRET=xxx
```

## Open Questions & Future Enhancements

### Resolved (No Clarification Needed)

- **CSS framework**: Decided on Pico CSS
- **Session storage**: Decided on encrypted cookies
- **HTMX version**: v1.9+ (latest stable)

### Future Enhancements (Out of Scope for MVP)

- Real-time WebSocket updates instead of polling
- Advanced search filtering UI
- Export format generation from web UI
- Multi-user support (requires authentication beyond OAuth)
- Theming system (light/dark/custom themes)

## Summary

All technical decisions finalized:
- **Backend**: Go 1.21+ with net/http, chi router, gorilla/sessions, bskyoauth
- **AT Protocol**: indigo SDK for Bluesky API integration, rate limiting, media download
- **Storage**: modernc.org/sqlite with FTS5, database migrations, local media storage
- **Archival**: Background worker pattern, progress tracking in DB, resumable syncs
- **Frontend**: HTML templates, Pico CSS (dark theme), HTMX, minimal vanilla JS
- **Security**: Encrypted sessions, CSRF protection, OAuth 2.0, 7-day expiration
- **Testing**: Unit tests (handlers, collector, storage), integration tests (OAuth, AT Protocol)
- **Performance**: Template caching, static asset optimization, server-side pagination, FTS5 search

No NEEDS CLARIFICATION items remain. Ready to proceed to Phase 1 (data models and contracts).
