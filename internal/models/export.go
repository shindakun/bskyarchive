package models

import (
	"fmt"
	"strings"
	"time"
)

// ExportFormat specifies the output format for exports
type ExportFormat string

const (
	ExportFormatJSON ExportFormat = "json"
	ExportFormatCSV  ExportFormat = "csv"
)

// DateRange represents an optional time range filter for exports
type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// Validate checks if the date range is valid
func (dr *DateRange) Validate() error {
	if dr == nil {
		return nil
	}
	if dr.EndDate.Before(dr.StartDate) {
		return fmt.Errorf("end date must be after start date")
	}
	return nil
}

// ExportOptions defines user-configurable options for export operations
type ExportOptions struct {
	// Format specifies output format: "json" or "csv"
	Format ExportFormat `json:"format"`

	// OutputDir is the base directory for exports (default: "./exports")
	OutputDir string `json:"output_dir"`

	// IncludeMedia determines if media files should be copied
	IncludeMedia bool `json:"include_media"`

	// DateRange filters posts by creation date (nil = all posts)
	DateRange *DateRange `json:"date_range,omitempty"`

	// DID specifies which user's posts to export (required)
	DID string `json:"did"`
}

// Validate checks if export options are valid
func (opts *ExportOptions) Validate() error {
	if opts.Format != ExportFormatJSON && opts.Format != ExportFormatCSV {
		return fmt.Errorf("format must be 'json' or 'csv'")
	}
	if opts.DID == "" {
		return fmt.Errorf("DID is required")
	}
	if opts.DateRange != nil {
		if err := opts.DateRange.Validate(); err != nil {
			return fmt.Errorf("invalid date range: %w", err)
		}
	}
	return nil
}

// ExportStatus represents the current state of an export operation
type ExportStatus string

const (
	ExportStatusQueued    ExportStatus = "queued"
	ExportStatusRunning   ExportStatus = "running"
	ExportStatusCompleted ExportStatus = "completed"
	ExportStatusFailed    ExportStatus = "failed"
)

// ExportProgress tracks the current state of an export operation
type ExportProgress struct {
	PostsProcessed int          `json:"posts_processed"`
	PostsTotal     int          `json:"posts_total"`
	MediaCopied    int          `json:"media_copied"`
	MediaTotal     int          `json:"media_total"`
	Status         ExportStatus `json:"status"`
	Error          string       `json:"error,omitempty"` // Empty if no error
}

// PercentComplete calculates the overall completion percentage
func (p *ExportProgress) PercentComplete() int {
	if p.PostsTotal == 0 {
		return 0
	}
	return (p.PostsProcessed * 100) / p.PostsTotal
}

// ExportJob tracks an in-progress or completed export operation
type ExportJob struct {
	// ID uniquely identifies this export (timestamp-based)
	ID string `json:"id"`

	// Options used for this export
	Options ExportOptions `json:"options"`

	// Progress tracks current state
	Progress ExportProgress `json:"progress"`

	// ExportDir is the full path to this export's output directory
	ExportDir string `json:"export_dir"`

	// CreatedAt is when export was initiated
	CreatedAt time.Time `json:"created_at"`

	// CompletedAt is when export finished (nil if still running)
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ExportManifest describes the contents of an export
type ExportManifest struct {
	// ExportFormat is "json" or "csv"
	ExportFormat string `json:"export_format"`

	// ExportTimestamp is when export was created
	ExportTimestamp time.Time `json:"export_timestamp"`

	// PostCount is number of posts in export
	PostCount int `json:"post_count"`

	// MediaCount is number of media files copied
	MediaCount int `json:"media_count"`

	// DateRange describes filtered time period (null if no filter)
	DateRange *DateRange `json:"date_range,omitempty"`

	// Version is the bskyarchive version that created export
	Version string `json:"version"`

	// Files lists output files in export directory
	Files []string `json:"files"`
}

// ExportRecord represents a completed export in the database
// Used for tracking, listing, and managing exports for download/deletion
type ExportRecord struct {
	ID             string     `json:"id"`                           // Format: {did}/{timestamp}
	DID            string     `json:"did"`                          // Owner's DID for security
	Format         string     `json:"format"`                       // "json" or "csv"
	CreatedAt      time.Time  `json:"created_at"`                   // When export was created
	DirectoryPath  string     `json:"directory_path"`               // Full filesystem path
	PostCount      int        `json:"post_count"`                   // Number of posts
	MediaCount     int        `json:"media_count"`                  // Number of media files
	SizeBytes      int64      `json:"size_bytes"`                   // Total size in bytes
	DateRangeStart *time.Time `json:"date_range_start,omitempty"`  // Filter start (nullable)
	DateRangeEnd   *time.Time `json:"date_range_end,omitempty"`    // Filter end (nullable)
	ManifestPath   string     `json:"manifest_path,omitempty"`      // Path to manifest.json
}

// Validate checks if the export record is valid
func (e *ExportRecord) Validate() error {
	if e.ID == "" {
		return fmt.Errorf("ID is required")
	}
	if e.DID == "" {
		return fmt.Errorf("DID is required")
	}
	if e.Format != "json" && e.Format != "csv" {
		return fmt.Errorf("format must be 'json' or 'csv'")
	}
	if e.PostCount < 0 {
		return fmt.Errorf("post count must be >= 0")
	}
	if e.MediaCount < 0 {
		return fmt.Errorf("media count must be >= 0")
	}
	if e.SizeBytes < 0 {
		return fmt.Errorf("size must be >= 0")
	}
	if e.DateRangeStart != nil && e.DateRangeEnd != nil {
		if e.DateRangeEnd.Before(*e.DateRangeStart) {
			return fmt.Errorf("date range end must be after start")
		}
	}
	// Security validation: Prevent path traversal attacks
	if !strings.HasPrefix(e.DirectoryPath, "./exports/") {
		return fmt.Errorf("invalid directory path (security)")
	}

	// Additional security: Check for path traversal attempts
	if strings.Contains(e.DirectoryPath, "..") {
		return fmt.Errorf("invalid directory path: path traversal not allowed")
	}

	// Check for null bytes
	if strings.Contains(e.DirectoryPath, "\x00") {
		return fmt.Errorf("invalid directory path: null bytes not allowed")
	}

	return nil
}

// HumanSize returns size in human-readable format (KB, MB, GB)
func (e *ExportRecord) HumanSize() string {
	const unit = 1024
	if e.SizeBytes < unit {
		return fmt.Sprintf("%d B", e.SizeBytes)
	}
	div, exp := int64(unit), 0
	for n := e.SizeBytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(e.SizeBytes)/float64(div), "KMGTPE"[exp])
}

// DateRangeString returns formatted date range or "All posts"
func (e *ExportRecord) DateRangeString() string {
	if e.DateRangeStart == nil && e.DateRangeEnd == nil {
		return "All posts"
	}
	if e.DateRangeStart != nil && e.DateRangeEnd != nil {
		return fmt.Sprintf("%s to %s",
			e.DateRangeStart.Format("2006-01-02"),
			e.DateRangeEnd.Format("2006-01-02"))
	}
	if e.DateRangeStart != nil {
		return fmt.Sprintf("From %s", e.DateRangeStart.Format("2006-01-02"))
	}
	return fmt.Sprintf("Until %s", e.DateRangeEnd.Format("2006-01-02"))
}

// ExportListResponse is returned by the list exports API
type ExportListResponse struct {
	Exports []ExportRecord `json:"exports"`
	Total   int            `json:"total"`
}
