package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/shindakun/bskyarchive/internal/auth"
	"github.com/shindakun/bskyarchive/internal/exporter"
	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
)

// exportJobs stores active and completed export jobs
// In a production system, this would be stored in the database
var (
	exportJobs   = make(map[string]*models.ExportJob)
	exportJobsMu sync.RWMutex
)

// Download rate limiting state
// Tracks concurrent downloads per DID to prevent resource exhaustion
var (
	activeDownloads   = make(map[string]int) // DID -> count
	activeDownloadsMu sync.RWMutex
	maxDownloadsPerUser = 10 // Maximum concurrent downloads per user
)

// ExportPage renders the export management page with list of user's exports
func (h *Handlers) ExportPage(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Redirect(w, r, "/auth/login", http.StatusSeeOther)
		return
	}

	// Get archive status for display
	status, err := storage.GetArchiveStatus(h.db, session.DID)
	if err != nil {
		h.logger.Printf("Error getting archive status: %v", err)
	}

	// Get list of user's exports (most recent first, limit 50)
	exports, err := storage.ListExportsByDID(h.db, session.DID, 50, 0)
	if err != nil {
		h.logger.Printf("Error listing exports: %v", err)
		// Continue rendering page even if exports can't be loaded
	}

	data := TemplateData{
		Session: session,
		Status:  status,
		Exports: exports, // Pass exports to template
	}

	if err := h.renderTemplate(w, r, "export", data); err != nil {
		h.logger.Printf("Error rendering export template: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// StartExport initiates a new export operation
func (h *Handlers) StartExport(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse form data
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	format := r.FormValue("format")
	includeMedia := r.FormValue("include_media") == "true"
	startDateStr := r.FormValue("start_date")
	endDateStr := r.FormValue("end_date")

	// Validate format
	var exportFormat models.ExportFormat
	if format == "csv" {
		exportFormat = models.ExportFormatCSV
	} else {
		exportFormat = models.ExportFormatJSON // default
	}

	// Parse and validate date range
	var dateRange *models.DateRange
	if startDateStr != "" || endDateStr != "" {
		// Parse dates
		var startDate, endDate time.Time
		var err error

		if startDateStr != "" {
			startDate, err = time.Parse("2006-01-02", startDateStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid start date format: %v", err), http.StatusBadRequest)
				return
			}
		}

		if endDateStr != "" {
			endDate, err = time.Parse("2006-01-02", endDateStr)
			if err != nil {
				http.Error(w, fmt.Sprintf("Invalid end date format: %v", err), http.StatusBadRequest)
				return
			}
			// Set to end of day
			endDate = endDate.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
		}

		// Validate date range
		if !startDate.IsZero() && !endDate.IsZero() {
			if endDate.Before(startDate) {
				http.Error(w, "End date must be after start date", http.StatusBadRequest)
				return
			}
		}

		// Check for future dates
		now := time.Now()
		if !startDate.IsZero() && startDate.After(now) {
			http.Error(w, "Start date cannot be in the future", http.StatusBadRequest)
			return
		}
		if !endDate.IsZero() && endDate.After(now) {
			http.Error(w, "End date cannot be in the future", http.StatusBadRequest)
			return
		}

		// Create date range
		dateRange = &models.DateRange{
			StartDate: startDate,
			EndDate:   endDate,
		}

		// Validate using model's Validate method
		if err := dateRange.Validate(); err != nil {
			http.Error(w, fmt.Sprintf("Invalid date range: %v", err), http.StatusBadRequest)
			return
		}
	}

	// Create export options
	opts := models.ExportOptions{
		Format:       exportFormat,
		OutputDir:    "./exports",
		IncludeMedia: includeMedia,
		DID:          session.DID,
		DateRange:    dateRange,
	}

	// Validate options
	if err := opts.Validate(); err != nil {
		http.Error(w, fmt.Sprintf("Invalid export options: %v", err), http.StatusBadRequest)
		return
	}

	// Check for concurrent exports (prevent multiple exports running at once)
	exportJobsMu.RLock()
	for _, existingJob := range exportJobs {
		// Check if there's an active export for this user
		if existingJob.Options.DID == session.DID &&
			(existingJob.Progress.Status == models.ExportStatusQueued ||
				existingJob.Progress.Status == models.ExportStatusRunning) {
			exportJobsMu.RUnlock()
			http.Error(w, "An export is already in progress. Please wait for it to complete before starting a new one.", http.StatusConflict)
			return
		}
	}
	exportJobsMu.RUnlock()

	// Create export job
	jobID := time.Now().Format("2006-01-02_15-04-05")
	job := &models.ExportJob{
		ID:        jobID,
		Options:   opts,
		CreatedAt: time.Now(),
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Store job
	exportJobsMu.Lock()
	exportJobs[jobID] = job
	exportJobsMu.Unlock()

	// Start export in background
	go func() {
		progressChan := make(chan models.ExportProgress, 100)

		// Update progress in background
		go func() {
			for progress := range progressChan {
				exportJobsMu.Lock()
				if j, exists := exportJobs[jobID]; exists {
					j.Progress = progress
				}
				exportJobsMu.Unlock()
			}
		}()

		// Run the export
		if err := exporter.Run(h.db, job, progressChan); err != nil {
			h.logger.Printf("Export failed: %v", err)
		}
	}()

	// Return success with job ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":  jobID,
		"status":  "running",
		"message": "Export started successfully",
	})
}

// ExportProgress returns the current progress of an export job
func (h *Handlers) ExportProgress(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	jobID := chi.URLParam(r, "job_id")
	if jobID == "" {
		http.Error(w, "Job ID required", http.StatusBadRequest)
		return
	}

	// Get job
	exportJobsMu.RLock()
	job, exists := exportJobs[jobID]
	exportJobsMu.RUnlock()

	if !exists {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	// Verify job ownership - users can only access their own exports
	if job.Options.DID != session.DID {
		h.logger.Printf("Security: Unauthorized export access attempt - user %s attempted to access job %s owned by %s",
			session.DID, jobID, job.Options.DID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check if this is an HTMX request
	if r.Header.Get("HX-Request") == "true" {
		// Return HTML fragment for HTMX
		w.Header().Set("Content-Type", "text/html")

		switch job.Progress.Status {
		case models.ExportStatusRunning:
			percent := job.Progress.PercentComplete()
			fmt.Fprintf(w, `
				<p>Exporting posts: %d / %d (%d%%)</p>
				<progress value="%d" max="100"></progress>
				<p>Media files copied: %d / %d</p>
			`, job.Progress.PostsProcessed, job.Progress.PostsTotal, percent, percent,
				job.Progress.MediaCopied, job.Progress.MediaTotal)

		case models.ExportStatusCompleted:
			// Extract export ID from ExportDir (format: ./exports/did:plc:xxx/timestamp)
			// The ID is: did:plc:xxx/timestamp
			exportID := ""
			if len(job.ExportDir) > len("./exports/") {
				exportID = job.ExportDir[len("./exports/"):]
			}
			fmt.Fprintf(w, `
				<div data-export-id="%s">
					<p><strong>Export completed successfully!</strong></p>
					<p>Posts exported: %d</p>
					<p>Media files copied: %d</p>
					<p>Export directory: <code>%s</code></p>
					<a href="/dashboard" role="button">Return to Dashboard</a>
				</div>
			`, exportID, job.Progress.PostsTotal, job.Progress.MediaCopied, job.ExportDir)

		case models.ExportStatusFailed:
			fmt.Fprintf(w, `
				<p><strong>Export failed</strong></p>
				<p>Error: %s</p>
				<a href="/export" role="button">Try Again</a>
			`, job.Progress.Error)

		default:
			fmt.Fprintf(w, `<p>Export status: %s</p>`, job.Progress.Status)
		}
		return
	}

	// Return JSON for API requests
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"job_id":          jobID,
		"status":          job.Progress.Status,
		"posts_processed": job.Progress.PostsProcessed,
		"posts_total":     job.Progress.PostsTotal,
		"media_copied":    job.Progress.MediaCopied,
		"media_total":     job.Progress.MediaTotal,
		"percent_complete": job.Progress.PercentComplete(),
		"error":           job.Progress.Error,
	})
}

// sanitizeID converts an export ID to a valid CSS selector ID
// by replacing special characters with hyphens
func sanitizeID(id string) string {
	replacer := strings.NewReplacer(
		":", "-",
		"/", "-",
		" ", "-",
	)
	sanitized := replacer.Replace(id)
	// Remove any remaining non-alphanumeric characters except hyphens
	var result strings.Builder
	for _, ch := range sanitized {
		if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '-' || ch == '_' {
			result.WriteRune(ch)
		}
	}
	return result.String()
}

// ExportRow returns a single export as an HTML table row fragment for HTMX
func (h *Handlers) ExportRow(w http.ResponseWriter, r *http.Request) {
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get export ID from URL wildcard path
	exportID := chi.URLParam(r, "*")
	if exportID == "" {
		http.Error(w, "Export ID required", http.StatusBadRequest)
		return
	}

	// Retrieve export record from database
	exportRecord, err := storage.GetExportByID(h.db, exportID)
	if err != nil {
		http.Error(w, "Export not found", http.StatusNotFound)
		return
	}

	// Verify ownership - users can only access their own exports
	if exportRecord.DID != session.DID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Get CSRF token for the delete button
	csrfToken := csrf.Token(r)

	// Return HTML table row
	w.Header().Set("Content-Type", "text/html")
	sanitizedID := sanitizeID(exportRecord.ID)
	fmt.Fprintf(w, `<tr id="export-%s" data-export-id="%s">
		<td>%s</td>
		<td>%s</td>
		<td>%s</td>
		<td>%d</td>
		<td>%d</td>
		<td>%s</td>
		<td>
			<div style="display: flex; flex-direction: column; gap: 0.5rem;">
				<div style="display: flex; gap: 0.5rem;">
					<a href="/export/download/%s"
					   role="button"
					   class="secondary download-btn"
					   style="margin: 0; flex: 1;"
					   data-export-id="%s">
						Download ZIP
					</a>
					<button type="button"
							class="outline delete-export-btn"
							style="margin: 0;"
							hx-delete="/export/delete/%s"
							hx-confirm="Are you sure you want to delete this export? This action cannot be undone."
							hx-target="#export-%s"
							hx-swap="outerHTML"
							hx-headers='{"X-CSRF-Token": "%s"}'>
						Delete
					</button>
				</div>
				<label style="margin: 0; font-size: 0.875rem;">
					<input type="checkbox"
					       class="delete-after-checkbox"
					       data-export-id="%s"
					       style="margin-right: 0.25rem;">
					Delete after download
				</label>
			</div>
		</td>
	</tr>`,
		sanitizedID,     // For tr id="export-%s"
		exportRecord.ID, // For tr data-export-id="%s" (original ID)
		exportRecord.CreatedAt.Format("2006-01-02 15:04"),
		exportRecord.Format,
		exportRecord.DateRangeString(),
		exportRecord.PostCount,
		exportRecord.MediaCount,
		exportRecord.HumanSize(),
		exportRecord.ID, // For download link href
		exportRecord.ID, // For download button data-export-id
		exportRecord.ID, // For delete button hx-delete
		sanitizedID,     // For delete button hx-target="#export-%s"
		csrfToken,       // For delete button X-CSRF-Token
		exportRecord.ID, // For checkbox data-export-id
	)
}

// DownloadExport streams an export as a ZIP archive for download
// Implements rate limiting, authentication, and ownership verification
func (h *Handlers) DownloadExport(w http.ResponseWriter, r *http.Request) {
	// Get session from context
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		h.logger.Printf("Download attempt without authentication")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get export ID from URL wildcard path
	// The wildcard captures everything after /export/download/
	// For example: /export/download/did:plc:xxx/timestamp -> "did:plc:xxx/timestamp"
	exportID := chi.URLParam(r, "*")
	if exportID == "" {
		h.logger.Printf("Download attempt without export ID")
		http.Error(w, "Export ID required", http.StatusBadRequest)
		return
	}

	// Retrieve export record from database
	exportRecord, err := storage.GetExportByID(h.db, exportID)
	if err != nil {
		h.logger.Printf("Export not found: %s (error: %v)", exportID, err)
		http.Error(w, "Export not found", http.StatusNotFound)
		return
	}

	// Verify ownership - users can only download their own exports
	if exportRecord.DID != session.DID {
		h.logger.Printf("Security: Unauthorized download attempt - user %s attempted to download export %s owned by %s",
			session.DID, exportID, exportRecord.DID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Check rate limit
	activeDownloadsMu.Lock()
	currentDownloads := activeDownloads[session.DID]
	if currentDownloads >= maxDownloadsPerUser {
		activeDownloadsMu.Unlock()
		h.logger.Printf("Rate limit exceeded for user %s (%d concurrent downloads)",
			session.DID, currentDownloads)
		http.Error(w, "Too many concurrent downloads. Please wait for current downloads to complete.",
			http.StatusTooManyRequests)
		return
	}
	activeDownloads[session.DID]++
	activeDownloadsMu.Unlock()

	// Cleanup rate limit tracker when done
	defer func() {
		activeDownloadsMu.Lock()
		activeDownloads[session.DID]--
		if activeDownloads[session.DID] == 0 {
			delete(activeDownloads, session.DID)
		}
		activeDownloadsMu.Unlock()
	}()

	// Verify export directory exists
	if _, err := os.Stat(exportRecord.DirectoryPath); os.IsNotExist(err) {
		h.logger.Printf("Export directory missing for %s: %s", exportID, exportRecord.DirectoryPath)
		http.Error(w, "Export files not found", http.StatusNotFound)
		return
	}

	// Set headers for ZIP download
	filename := fmt.Sprintf("bskyarchive-%s.zip", exportRecord.Format)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	// Check if user wants to delete export after download (T043)
	deleteAfter := r.URL.Query().Get("delete_after") == "true"

	// Log download start (audit logging - T020)
	h.logger.Printf("Download started: user=%s export=%s format=%s size=%d delete_after=%v",
		session.DID, exportID, exportRecord.Format, exportRecord.SizeBytes, deleteAfter)

	// Stream the ZIP archive
	if err := exporter.StreamDirectoryAsZIP(exportRecord.DirectoryPath, w); err != nil {
		h.logger.Printf("Failed to stream export %s: %v", exportID, err)
		// Can't send HTTP error after streaming starts, just log it
		// IMPORTANT: Do NOT delete export if download failed (T042, T048)
		return
	}

	// Log successful download (audit logging - T020)
	h.logger.Printf("Download completed: user=%s export=%s size=%d",
		session.DID, exportID, exportRecord.SizeBytes)

	// Delete export after successful download if requested (T044)
	if deleteAfter {
		if err := deleteExportInternal(h.db, exportID); err != nil {
			h.logger.Printf("Warning: Failed to delete export %s after download: %v", exportID, err)
			// Don't fail the download - it completed successfully
			// The export will remain and can be deleted manually
		} else {
			h.logger.Printf("Export deleted after download: user=%s export=%s",
				session.DID, exportID)
		}
	}
}

// DeleteExport handles deletion of an export with authentication and ownership checks
func (h *Handlers) DeleteExport(w http.ResponseWriter, r *http.Request) {
	// Get session from context
	session, ok := auth.GetSessionFromContext(r.Context())
	if !ok || session == nil {
		h.logger.Printf("Delete attempt without authentication")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get export ID from URL wildcard path
	exportID := chi.URLParam(r, "*")
	if exportID == "" {
		h.logger.Printf("Delete attempt without export ID")
		http.Error(w, "Export ID required", http.StatusBadRequest)
		return
	}

	// Retrieve export record from database
	exportRecord, err := storage.GetExportByID(h.db, exportID)
	if err != nil {
		h.logger.Printf("Export not found for deletion: %s (error: %v)", exportID, err)
		http.Error(w, "Export not found", http.StatusNotFound)
		return
	}

	// Verify ownership - users can only delete their own exports
	if exportRecord.DID != session.DID {
		h.logger.Printf("Security: Unauthorized delete attempt - user %s attempted to delete export %s owned by %s",
			session.DID, exportID, exportRecord.DID)
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	// Audit log: deletion started (T038)
	h.logger.Printf("Delete started: user=%s export=%s size=%d",
		session.DID, exportID, exportRecord.SizeBytes)

	// Perform deletion
	if err := deleteExportInternal(h.db, exportID); err != nil {
		h.logger.Printf("Failed to delete export %s: %v", exportID, err)
		http.Error(w, "Failed to delete export", http.StatusInternalServerError)
		return
	}

	// Audit log: deletion completed (T038)
	h.logger.Printf("Delete completed: user=%s export=%s",
		session.DID, exportID)

	// Return 200 OK with empty body for HTMX outerHTML swap
	// The hx-swap="outerHTML" with empty response will remove the target element
	w.WriteHeader(http.StatusOK)
	w.Write([]byte{})
}

// deleteExportInternal performs the actual deletion of export files and database record
// This helper function allows for easier testing and potential reuse
func deleteExportInternal(db *sql.DB, exportID string) error {
	// Get export record to get directory path
	exportRecord, err := storage.GetExportByID(db, exportID)
	if err != nil {
		return fmt.Errorf("failed to retrieve export record: %w", err)
	}

	// Delete directory and all contents
	// RemoveAll handles non-existent paths gracefully (returns nil)
	if exportRecord.DirectoryPath != "" {
		if err := os.RemoveAll(exportRecord.DirectoryPath); err != nil {
			// Log warning but continue with DB deletion
			// This handles permission errors or concurrent deletions
			log.Printf("Warning: Failed to delete export directory %s: %v",
				exportRecord.DirectoryPath, err)
		}
	}

	// Delete database record
	if err := storage.DeleteExport(db, exportID); err != nil {
		return fmt.Errorf("failed to delete export record: %w", err)
	}

	return nil
}
