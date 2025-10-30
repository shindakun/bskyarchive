package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/shindakun/bskyarchive/internal/web/middleware"
)

// Handlers holds dependencies for HTTP handlers
type Handlers struct {
	db           *sql.DB
	sessionStore *middleware.SessionStore
	logger       *log.Logger
}

// New creates a new Handlers instance
func New(db *sql.DB, sessionStore *middleware.SessionStore, logger *log.Logger) *Handlers {
	return &Handlers{
		db:           db,
		sessionStore: sessionStore,
		logger:       logger,
	}
}

// Landing renders the landing page
func (h *Handlers) Landing(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Bluesky Personal Archive Tool</title>
	<link rel="stylesheet" href="/static/css/pico.min.css">
</head>
<body>
	<main class="container">
		<h1>Bluesky Personal Archive Tool</h1>
		<p>Archive and search your Bluesky posts locally.</p>
		<a href="/auth/login" role="button">Sign in with Bluesky</a>
	</main>
</body>
</html>`))
}

// About renders the about page
func (h *Handlers) About(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>About - Bluesky Personal Archive Tool</title>
	<link rel="stylesheet" href="/static/css/pico.min.css">
</head>
<body>
	<main class="container">
		<h1>About</h1>
		<p>Bluesky Personal Archive Tool - A local-first archival solution.</p>
		<a href="/">Back to Home</a>
	</main>
</body>
</html>`))
}

// Login initiates OAuth flow
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	// OAuth implementation in Phase 3
	http.Error(w, "OAuth not yet implemented", http.StatusNotImplemented)
}

// Callback handles OAuth callback
func (h *Handlers) Callback(w http.ResponseWriter, r *http.Request) {
	// OAuth implementation in Phase 3
	http.Error(w, "OAuth not yet implemented", http.StatusNotImplemented)
}

// Logout clears the session
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	// Session clearing implementation in Phase 3
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Dashboard renders the user dashboard
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`<!DOCTYPE html>
<html>
<head>
	<title>Dashboard - Bluesky Personal Archive Tool</title>
	<link rel="stylesheet" href="/static/css/pico.min.css">
</head>
<body>
	<main class="container">
		<h1>Dashboard</h1>
		<p>Welcome to your dashboard.</p>
	</main>
</body>
</html>`))
}

// Archive renders the archive management page
func (h *Handlers) Archive(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// ArchiveStart initiates an archive operation
func (h *Handlers) ArchiveStart(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// ArchiveStatus returns the status of an archive operation
func (h *Handlers) ArchiveStatus(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// Browse renders the post browsing page
func (h *Handlers) Browse(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// ServeMedia serves archived media files
func (h *Handlers) ServeMedia(w http.ResponseWriter, r *http.Request) {
	http.Error(w, "Not implemented", http.StatusNotImplemented)
}

// ServeStatic serves static files
func (h *Handlers) ServeStatic(w http.ResponseWriter, r *http.Request) {
	// Remove /static prefix
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	if path == "" || path == "/" {
		http.NotFound(w, r)
		return
	}

	// Serve from internal/web/static
	fullPath := filepath.Join("internal", "web", "static", filepath.Clean(path))
	http.ServeFile(w, r, fullPath)
}
