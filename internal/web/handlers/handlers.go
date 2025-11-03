package handlers

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
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
	if err := h.renderTemplate(w, r, "landing", data); err != nil {
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

	if err := h.renderTemplate(w, r, "about", data); err != nil {
		h.logger.Printf("Error rendering about template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// Login initiates OAuth flow
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	// GET: Display login form
	if r.Method == http.MethodGet {
		data := TemplateData{
			Error:   "",
			Message: "",
		}
		if err := h.renderTemplate(w, r, "login", data); err != nil {
			h.logger.Printf("Error rendering login template: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// POST: Handle handle submission and start OAuth flow
	handle := r.FormValue("handle")
	if handle == "" {
		// Validation error - re-render template with error
		data := TemplateData{
			Error:   "Bluesky handle is required",
			Message: "",
			Handle:  handle, // Repopulate form (empty in this case)
		}
		if err := h.renderTemplate(w, r, "login", data); err != nil {
			h.logger.Printf("Error rendering login template with error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Start OAuth flow
	authURL, err := h.oauthManager.StartOAuthFlow(r.Context(), handle)
	if err != nil {
		// OAuth error - re-render template with error
		h.logger.Printf("Failed to start OAuth flow for handle %s: %v", handle, err)
		data := TemplateData{
			Error:   "Failed to connect to Bluesky. Please try again.",
			Message: "",
			Handle:  handle, // Repopulate form so user doesn't have to retype
		}
		if renderErr := h.renderTemplate(w, r, "login", data); renderErr != nil {
			h.logger.Printf("Error rendering login template with OAuth error: %v", renderErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
		return
	}

	// Redirect to Bluesky authorization page
	http.Redirect(w, r, authURL, http.StatusSeeOther)
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

	if err := h.renderTemplate(w, r, "dashboard", data); err != nil {
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

	if err := h.renderTemplate(w, r, "archive", data); err != nil {
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

	if err := h.renderTemplate(w, r, "browse", data); err != nil {
		h.logger.Printf("Error rendering browse template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// ServeMedia serves archived media files with path traversal protection
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

	// Validate the file path to prevent path traversal
	// Get absolute path of the media file
	absMediaPath, err := filepath.Abs(media.FilePath)
	if err != nil {
		h.logger.Printf("Security: Failed to resolve media path: %v", err)
		http.NotFound(w, r)
		return
	}

	// Get the expected media directory (typically "media/" from project root)
	mediaDir := "media"
	absMediaDir, err := filepath.Abs(mediaDir)
	if err != nil {
		h.logger.Printf("Security: Failed to resolve media directory: %v", err)
		http.NotFound(w, r)
		return
	}

	// Verify the resolved path is within the media directory (path traversal protection)
	if !strings.HasPrefix(absMediaPath, absMediaDir+string(filepath.Separator)) &&
		absMediaPath != absMediaDir {
		h.logger.Printf("Security: Path traversal attempt blocked in media - requested: %s, resolved: %s", media.FilePath, absMediaPath)
		http.NotFound(w, r)
		return
	}

	// Verify file exists and is not a directory
	fileInfo, err := os.Stat(absMediaPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if fileInfo.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Serve the file
	http.ServeFile(w, r, absMediaPath)
}

// ServeStatic serves static files with path traversal protection
func (h *Handlers) ServeStatic(w http.ResponseWriter, r *http.Request) {
	// Remove /static prefix
	path := strings.TrimPrefix(r.URL.Path, "/static/")
	if path == "" || path == "/" {
		http.NotFound(w, r)
		return
	}

	// Clean the path to remove any path traversal attempts
	cleanPath := filepath.Clean(path)

	// Build full path from static directory
	staticDir := filepath.Join("internal", "web", "static")
	fullPath := filepath.Join(staticDir, cleanPath)

	// Get absolute paths for validation
	absStaticDir, err := filepath.Abs(staticDir)
	if err != nil {
		h.logger.Printf("Security: Failed to resolve static directory: %v", err)
		http.NotFound(w, r)
		return
	}

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		h.logger.Printf("Security: Failed to resolve requested path: %v", err)
		http.NotFound(w, r)
		return
	}

	// Verify the resolved path is within the static directory (path traversal protection)
	if !strings.HasPrefix(absFullPath, absStaticDir+string(filepath.Separator)) &&
		absFullPath != absStaticDir {
		h.logger.Printf("Security: Path traversal attempt blocked - requested: %s, resolved: %s", path, absFullPath)
		http.NotFound(w, r)
		return
	}

	// Verify file exists and is not a directory
	fileInfo, err := os.Stat(absFullPath)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if fileInfo.IsDir() {
		http.NotFound(w, r)
		return
	}

	// Serve the file
	http.ServeFile(w, r, absFullPath)
}

// NotFound renders the 404 error page
func (h *Handlers) NotFound(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionManager.GetSession(r)

	data := TemplateData{
		Session: session,
	}

	w.WriteHeader(http.StatusNotFound)
	if err := h.renderTemplate(w, r, "404", data); err != nil {
		h.logger.Printf("Error rendering 404 template: %v", err)
		http.Error(w, "Not Found", http.StatusNotFound)
	}
}
