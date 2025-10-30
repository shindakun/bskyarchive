# Quickstart Guide: Web Interface Implementation

**Phase 1 Output** | **Date**: 2025-10-30 | **Plan**: [plan.md](./plan.md)

## Overview

This guide provides step-by-step instructions for implementing the Bluesky Personal Archive Tool web interface. It covers environment setup, phased implementation, testing strategies, and deployment.

---

## Prerequisites

### Required Tools

- **Go 1.21+**: [Download](https://go.dev/dl/)
- **Git**: For version control
- **Text Editor/IDE**: VS Code, GoLand, or similar
- **Web Browser**: For testing (Chrome, Firefox, Safari, or Edge)

### Recommended Tools

- **Air**: Hot reload during development (`go install github.com/cosmtrek/air@latest`)
- **golangci-lint**: Code linting (`go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest`)
- **go-migrate**: Database migrations (optional, can use custom implementation)

---

## Phase 1: Project Setup & Dependencies

### Initialize Go Module

```bash
# From repository root
cd /Users/steve/go/src/github.com/shindakun/bskyarchive

# Initialize go.mod if not already done
go mod init github.com/shindakun/bskyarchive

# Tidy dependencies
go mod tidy
```

### Install Dependencies

```bash
# Web framework and routing
go get github.com/go-chi/chi/v5

# Session management
go get github.com/gorilla/sessions

# CSRF protection
go get github.com/gorilla/csrf

# OAuth authentication
go get github.com/shindakun/bskyoauth

# AT Protocol SDK
go get github.com/bluesky-social/indigo

# SQLite (pure Go)
go get modernc.org/sqlite

# YAML configuration
go get gopkg.in/yaml.v3

# UUID generation
go get github.com/google/uuid
```

### Directory Structure

Create the directory structure as defined in [plan.md](./plan.md):

```bash
# Create main application directories
mkdir -p cmd/bskyarchive
mkdir -p internal/web/{handlers,middleware,templates,static}
mkdir -p internal/web/templates/{layouts,pages,partials}
mkdir -p internal/web/static/{css,js,images}
mkdir -p internal/auth
mkdir -p internal/archiver
mkdir -p internal/storage/migrations
mkdir -p internal/models

# Create test directories
mkdir -p tests/{integration,contract,unit}

# Create runtime data directories (will be created automatically, but good for dev)
mkdir -p archive/{media,db}
```

### Download Static Assets

```bash
# Download Pico CSS
curl -o internal/web/static/css/pico.min.css \
  https://cdn.jsdelivr.net/npm/@picocss/pico@1/css/pico.min.css

# Download HTMX
curl -o internal/web/static/js/htmx.min.js \
  https://unpkg.com/htmx.org@1.9.10/dist/htmx.min.js
```

---

## Phase 2: Core Implementation

### Step 1: Configuration Management

Create `internal/config/config.go`:

```go
package config

import (
    "os"
    "gopkg.in/yaml.v3"
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

    OAuth struct {
        CallbackURL string   `yaml:"callback_url"`
        Scopes      []string `yaml:"scopes"`
    } `yaml:"oauth"`
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

Create `config.yaml`:

```yaml
server:
  host: "localhost"
  port: 8080
  session_secret: "${SESSION_SECRET}"

archive:
  data_dir: "./archive"
  db_path: "./archive/db/archive.db"
  media_dir: "./archive/media"

oauth:
  callback_url: "http://localhost:8080/auth/callback"
  scopes:
    - "atproto"
    - "transition:generic"
```

### Step 2: Data Models

Create models in `internal/models/` based on [data-model.md](./data-model.md):

- `post.go`: Post struct and methods
- `profile.go`: Profile struct and methods
- `media.go`: Media struct and methods
- `session.go`: Session struct and methods
- `operation.go`: ArchiveOperation struct and methods

**Example** (`internal/models/post.go`):

```go
package models

import (
    "encoding/json"
    "time"
)

type Post struct {
    URI         string          `json:"uri" db:"uri"`
    CID         string          `json:"cid" db:"cid"`
    DID         string          `json:"did" db:"did"`
    Text        string          `json:"text" db:"text"`
    CreatedAt   time.Time       `json:"created_at" db:"created_at"`
    IndexedAt   time.Time       `json:"indexed_at" db:"indexed_at"`
    HasMedia    bool            `json:"has_media" db:"has_media"`
    LikeCount   int             `json:"like_count" db:"like_count"`
    RepostCount int             `json:"repost_count" db:"repost_count"`
    ReplyCount  int             `json:"reply_count" db:"reply_count"`
    IsReply     bool            `json:"is_reply" db:"is_reply"`
    ReplyParent string          `json:"reply_parent,omitempty" db:"reply_parent"`
    EmbedType   string          `json:"embed_type,omitempty" db:"embed_type"`
    EmbedData   json.RawMessage `json:"embed_data,omitempty" db:"embed_data"`
    Labels      json.RawMessage `json:"labels,omitempty" db:"labels"`
    ArchivedAt  time.Time       `json:"archived_at" db:"archived_at"`
}
```

### Step 3: Database Layer

Create `internal/storage/db.go`:

```go
package storage

import (
    "database/sql"
    "fmt"
    "os"
    "path/filepath"

    _ "modernc.org/sqlite"
)

func InitDB(path string) (*sql.DB, error) {
    // Ensure directory exists
    if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
        return nil, err
    }

    // Connection string with production-ready settings
    dsn := fmt.Sprintf("%s?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache=private&_temp_store=memory", path)

    // Open database
    db, err := sql.Open("sqlite", dsn)
    if err != nil {
        return nil, err
    }

    // Enable foreign keys
    if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
        return nil, err
    }

    // Optional: Set cache size and memory-mapped I/O
    db.Exec("PRAGMA cache_size = -64000")     // 64MB cache
    db.Exec("PRAGMA mmap_size = 268435456")   // 256MB mmap

    // Set connection pool limits (SQLite needs single writer)
    db.SetMaxOpenConns(1)
    db.SetMaxIdleConns(1)
    db.SetConnMaxLifetime(0)

    // Run migrations
    if err := runMigrations(db); err != nil {
        return nil, err
    }

    return db, nil
}

func runMigrations(db *sql.DB) error {
    // TODO: Implement migration logic
    // For now, just run all .sql files in migrations/
    return nil
}
```

Create migration files in `internal/storage/migrations/`:

- `001_initial.sql`: Posts, profiles, media tables
- `002_fts.sql`: Full-text search virtual tables
- `003_operations.sql`: Archive operations table

**Example** (`internal/storage/migrations/001_initial.sql`):

```sql
CREATE TABLE IF NOT EXISTS posts (
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

CREATE INDEX IF NOT EXISTS idx_posts_did ON posts(did);
CREATE INDEX IF NOT EXISTS idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_posts_has_media ON posts(has_media);

-- Add similar CREATE TABLE statements for profiles, media, operations
```

### Step 4: Session Management

Create `internal/auth/session.go`:

```go
package auth

import (
    "net/http"
    "github.com/gorilla/sessions"
)

var store *sessions.CookieStore

func InitSessions(secret []byte) {
    store = sessions.NewCookieStore(secret)
    store.Options = &sessions.Options{
        Path:     "/",
        MaxAge:   7 * 24 * 60 * 60, // 7 days
        HttpOnly: true,
        Secure:   false, // Set to true in production with HTTPS
        SameSite: http.SameSiteLaxMode,
    }
}

func SaveSession(w http.ResponseWriter, r *http.Request, did, handle, accessToken string) error {
    session, _ := store.Get(r, "auth")
    session.Values["did"] = did
    session.Values["handle"] = handle
    session.Values["access_token"] = accessToken
    session.Values["authenticated"] = true
    return session.Save(r, w)
}

func GetSession(r *http.Request) (*Session, error) {
    session, _ := store.Get(r, "auth")
    if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
        return nil, ErrNotAuthenticated
    }

    return &Session{
        DID:         session.Values["did"].(string),
        Handle:      session.Values["handle"].(string),
        AccessToken: session.Values["access_token"].(string),
    }, nil
}

type Session struct {
    DID         string
    Handle      string
    AccessToken string
}

var ErrNotAuthenticated = errors.New("not authenticated")
```

### Step 5: OAuth Integration

Create `internal/auth/oauth.go`:

```go
package auth

import (
    "net/http"
    "github.com/shindakun/bskyoauth"
)

var oauthClient *bskyoauth.Client

func InitOAuth(callbackURL string, scopes []string) {
    oauthClient = bskyoauth.NewClient(callbackURL, scopes)
}

func HandleOAuthLogin(w http.ResponseWriter, r *http.Request) {
    authURL, state, codeVerifier := oauthClient.GetAuthURL()

    // Store state and verifier in temporary session
    session, _ := store.Get(r, "oauth")
    session.Values["state"] = state
    session.Values["code_verifier"] = codeVerifier
    session.Save(r, w)

    http.Redirect(w, r, authURL, http.StatusSeeOther)
}

func HandleOAuthCallback(w http.ResponseWriter, r *http.Request) {
    session, _ := store.Get(r, "oauth")
    state := session.Values["state"].(string)
    codeVerifier := session.Values["code_verifier"].(string)

    // Verify state
    if r.URL.Query().Get("state") != state {
        http.Redirect(w, r, "/?error=oauth_failed", http.StatusSeeOther)
        return
    }

    // Exchange code for tokens
    code := r.URL.Query().Get("code")
    tokens, err := oauthClient.ExchangeCode(code, codeVerifier)
    if err != nil {
        http.Redirect(w, r, "/?error=oauth_failed", http.StatusSeeOther)
        return
    }

    // Save authenticated session
    SaveSession(w, r, tokens.DID, tokens.Handle, tokens.AccessToken)
    http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}
```

### Step 6: HTTP Handlers

Create handlers in `internal/web/handlers/` based on [contracts/http-api.md](./contracts/http-api.md):

- `landing.go`: GET /
- `about.go`: GET /about
- `dashboard.go`: GET /dashboard
- `archive.go`: GET /archive, POST /archive/start, GET /archive/status
- `browse.go`: GET /browse

**Example** (`internal/web/handlers/landing.go`):

```go
package handlers

import (
    "net/http"
    "github.com/shindakun/bskyarchive/internal/auth"
)

func Landing(w http.ResponseWriter, r *http.Request) {
    // If already authenticated, redirect to dashboard
    if _, err := auth.GetSession(r); err == nil {
        http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
        return
    }

    // Render landing page
    data := LandingData{
        Error:   r.URL.Query().Get("error"),
        Message: r.URL.Query().Get("message"),
    }

    RenderTemplate(w, "landing", data)
}

type LandingData struct {
    Error   string
    Message string
}
```

### Step 7: Middleware

Create `internal/web/middleware/auth.go`:

```go
package middleware

import (
    "net/http"
    "github.com/shindakun/bskyarchive/internal/auth"
)

func RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if _, err := auth.GetSession(r); err != nil {
            http.Redirect(w, r, "/?error=auth_required", http.StatusSeeOther)
            return
        }
        next.ServeHTTP(w, r)
    })
}
```

### Step 8: Router Setup

Create `internal/web/router.go`:

```go
package web

import (
    "net/http"
    "github.com/go-chi/chi/v5"
    "github.com/go-chi/chi/v5/middleware"

    "github.com/shindakun/bskyarchive/internal/auth"
    "github.com/shindakun/bskyarchive/internal/web/handlers"
    mw "github.com/shindakun/bskyarchive/internal/web/middleware"
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
    r.Get("/auth/login", auth.HandleOAuthLogin)
    r.Get("/auth/callback", auth.HandleOAuthCallback)

    // Protected routes
    r.Group(func(r chi.Router) {
        r.Use(mw.RequireAuth)
        r.Get("/dashboard", handlers.Dashboard)
        r.Get("/archive", handlers.Archive)
        r.Post("/archive/start", handlers.StartArchive)
        r.Get("/archive/status", handlers.ArchiveStatus)
        r.Get("/browse", handlers.Browse)
        r.Get("/auth/logout", auth.HandleLogout)
    })

    // Static assets
    r.Handle("/static/*", http.StripPrefix("/static/",
        http.FileServer(http.Dir("internal/web/static"))))

    return r
}
```

### Step 9: Main Application

Create `cmd/bskyarchive/main.go`:

```go
package main

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "time"

    "github.com/shindakun/bskyarchive/internal/auth"
    "github.com/shindakun/bskyarchive/internal/config"
    "github.com/shindakun/bskyarchive/internal/storage"
    "github.com/shindakun/bskyarchive/internal/web"
)

func main() {
    // Load configuration
    cfg, err := config.LoadConfig("config.yaml")
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    // Generate session secret if not set
    if cfg.Server.SessionSecret == "" {
        secret := make([]byte, 32)
        rand.Read(secret)
        cfg.Server.SessionSecret = hex.EncodeToString(secret)
        log.Println("Generated session secret. Set SESSION_SECRET env var for production.")
    }

    // Initialize sessions
    auth.InitSessions([]byte(cfg.Server.SessionSecret))

    // Initialize OAuth
    auth.InitOAuth(cfg.OAuth.CallbackURL, cfg.OAuth.Scopes)

    // Initialize database
    db, err := storage.InitDB(cfg.Archive.DBPath)
    if err != nil {
        log.Fatalf("Failed to initialize database: %v", err)
    }
    defer db.Close()

    // Create router
    router := web.NewRouter()

    // Start server
    addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
    srv := &http.Server{
        Addr:         addr,
        Handler:      router,
        ReadTimeout:  10 * time.Second,
        WriteTimeout: 10 * time.Second,
        IdleTimeout:  120 * time.Second,
    }

    // Graceful shutdown
    go func() {
        log.Printf("Starting server on %s", addr)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("Server failed: %v", err)
        }
    }()

    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, os.Interrupt)
    <-quit

    log.Println("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatalf("Server forced to shutdown: %v", err)
    }

    log.Println("Server exited")
}
```

---

## Phase 3: Templates & Frontend

### Create Base Template

Create `internal/web/templates/layouts/base.html`:

```html
<!DOCTYPE html>
<html lang="en" data-theme="dark">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>{{block "title" .}}Bluesky Archive Tool{{end}}</title>
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

### Create Page Templates

Create templates in `internal/web/templates/pages/`:

- `landing.html`
- `dashboard.html`
- `archive.html`
- `browse.html`
- `about.html`

**Example** (`internal/web/templates/pages/landing.html`):

```html
{{define "title"}}Welcome - Bluesky Archive{{end}}

{{define "content"}}
<article>
  <header>
    <h1>Bluesky Personal Archive Tool</h1>
  </header>
  <p>Archive your Bluesky account locally with full control over your data.</p>

  {{if .Error}}
    <p style="color: red;">Error: {{.Error}}</p>
  {{end}}

  <a href="/auth/login" role="button">Login with Bluesky</a>

  <footer>
    <small><a href="/about">Learn more</a></small>
  </footer>
</article>
{{end}}
```

### Custom Styles

Create `internal/web/static/css/custom.css`:

```css
:root[data-theme="dark"] {
  --primary: #1DA1F2;
  --primary-hover: #1A8CD8;
  --card-background-color: #1A1A1A;
}

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

### Minimal JavaScript

Create `internal/web/static/js/app.js`:

```javascript
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

---

## Phase 4: Testing

### Unit Tests

Create `internal/storage/posts_test.go`:

```go
package storage_test

import (
    "context"
    "testing"
    "time"

    "github.com/shindakun/bskyarchive/internal/models"
    "github.com/shindakun/bskyarchive/internal/storage"
)

func TestSavePost(t *testing.T) {
    db := setupTestDB(t)
    defer db.Close()

    post := &models.Post{
        URI:       "at://did:plc:abc123/app.bsky.feed.post/xyz789",
        CID:       "bafyreiabc123",
        DID:       "did:plc:abc123",
        Text:      "Hello world",
        CreatedAt: time.Now(),
        IndexedAt: time.Now(),
    }

    err := storage.SavePost(context.Background(), db, post)
    if err != nil {
        t.Fatalf("SavePost() error = %v", err)
    }

    // Verify post was saved
    retrieved, err := storage.GetPost(context.Background(), db, post.URI)
    if err != nil {
        t.Fatalf("GetPost() error = %v", err)
    }

    if retrieved.Text != post.Text {
        t.Errorf("expected text %q, got %q", post.Text, retrieved.Text)
    }
}

func setupTestDB(t *testing.T) *sql.DB {
    // Create in-memory database for testing
    db, err := storage.InitDB(":memory:")
    if err != nil {
        t.Fatalf("Failed to setup test DB: %v", err)
    }
    return db
}
```

### Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out

# Run only short tests (skip integration tests)
go test ./... -short
```

---

## Phase 5: Running & Development

### Generate Session Secret

```bash
export SESSION_SECRET=$(openssl rand -hex 32)
```

### Run Application

```bash
# Build and run
go build -o bskyarchive cmd/bskyarchive/main.go
./bskyarchive

# Or run directly
go run cmd/bskyarchive/main.go
```

### Hot Reload (Development)

Install Air:

```bash
go install github.com/cosmtrek/air@latest
```

Create `.air.toml`:

```toml
[build]
  cmd = "go build -o ./tmp/main cmd/bskyarchive/main.go"
  bin = "tmp/main"
  include_ext = ["go", "html", "css", "js"]
  exclude_dir = ["tmp", "vendor"]
```

Run with hot reload:

```bash
air
```

---

## Phase 6: Deployment

### Build for Production

```bash
# Build for current platform
go build -ldflags="-s -w" -o bskyarchive cmd/bskyarchive/main.go

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bskyarchive-linux cmd/bskyarchive/main.go

# Cross-compile for Windows
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o bskyarchive.exe cmd/bskyarchive/main.go
```

### Environment Variables

Required for production:

```bash
export SESSION_SECRET="your-32-byte-hex-secret"
```

### Run in Production

```bash
./bskyarchive
```

---

## Troubleshooting

### Common Issues

1. **Port already in use**:
   ```bash
   # Change port in config.yaml or kill existing process
   lsof -ti:8080 | xargs kill
   ```

2. **Session secret not set**:
   ```bash
   export SESSION_SECRET=$(openssl rand -hex 32)
   ```

3. **Database locked**:
   - Ensure only one instance is running
   - Delete `archive/db/archive.db-wal` and `archive/db/archive.db-shm`

4. **Templates not found**:
   - Verify template paths in handlers
   - Ensure templates are embedded in production builds

---

## Next Steps

After implementing the web interface:

1. **Add Archiver Logic**: Implement AT Protocol data collection in `internal/archiver/`
2. **Add Background Worker**: Implement async operations in `internal/archiver/worker.go`
3. **Add Search**: Implement full-text search in `internal/storage/search.go`
4. **Add Export**: Implement JSON/Markdown/HTML export (future feature)
5. **Add CLI**: Create CLI commands for non-web operations (future feature)

---

## Resources

- [Go Documentation](https://go.dev/doc/)
- [Chi Router](https://github.com/go-chi/chi)
- [Gorilla Sessions](https://github.com/gorilla/sessions)
- [AT Protocol](https://atproto.com/docs)
- [Indigo SDK](https://github.com/bluesky-social/indigo)
- [Pico CSS](https://picocss.com/)
- [HTMX](https://htmx.org/)
