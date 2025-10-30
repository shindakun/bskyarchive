# Research: Web Interface

**Phase 0 Output** | **Date**: 2025-10-30 | **Plan**: [plan.md](./plan.md)

## Overview

This document consolidates all technical research, design decisions, and implementation patterns for the Bluesky Personal Archive Tool web interface. It resolves unknowns from the Technical Context and establishes best practices for all dependencies.

---

## 1. Go HTTP Server & Routing

### Decision: net/http + gorilla/mux or chi router

**Rationale**:
- Use Go's standard library `net/http` as the foundation
- Add `chi` router for cleaner route definitions and middleware support
- Avoids heavyweight frameworks while providing essential routing features
- Chi is lightweight, idiomatic Go, and widely adopted

**Best Practices**:
```go
import (
  "net/http"
  "github.com/go-chi/chi/v5"
  "github.com/go-chi/chi/v5/middleware"
)

func NewRouter() *chi.Mux {
  r := chi.NewRouter()

  // Middleware
  r.Use(middleware.Logger)
  r.Use(middleware.Recoverer)
  r.Use(middleware.Compress(5))

  // Public routes
  r.Get("/", handlers.Landing)
  r.Get("/about", handlers.About)
  r.Get("/auth/login", handlers.OAuthLogin)
  r.Get("/auth/callback", handlers.OAuthCallback)

  // Protected routes
  r.Group(func(r chi.Router) {
    r.Use(auth.RequireAuth)
    r.Get("/dashboard", handlers.Dashboard)
    r.Get("/archive", handlers.Archive)
    r.Post("/archive/start", handlers.StartArchive)
    r.Get("/archive/status", handlers.ArchiveStatus)
    r.Get("/browse", handlers.Browse)
  })

  // Static assets
  r.Handle("/static/*", http.StripPrefix("/static/",
    http.FileServer(http.Dir("internal/web/static"))))

  return r
}
```

**Alternatives Considered**:
- Echo/Gin: Too opinionated, unnecessary features
- stdlib only: Verbose routing, harder middleware composition

---

## 2. HTML Templating

### Decision: Go html/template with layout inheritance

**Rationale**:
- Standard library, no external dependencies
- Automatic HTML escaping prevents XSS
- Template composition via `{{template}}` and `{{block}}`
- Integrates seamlessly with HTMX partial responses

**Best Practices**:
```go
// Template structure
// internal/web/templates/
// ├── layouts/
// │   └── base.html        (shell with {{block "content" .}})
// ├── pages/
// │   ├── landing.html     ({{define "content"}}...{{end}})
// │   ├── dashboard.html
// │   └── archive.html
// └── partials/
//     ├── nav.html
//     └── archive-status.html

// Loading templates
var templates *template.Template

func init() {
  templates = template.Must(template.ParseGlob("internal/web/templates/**/*.html"))
}

// Rendering with layout
func RenderPage(w http.ResponseWriter, name string, data interface{}) error {
  return templates.ExecuteTemplate(w, name, data)
}

// HTMX partial response (no layout)
func RenderPartial(w http.ResponseWriter, name string, data interface{}) error {
  return templates.ExecuteTemplate(w, name, data)
}
```

**Template Example** (base.html):
```html
<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{block "title" .}}Bluesky Archive{{end}}</title>
  <link rel="stylesheet" href="/static/css/pico.min.css">
  <link rel="stylesheet" href="/static/css/custom.css">
  <script src="/static/js/htmx.min.js"></script>
</head>
<body>
  {{template "nav" .}}
  <main class="container">
    {{block "content" .}}{{end}}
  </main>
  <script src="/static/js/app.js"></script>
</body>
</html>
```

**Alternatives Considered**:
- Templ: Type-safe but adds build complexity
- Manual string concatenation: Error-prone, no escaping

---

## 3. Session Management

### Decision: gorilla/sessions with encrypted cookie store

**Rationale**:
- Proven library with secure defaults
- Encrypted cookies avoid database round-trips
- Built-in flash message support
- Easy integration with middleware

**Best Practices**:
```go
import (
  "github.com/gorilla/sessions"
)

var store *sessions.CookieStore

func InitSessions(secret []byte) {
  store = sessions.NewCookieStore(secret)
  store.Options = &sessions.Options{
    Path:     "/",
    MaxAge:   7 * 24 * 60 * 60, // 7 days
    HttpOnly: true,
    Secure:   false, // true in production with HTTPS
    SameSite: http.SameSiteLaxMode,
  }
}

// Store session data
func SaveSession(w http.ResponseWriter, r *http.Request, userID, handle, did, accessToken string) error {
  session, _ := store.Get(r, "auth")
  session.Values["user_id"] = userID
  session.Values["handle"] = handle
  session.Values["did"] = did
  session.Values["access_token"] = accessToken
  session.Values["authenticated"] = true
  return session.Save(r, w)
}

// Middleware: require authentication
func RequireAuth(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "auth")
    if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
      http.Redirect(w, r, "/?error=auth_required", http.StatusSeeOther)
      return
    }
    next.ServeHTTP(w, r)
  })
}
```

**Security Considerations**:
- Generate secret key with `crypto/rand` (32 bytes)
- Store secret in environment variable or config file (not in code)
- Rotate session secret periodically
- Use `Secure: true` when serving over HTTPS

**Alternatives Considered**:
- Database-backed sessions: Overkill for single-user app
- JWT: Stateless but harder to revoke, more complex validation

---

## 4. OAuth Authentication (bskyoauth)

### Decision: github.com/shindakun/bskyoauth

**Rationale**:
- Purpose-built for Bluesky OAuth 2.0
- Handles PKCE flow automatically
- Returns access token, refresh token, DID, and handle

**Best Practices**:
```go
import (
  "github.com/shindakun/bskyoauth"
)

var oauthClient *bskyoauth.Client

func InitOAuth() {
  oauthClient = bskyoauth.NewClient(
    "http://localhost:8080/auth/callback",
    []string{"atproto", "transition:generic"},
  )
}

// Start OAuth flow
func HandleOAuthLogin(w http.ResponseWriter, r *http.Request) {
  authURL, state, codeVerifier := oauthClient.GetAuthURL()

  // Store state and verifier in session
  session, _ := store.Get(r, "oauth")
  session.Values["state"] = state
  session.Values["code_verifier"] = codeVerifier
  session.Save(r, w)

  http.Redirect(w, r, authURL, http.StatusSeeOther)
}

// OAuth callback
func HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
  session, _ := store.Get(r, "oauth")
  state := session.Values["state"].(string)
  codeVerifier := session.Values["code_verifier"].(string)

  // Verify state
  if r.URL.Query().Get("state") != state {
    http.Error(w, "Invalid state", http.StatusBadRequest)
    return
  }

  // Exchange code for tokens
  code := r.URL.Query().Get("code")
  tokens, err := oauthClient.ExchangeCode(code, codeVerifier)
  if err != nil {
    http.Error(w, "OAuth exchange failed", http.StatusInternalServerError)
    return
  }

  // Save to auth session
  SaveSession(w, r, tokens.DID, tokens.Handle, tokens.DID, tokens.AccessToken)
  http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
```

**Error Handling**:
- User denies authorization: Show friendly message, redirect to landing
- Invalid tokens: Clear session, redirect to login
- Expired tokens: Implement refresh logic (future enhancement)

**Alternatives Considered**:
- Generic OAuth library: More complex, no Bluesky-specific helpers

---

## 5. AT Protocol Integration & Data Collection

### Decision: github.com/bluesky-social/indigo

**Rationale**:
- Official Go SDK for AT Protocol
- Type-safe generated code from Lexicon schemas
- XRPC client with automatic retries
- Active maintenance by Bluesky team

**Best Practices**:
```go
import (
  "github.com/bluesky-social/indigo/api/atproto"
  "github.com/bluesky-social/indigo/api/bsky"
  "github.com/bluesky-social/indigo/xrpc"
)

// Create authenticated client
func NewATProtoClient(accessToken, did, handle string) *xrpc.Client {
  client := &xrpc.Client{
    Host: "https://bsky.social",
    Auth: &xrpc.AuthInfo{
      AccessJwt:  accessToken,
      RefreshJwt: "", // Store refresh token separately if using
      Did:        did,
      Handle:     handle,
    },
  }
  return client
}

// Fetch user's posts (paginated)
func FetchPosts(ctx context.Context, client *xrpc.Client, did string, cursor string) (*bsky.FeedGetAuthorFeed_Output, error) {
  return bsky.FeedGetAuthorFeed(ctx, client, did, "", 100, cursor)
}

// Fetch profile
func FetchProfile(ctx context.Context, client *xrpc.Client, actor string) (*bsky.ActorGetProfile_Output, error) {
  return bsky.ActorGetProfile(ctx, client, actor)
}
```

**Data Collection Strategy**:

1. **Full Sync** (first archive):
   - Fetch all posts via `app.bsky.feed.getAuthorFeed`
   - Paginate with cursor (100 posts per page)
   - Download all media in parallel (max 5 concurrent)
   - Store posts, profiles, and media in SQLite

2. **Incremental Sync**:
   - Query most recent post timestamp from database
   - Fetch only posts newer than `lastSyncTime`
   - Update existing posts if edited (check CID)

3. **Rate Limiting**:
   - Bluesky limit: 300 requests per 5 minutes
   - Implement token bucket or sliding window
   - Exponential backoff on 429 responses
   - Track requests per operation in database

**Rate Limiter Example**:
```go
type RateLimiter struct {
  tokens    int
  maxTokens int
  refillRate time.Duration
  mu        sync.Mutex
}

func (rl *RateLimiter) Wait(ctx context.Context) error {
  rl.mu.Lock()
  defer rl.mu.Unlock()

  for rl.tokens == 0 {
    select {
    case <-ctx.Done():
      return ctx.Err()
    case <-time.After(rl.refillRate):
      rl.tokens = min(rl.tokens+1, rl.maxTokens)
    }
  }

  rl.tokens--
  return nil
}
```

**Media Download**:
```go
func DownloadMedia(ctx context.Context, mediaURL, localPath string) error {
  resp, err := http.Get(mediaURL)
  if err != nil {
    return err
  }
  defer resp.Body.Close()

  // Create directory structure (YYYY/MM)
  os.MkdirAll(filepath.Dir(localPath), 0755)

  // Write file
  out, err := os.Create(localPath)
  if err != nil {
    return err
  }
  defer out.Close()

  _, err = io.Copy(out, resp.Body)
  return err
}
```

**Background Worker**:
- Start goroutine for long-running archive operations
- Store operation status in database
- Provide progress updates via database polling
- Cancel operations with context cancellation

**Alternatives Considered**:
- Custom XRPC implementation: Too much work, no type safety
- REST API directly: Verbose, error-prone, no Lexicon types

---

## 6. SQLite Storage & Full-Text Search

### Decision: modernc.org/sqlite (pure Go) + FTS5

**Rationale**:
- Pure Go implementation (no CGO, easier cross-compilation)
- FTS5 virtual tables for full-text search
- Single-file database (aligns with local-first principle)
- Standard database/sql interface

**Database Schema**:

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
  embed_type TEXT,
  embed_data JSON,
  labels JSON,
  archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_posts_did ON posts(did);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_posts_has_media ON posts(has_media);

-- Full-text search
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

CREATE TRIGGER posts_ad AFTER DELETE ON posts BEGIN
  INSERT INTO posts_fts(posts_fts, rowid, uri, text) VALUES('delete', old.rowid, old.uri, old.text);
END;

-- Profiles table (snapshots over time)
CREATE TABLE profiles (
  did TEXT NOT NULL,
  handle TEXT NOT NULL,
  display_name TEXT,
  description TEXT,
  avatar_url TEXT,
  banner_url TEXT,
  followers_count INTEGER DEFAULT 0,
  follows_count INTEGER DEFAULT 0,
  posts_count INTEGER DEFAULT 0,
  snapshot_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (did, snapshot_at)
);

-- Media table
CREATE TABLE media (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  post_uri TEXT NOT NULL,
  media_url TEXT NOT NULL,
  local_path TEXT NOT NULL,
  alt_text TEXT,
  mime_type TEXT,
  size_bytes INTEGER,
  downloaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (post_uri) REFERENCES posts(uri) ON DELETE CASCADE
);

CREATE INDEX idx_media_post_uri ON media(post_uri);

-- Archive operations (track sync progress)
CREATE TABLE archive_operations (
  id TEXT PRIMARY KEY,
  did TEXT NOT NULL,
  operation_type TEXT NOT NULL, -- 'full_sync', 'incremental_sync'
  status TEXT NOT NULL, -- 'running', 'completed', 'failed'
  progress_current INTEGER DEFAULT 0,
  progress_total INTEGER DEFAULT 0,
  started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  completed_at TIMESTAMP,
  error_message TEXT
);

-- Sessions table (optional if using database-backed sessions)
CREATE TABLE sessions (
  id TEXT PRIMARY KEY,
  did TEXT NOT NULL,
  access_token TEXT NOT NULL,
  refresh_token TEXT,
  expires_at TIMESTAMP NOT NULL,
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**Best Practices**:

```go
import (
  "database/sql"
  "fmt"
  _ "modernc.org/sqlite"
)

// Initialize database with production-ready settings
func InitDB(path string) (*sql.DB, error) {
  // Connection string with production settings
  dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache=private&_temp_store=memory", path)

  db, err := sql.Open("sqlite", dsn)
  if err != nil {
    return nil, err
  }

  // Enable foreign keys
  if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
    return nil, err
  }

  // Run migrations
  if err := runMigrations(db); err != nil {
    return nil, err
  }

  return db, nil
}
```

**Production PRAGMA Settings Explained**:
- `_journal_mode=WAL`: Write-Ahead Logging for better concurrency and performance
- `_synchronous=NORMAL`: Balance between safety and performance (safe for WAL mode)
- `_busy_timeout=5000`: Wait up to 5 seconds if database is locked
- `_cache=private`: Use private page cache (better for single-user app)
- `_temp_store=memory`: Store temporary tables in memory for speed

**Additional Runtime PRAGMAs** (optional, applied after connection):
```go
// Optional: Set cache size (negative = KB, positive = pages)
db.Exec("PRAGMA cache_size = -64000") // 64MB cache

// Optional: Enable memory-mapped I/O for read performance
db.Exec("PRAGMA mmap_size = 268435456") // 256MB

// Optional: Set connection pool limits
db.SetMaxOpenConns(1) // Single writer for SQLite
db.SetMaxIdleConns(1)
db.SetConnMaxLifetime(0)

// Full-text search
func SearchPosts(db *sql.DB, query string, limit, offset int) ([]Post, error) {
  rows, err := db.Query(`
    SELECT p.uri, p.text, p.created_at, p.like_count
    FROM posts p
    JOIN posts_fts fts ON p.rowid = fts.rowid
    WHERE posts_fts MATCH ?
    ORDER BY p.created_at DESC
    LIMIT ? OFFSET ?
  `, query, limit, offset)
  if err != nil {
    return nil, err
  }
  defer rows.Close()

  var posts []Post
  for rows.Next() {
    var p Post
    if err := rows.Scan(&p.URI, &p.Text, &p.CreatedAt, &p.LikeCount); err != nil {
      return nil, err
    }
    posts = append(posts, p)
  }

  return posts, nil
}
```

**Migration Strategy**:
- Store SQL files in `internal/storage/migrations/`
- Name files: `001_initial.sql`, `002_fts.sql`, etc.
- Track applied migrations in `schema_migrations` table
- Apply migrations on startup

**Alternatives Considered**:
- PostgreSQL: Overkill for single-user local app
- SQLite with CGO: Complicates cross-compilation
- File-based storage: No search, harder queries

---

## 7. CSS Framework & Styling

### Decision: Pico CSS (classless) + custom dark theme

**Rationale**:
- Classless CSS means minimal HTML changes
- Built-in dark theme support
- Responsive by default
- Small footprint (~10KB minified)
- Semantic HTML automatically styled

**Best Practices**:

```html
<!-- No classes needed for basic styling -->
<nav>
  <ul>
    <li><strong>Bluesky Archive</strong></li>
  </ul>
  <ul>
    <li><a href="/dashboard">Dashboard</a></li>
    <li><a href="/archive">Archive</a></li>
    <li><a href="/about">About</a></li>
  </ul>
</nav>

<main class="container">
  <article>
    <header>
      <h1>Archive Status</h1>
    </header>
    <p>Last sync: <time>2025-10-30</time></p>
    <progress value="750" max="1000"></progress>
  </article>
</main>
```

**Custom CSS** (custom.css):
```css
/* Override Pico variables */
:root[data-theme="dark"] {
  --primary: #1DA1F2; /* Bluesky blue */
  --primary-hover: #1A8CD8;
  --card-background-color: #1A1A1A;
}

/* Archive-specific styles */
.post-card {
  border-left: 3px solid var(--primary);
  padding-left: 1rem;
  margin-bottom: 1rem;
}

.progress-container {
  position: relative;
}

.progress-label {
  position: absolute;
  top: 50%;
  left: 50%;
  transform: translate(-50%, -50%);
  font-weight: bold;
}
```

**Responsive Design**:
- Pico handles breakpoints automatically
- Use `<picture>` for responsive images
- Test on mobile (iPhone, Android), tablet (iPad), desktop

**Alternatives Considered**:
- Tailwind CSS: Requires build step, verbose classes
- Bootstrap: Heavy, opinionated, not classless
- Custom CSS from scratch: Too much work, reinventing the wheel

---

## 8. HTMX Integration

### Decision: HTMX for dynamic interactions

**Rationale**:
- Avoid full JavaScript framework (React, Vue)
- Declarative HTML attributes for AJAX
- Server-side rendering (Go templates)
- Progressive enhancement (works without JS)

**Best Practices**:

```html
<!-- Polling for archive progress -->
<div
  hx-get="/archive/status"
  hx-trigger="every 2s"
  hx-target="#archive-status"
  hx-swap="innerHTML">
  <div id="archive-status">
    <progress value="0" max="100"></progress>
    <p>Starting archive...</p>
  </div>
</div>

<!-- Start archive with POST -->
<button
  hx-post="/archive/start"
  hx-target="#archive-status"
  hx-swap="innerHTML">
  Start Archive
</button>

<!-- Load more posts (infinite scroll) -->
<div
  hx-get="/browse?page=2"
  hx-trigger="revealed"
  hx-swap="afterend">
  <!-- Posts will be appended here -->
</div>
```

**Server-side handler** (returns HTML fragment):
```go
func HandleArchiveStatus(w http.ResponseWriter, r *http.Request) {
  session, _ := store.Get(r, "auth")
  did := session.Values["did"].(string)

  op, err := storage.GetActiveOperation(r.Context(), did)
  if err != nil {
    http.Error(w, "No active operation", http.StatusNotFound)
    return
  }

  // Return HTML fragment
  data := struct {
    Progress int
    Total    int
    Status   string
  }{
    Progress: op.ProgressCurrent,
    Total:    op.ProgressTotal,
    Status:   op.Status,
  }

  templates.ExecuteTemplate(w, "archive-status-partial", data)
}
```

**Vanilla JS** (minimal):
```javascript
// app.js - only for non-HTMX interactions
document.addEventListener('DOMContentLoaded', () => {
  // Add confirmation for destructive actions
  document.querySelectorAll('[data-confirm]').forEach(el => {
    el.addEventListener('click', (e) => {
      if (!confirm(e.target.dataset.confirm)) {
        e.preventDefault();
      }
    });
  });
});
```

**Alternatives Considered**:
- Full SPA (React): Overkill, requires API, more complexity
- Alpine.js: Adds dependency, HTMX sufficient for this use case

---

## 9. Background Worker & Progress Tracking

### Decision: Goroutine with database-backed progress

**Rationale**:
- Goroutines are lightweight, built-in concurrency
- Database stores operation state (survives crashes)
- Web interface polls database for progress
- Context cancellation for graceful shutdown

**Best Practices**:

```go
// Start archive operation
func StartArchive(ctx context.Context, db *sql.DB, client *xrpc.Client, did string, operationType string) (string, error) {
  // Check for active operation
  existing, _ := storage.GetActiveOperation(ctx, did)
  if existing != nil {
    return "", errors.New("archive already in progress")
  }

  // Create operation record
  opID := uuid.New().String()
  op := &ArchiveOperation{
    ID:            opID,
    DID:           did,
    OperationType: operationType,
    Status:        "running",
    StartedAt:     time.Now(),
  }

  if err := storage.CreateOperation(ctx, db, op); err != nil {
    return "", err
  }

  // Start worker in background
  go archiveWorker(context.Background(), db, client, op)

  return opID, nil
}

// Worker function
func archiveWorker(ctx context.Context, db *sql.DB, client *xrpc.Client, op *ArchiveOperation) {
  defer func() {
    if r := recover(); r != nil {
      op.Status = "failed"
      op.ErrorMessage = fmt.Sprintf("panic: %v", r)
      storage.UpdateOperation(ctx, db, op)
    }
  }()

  var cursor string
  totalFetched := 0

  for {
    select {
    case <-ctx.Done():
      op.Status = "cancelled"
      storage.UpdateOperation(ctx, db, op)
      return
    default:
    }

    // Fetch posts
    resp, err := bsky.FeedGetAuthorFeed(ctx, client, op.DID, "", 100, cursor)
    if err != nil {
      op.Status = "failed"
      op.ErrorMessage = err.Error()
      storage.UpdateOperation(ctx, db, op)
      return
    }

    // Save posts
    for _, feedItem := range resp.Feed {
      post := mapFeedItemToPost(feedItem)
      storage.SavePost(ctx, db, post)
      totalFetched++

      // Update progress
      op.ProgressCurrent = totalFetched
      storage.UpdateOperation(ctx, db, op)
    }

    // Check for more pages
    if resp.Cursor == nil || *resp.Cursor == "" {
      break
    }
    cursor = *resp.Cursor
  }

  // Mark complete
  op.Status = "completed"
  op.CompletedAt = &time.Now()
  op.ProgressTotal = totalFetched
  storage.UpdateOperation(ctx, db, op)
}
```

**Error Handling**:
- Network errors: Retry with exponential backoff
- Rate limit errors: Sleep and retry
- Database errors: Log and fail operation
- Panic recovery: Mark operation as failed

**Alternatives Considered**:
- Job queue (e.g., Asynq): Overkill for single-user app
- Polling AT Protocol: Push not available, polling is standard

---

## 10. Configuration Management

### Decision: YAML config file + environment variables

**Rationale**:
- YAML is human-readable
- Environment variables for secrets
- Override config with env vars for deployment flexibility

**Configuration Structure** (config.yaml):
```yaml
server:
  host: "localhost"
  port: 8080
  session_secret: "${SESSION_SECRET}" # From env var

archive:
  data_dir: "./archive"
  db_path: "./archive/db/archive.db"
  media_dir: "./archive/media"

oauth:
  callback_url: "http://localhost:8080/auth/callback"
  scopes:
    - "atproto"
    - "transition:generic"

rate_limit:
  max_requests: 300
  window_seconds: 300
```

**Loading Config**:
```go
import (
  "gopkg.in/yaml.v3"
  "os"
)

type Config struct {
  Server struct {
    Host          string `yaml:"host"`
    Port          int    `yaml:"port"`
    SessionSecret string `yaml:"session_secret"`
  } `yaml:"server"`

  Archive struct {
    DataDir  string `yaml:"data_dir"`
    DBPath   string `yaml:"db_path"`
    MediaDir string `yaml:"media_dir"`
  } `yaml:"archive"`
}

func LoadConfig(path string) (*Config, error) {
  data, err := os.ReadFile(path)
  if err != nil {
    return nil, err
  }

  // Expand environment variables
  expanded := os.ExpandEnv(string(data))

  var cfg Config
  if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
    return nil, err
  }

  return &cfg, nil
}
```

**Alternatives Considered**:
- JSON: Less readable, no comments
- TOML: Less common in Go ecosystem
- Environment variables only: Hard to manage many settings

---

## 11. Testing Strategy

### Decision: Table-driven unit tests + integration tests + contract tests

**Unit Tests**:
```go
// internal/storage/posts_test.go
func TestSavePost(t *testing.T) {
  tests := []struct {
    name    string
    post    *Post
    wantErr bool
  }{
    {
      name: "valid post",
      post: &Post{
        URI:       "at://did:plc:abc123/app.bsky.feed.post/xyz789",
        CID:       "bafyreiabc123",
        DID:       "did:plc:abc123",
        Text:      "Hello world",
        CreatedAt: time.Now(),
      },
      wantErr: false,
    },
    // More test cases...
  }

  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      db := setupTestDB(t)
      defer db.Close()

      err := SavePost(context.Background(), db, tt.post)
      if (err != nil) != tt.wantErr {
        t.Errorf("SavePost() error = %v, wantErr %v", err, tt.wantErr)
      }
    })
  }
}
```

**Integration Tests** (with AT Protocol):
```go
// tests/integration/archiver_test.go
func TestFetchPosts(t *testing.T) {
  if testing.Short() {
    t.Skip("skipping integration test")
  }

  client := NewTestClient(t) // Use test account
  posts, err := FetchPosts(context.Background(), client, testDID, "")

  if err != nil {
    t.Fatalf("FetchPosts() error = %v", err)
  }

  if len(posts) == 0 {
    t.Error("expected posts, got 0")
  }
}
```

**Contract Tests** (HTTP API):
```go
// tests/contract/handlers_test.go
func TestDashboardHandler(t *testing.T) {
  app := setupTestApp(t)

  req := httptest.NewRequest("GET", "/dashboard", nil)
  req = addAuthSession(t, req) // Helper to add authenticated session
  w := httptest.NewRecorder()

  app.ServeHTTP(w, req)

  if w.Code != http.StatusOK {
    t.Errorf("expected status 200, got %d", w.Code)
  }

  body := w.Body.String()
  if !strings.Contains(body, "Archive Status") {
    t.Error("expected dashboard to contain 'Archive Status'")
  }
}
```

**Running Tests**:
```bash
# Unit tests only
go test ./... -short

# All tests including integration
go test ./...

# With coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

---

## 12. Deployment & Operations

### Decision: Single binary with embedded assets

**Rationale**:
- Simplifies deployment (one file)
- No external dependencies at runtime
- Embed templates, CSS, JS using `go:embed`

**Embedding Assets**:
```go
//go:embed internal/web/templates/* internal/web/static/*
var embeddedFS embed.FS

func init() {
  // Load templates from embedded FS
  templates = template.Must(template.ParseFS(embeddedFS, "internal/web/templates/**/*.html"))
}

// Serve static files from embedded FS
func StaticHandler() http.Handler {
  sub, _ := fs.Sub(embeddedFS, "internal/web/static")
  return http.FileServer(http.FS(sub))
}
```

**Building**:
```bash
# Build for current platform
go build -o bskyarchive cmd/bskyarchive/main.go

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o bskyarchive-linux cmd/bskyarchive/main.go

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -o bskyarchive.exe cmd/bskyarchive/main.go
```

**Running**:
```bash
# Generate session secret
export SESSION_SECRET=$(openssl rand -hex 32)

# Run application
./bskyarchive

# Or with custom config
./bskyarchive --config /path/to/config.yaml
```

**Systemd Service** (optional for Linux):
```ini
[Unit]
Description=Bluesky Archive Tool
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/home/youruser/bskyarchive
ExecStart=/home/youruser/bskyarchive/bskyarchive
Restart=on-failure
Environment="SESSION_SECRET=your_secret_here"

[Install]
WantedBy=multi-user.target
```

**Alternatives Considered**:
- Docker: Overkill for local app, adds complexity
- External asset serving: More complex deployment

---

## Summary

This research document resolves all unknowns from the Technical Context and establishes concrete implementation patterns for:

1. **Web Layer**: chi router + html/template + gorilla/sessions
2. **Authentication**: bskyoauth + encrypted cookie sessions
3. **Data Collection**: indigo SDK + rate limiting + background workers
4. **Storage**: modernc.org/sqlite + FTS5 full-text search
5. **Frontend**: Pico CSS + HTMX + vanilla JavaScript
6. **Deployment**: Single binary with embedded assets

All decisions align with the constitution's core principles (local-first, privacy, efficiency) and support the feature requirements from spec.md. The next phase will generate concrete data models and API contracts based on these decisions.
