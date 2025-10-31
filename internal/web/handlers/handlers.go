package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/shindakun/bskyarchive/internal/archiver"
	"github.com/shindakun/bskyarchive/internal/auth"
	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
	"github.com/shindakun/bskyarchive/internal/version"
)

// Handlers holds dependencies for HTTP handlers
type Handlers struct {
	db             *sql.DB
	sessionManager *auth.SessionManager
	oauthManager   *auth.OAuthManager
	worker         *archiver.Worker
	logger         *log.Logger
}

// New creates a new Handlers instance
func New(db *sql.DB, sessionManager *auth.SessionManager, oauthManager *auth.OAuthManager, worker *archiver.Worker, logger *log.Logger) *Handlers {
	return &Handlers{
		db:             db,
		sessionManager: sessionManager,
		oauthManager:   oauthManager,
		worker:         worker,
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
		Version: version.GetVersion(),
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

	// Fetch archive status
	status, err := storage.GetArchiveStatus(h.db, session.DID)
	if err != nil {
		h.logger.Printf("Error fetching archive status: %v", err)
		// Continue anyway, just don't show status
		status = nil
	}

	data := TemplateData{
		Session: session,
		Status:  status,
	}

	if err := h.renderTemplate(w, "dashboard", data); err != nil {
		h.logger.Printf("Error rendering dashboard template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Archive renders the archive management page
func (h *Handlers) Archive(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Fetch archive status
	status, err := storage.GetArchiveStatus(h.db, session.DID)
	if err != nil {
		h.logger.Printf("Error fetching archive status: %v", err)
		status = nil
	}

	data := TemplateData{
		Session:            session,
		Status:             status,
		HasActiveOperation: status != nil && status.HasActiveOperation(),
	}

	if err := h.renderTemplate(w, "archive", data); err != nil {
		h.logger.Printf("Error rendering archive template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ArchiveStart initiates an archive operation
func (h *Handlers) ArchiveStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse operation type from request - support both JSON and form data
	var operationType string

	contentType := r.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		var req struct {
			Type string `json:"type"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.logger.Printf("Failed to decode JSON: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		operationType = req.Type
	} else {
		// Parse as form data
		if err := r.ParseForm(); err != nil {
			h.logger.Printf("Failed to parse form: %v", err)
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}
		operationType = r.FormValue("type")
	}

	if operationType == "" {
		h.logger.Printf("Missing operation type")
		http.Error(w, "Operation type is required", http.StatusBadRequest)
		return
	}

	// Validate operation type
	var opType models.OperationType
	switch operationType {
	case "initial":
		opType = models.OperationTypeInitial
	case "incremental":
		opType = models.OperationTypeIncremental
	case "refresh":
		opType = models.OperationTypeRefresh
	default:
		h.logger.Printf("Invalid operation type: %s", operationType)
		http.Error(w, "Invalid operation type", http.StatusBadRequest)
		return
	}

	// Start archive operation
	// session.AccessToken now contains the bskyoauth session ID
	operationID, err := h.worker.StartArchive(r.Context(), session.DID, session.AccessToken, opType)
	if err != nil {
		h.logger.Printf("Failed to start archive: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.logger.Printf("Started archive operation %s for DID %s", operationID, session.DID)

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"operation_id": operationID,
		"status":       "started",
	})
}

// ArchiveStatus returns the status of an archive operation (for HTMX polling)
func (h *Handlers) ArchiveStatus(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Fetch archive status
	status, err := storage.GetArchiveStatus(h.db, session.DID)
	if err != nil {
		h.logger.Printf("Error fetching archive status: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	data := TemplateData{
		Session:            session,
		Status:             status,
		HasActiveOperation: status != nil && status.HasActiveOperation(),
	}

	// Render partial template for HTMX
	if err := h.renderPartial(w, "archive-status", data); err != nil {
		h.logger.Printf("Error rendering archive status partial: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Browse renders the post browsing page
func (h *Handlers) Browse(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get query parameters
	query := r.URL.Query().Get("q")
	pageStr := r.URL.Query().Get("page")
	showAll := r.URL.Query().Get("all") == "true"

	page := 1
	if pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	offset := (page - 1) * pageSize

	var posts []models.Post
	var total int
	var totalPages int

	// Determine which DID to filter by
	filterDID := session.DID
	if showAll {
		filterDID = "" // Empty string means show all posts
	}

	// Fetch posts (search or list)
	if query != "" {
		// Search posts
		result, err := storage.SearchPosts(h.db, filterDID, query, pageSize, offset)
		if err != nil {
			h.logger.Printf("Error searching posts: %v", err)
			posts = []models.Post{}
		} else {
			posts = result.Posts
			total = result.Total
		}
	} else {
		// List all posts
		result, err := storage.ListPosts(h.db, filterDID, pageSize, offset)
		if err != nil {
			h.logger.Printf("Error listing posts: %v", err)
			posts = []models.Post{}
		} else {
			posts = result.Posts
			total = result.Total
			totalPages = result.TotalPages
		}
	}

	// Calculate total pages if not already set
	if totalPages == 0 && total > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}

	// Fetch media for all posts
	mediaMap := make(map[string][]models.Media)
	for _, post := range posts {
		media, err := storage.ListMediaForPost(h.db, post.URI)
		if err != nil {
			h.logger.Printf("Warning: failed to fetch media for post %s: %v", post.URI, err)
			continue
		}
		if len(media) > 0 {
			mediaMap[post.URI] = media
		}
	}

	// Check which parent posts exist in the archive
	parentPostsInArchive := make(map[string]bool)
	for _, post := range posts {
		if post.IsReply && post.ReplyParent != "" {
			_, err := storage.GetPost(h.db, post.ReplyParent)
			if err == nil {
				// Parent post exists in archive
				parentPostsInArchive[post.ReplyParent] = true
			}
		}
	}

	// Fetch profiles for all DIDs in posts (for handle display)
	profilesMap := make(map[string]string)
	rows, err := h.db.Query("SELECT did, handle FROM profiles")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var did, handle string
			if err := rows.Scan(&did, &handle); err == nil {
				profilesMap[did] = handle
			}
		}
	}

	data := TemplateData{
		Session:              session,
		Posts:                posts,
		Media:                mediaMap,
		ParentPostsInArchive: parentPostsInArchive,
		Profiles:             profilesMap,
		Query:                query,
		Page:                 page,
		Total:                total,
		PageSize:             pageSize,
		TotalPages:           totalPages,
		ShowAll:              showAll,
	}

	if err := h.renderTemplate(w, "browse", data); err != nil {
		h.logger.Printf("Error rendering browse template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ServeMedia serves archived media files
func (h *Handlers) ServeMedia(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Extract hash from chi URL parameter
	hash := chi.URLParam(r, "hash")
	if hash == "" {
		http.Error(w, "Invalid media hash", http.StatusBadRequest)
		return
	}

	// Get media metadata from database
	media, err := storage.GetMediaByHash(h.db, hash)
	if err != nil {
		http.Error(w, "Media not found", http.StatusNotFound)
		return
	}

	// Open and serve the file
	http.ServeFile(w, r, media.FilePath)
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

// NotFound renders the 404 error page
func (h *Handlers) NotFound(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionManager.GetSession(r)

	data := TemplateData{
		Session: session,
	}

	w.WriteHeader(http.StatusNotFound)
	if err := h.renderTemplate(w, "404", data); err != nil {
		h.logger.Printf("Error rendering 404 template: %v", err)
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}
