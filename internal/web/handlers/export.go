package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
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

// ExportPage renders the export management page
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

	data := TemplateData{
		Session: session,
		Status:  status,
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
			fmt.Fprintf(w, `
				<p><strong>Export completed successfully!</strong></p>
				<p>Posts exported: %d</p>
				<p>Media files copied: %d</p>
				<p>Export directory: <code>%s</code></p>
				<a href="/dashboard" role="button">Return to Dashboard</a>
			`, job.Progress.PostsTotal, job.Progress.MediaCopied, job.ExportDir)

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
