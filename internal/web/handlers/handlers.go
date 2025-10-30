package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/shindakun/bskyarchive/internal/auth"
)

// Handlers holds dependencies for HTTP handlers
type Handlers struct {
	db             *sql.DB
	sessionManager *auth.SessionManager
	oauthManager   *auth.OAuthManager
	logger         *log.Logger
}

// New creates a new Handlers instance
func New(db *sql.DB, sessionManager *auth.SessionManager, oauthManager *auth.OAuthManager, logger *log.Logger) *Handlers {
	return &Handlers{
		db:             db,
		sessionManager: sessionManager,
		oauthManager:   oauthManager,
		logger:         logger,
	}
}

// Landing renders the landing page (check auth, redirect if authenticated)
func (h *Handlers) Landing(w http.ResponseWriter, r *http.Request) {
	// Check if user is already authenticated
	session, err := h.sessionManager.GetSession(r)
	if err == nil && session != nil && session.IsActive() {
		// Already logged in, redirect to dashboard
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}

	// Render landing page
	data := TemplateData{}
	if err := h.renderTemplate(w, "landing", data); err != nil {
		h.logger.Printf("Error rendering landing template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// About renders the about page
func (h *Handlers) About(w http.ResponseWriter, r *http.Request) {
	// Try to get session for navigation context
	session, _ := h.sessionManager.GetSession(r)

	data := TemplateData{
		Session: session,
	}

	if err := h.renderTemplate(w, "about", data); err != nil {
		h.logger.Printf("Error rendering about template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Login initiates OAuth flow
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	h.oauthManager.HandleOAuthLogin(w, r)
}

// Callback handles OAuth callback
func (h *Handlers) Callback(w http.ResponseWriter, r *http.Request) {
	h.oauthManager.HandleOAuthCallback(w, r)
}

// Logout clears the session
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	h.oauthManager.HandleLogout(w, r)
}

// Dashboard renders the user dashboard (protected route)
func (h *Handlers) Dashboard(w http.ResponseWriter, r *http.Request) {
	// Get session from context (set by RequireAuth middleware)
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	data := TemplateData{
		Session: session,
	}

	if err := h.renderTemplate(w, "dashboard", data); err != nil {
		h.logger.Printf("Error rendering dashboard template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
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
