package models

import (
	"fmt"
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
