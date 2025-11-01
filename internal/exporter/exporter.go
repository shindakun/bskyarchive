package exporter

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
	"github.com/shindakun/bskyarchive/internal/version"
)

// CreateExportDirectory creates a timestamped export directory
// Returns the full path to the created directory
func CreateExportDirectory(baseDir string) (string, error) {
	// Generate timestamp in filesystem-safe format (YYYY-MM-DD_HH-MM-SS)
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	exportDir := filepath.Join(baseDir, timestamp)

	// Create the directory with read/write/execute for owner, read/execute for group and others
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	return exportDir, nil
}

// CheckDiskSpace validates that sufficient disk space is available for the export
// requiredBytes is the estimated space needed for the export
func CheckDiskSpace(path string, requiredBytes uint64) error {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	// Calculate available bytes
	availableBytes := stat.Bavail * uint64(stat.Bsize)

	if availableBytes < requiredBytes {
		return fmt.Errorf("insufficient disk space: need %d bytes, have %d bytes available",
			requiredBytes, availableBytes)
	}

	return nil
}

// Run executes a complete export operation with progress tracking
// This is the main orchestrator that coordinates all export steps
func Run(db *sql.DB, job *models.ExportJob, progressChan chan<- models.ExportProgress) error {
	defer close(progressChan)

	// Update status to running
	job.Progress.Status = models.ExportStatusRunning
	progressChan <- job.Progress

	// Step 1: Create export directory
	exportDir, err := CreateExportDirectory(job.Options.OutputDir)
	if err != nil {
		job.Progress.Status = models.ExportStatusFailed
		job.Progress.Error = fmt.Sprintf("Failed to create export directory: %v", err)
		progressChan <- job.Progress
		return err
	}
	job.ExportDir = exportDir

	// Create media subdirectory
	mediaDir := filepath.Join(exportDir, "media")
	if job.Options.IncludeMedia {
		if err := os.MkdirAll(mediaDir, 0755); err != nil {
			job.Progress.Status = models.ExportStatusFailed
			job.Progress.Error = fmt.Sprintf("Failed to create media directory: %v", err)
			progressChan <- job.Progress
			return err
		}
	}

	// Step 2: Fetch posts from database
	log.Printf("Fetching posts for export (DID: %s)", job.Options.DID)
	posts, err := storage.ListPostsWithDateRange(db, job.Options.DID, job.Options.DateRange, 0, 0)
	if err != nil {
		job.Progress.Status = models.ExportStatusFailed
		job.Progress.Error = fmt.Sprintf("Failed to fetch posts: %v", err)
		progressChan <- job.Progress
		return err
	}

	job.Progress.PostsTotal = len(posts)
	if len(posts) == 0 {
		// Handle empty archive gracefully - this is not an error condition
		job.Progress.Status = models.ExportStatusCompleted
		job.Progress.Error = "No posts found in your archive matching the selected criteria. Try adjusting your date range or archive some posts first."
		progressChan <- job.Progress

		// Still create manifest for consistency
		manifest := GenerateManifest(
			job.Options.Format,
			0, // postCount
			0, // mediaCount
			job.Options.DateRange,
			version.GetVersion(),
			[]string{}, // no files
		)

		manifestPath := filepath.Join(exportDir, "manifest.json")
		if err := WriteManifest(manifestPath, manifest); err != nil {
			log.Printf("Warning: Failed to write manifest: %v", err)
		}

		return nil // Not an error - just empty
	}

	// Step 3: Export posts to JSON or CSV
	var dataFile string
	if job.Options.Format == models.ExportFormatJSON {
		dataFile = filepath.Join(exportDir, "posts.json")
		if err := ExportToJSON(posts, dataFile); err != nil {
			job.Progress.Status = models.ExportStatusFailed
			job.Progress.Error = fmt.Sprintf("Failed to export JSON: %v", err)
			progressChan <- job.Progress
			return err
		}
	} else if job.Options.Format == models.ExportFormatCSV {
		dataFile = filepath.Join(exportDir, "posts.csv")
		if err := ExportToCSV(posts, dataFile); err != nil {
			job.Progress.Status = models.ExportStatusFailed
			job.Progress.Error = fmt.Sprintf("Failed to export CSV: %v", err)
			progressChan <- job.Progress
			return err
		}
	} else {
		job.Progress.Status = models.ExportStatusFailed
		job.Progress.Error = fmt.Sprintf("Unknown export format: %s", job.Options.Format)
		progressChan <- job.Progress
		return fmt.Errorf("unknown export format: %s", job.Options.Format)
	}

	job.Progress.PostsProcessed = len(posts)
	progressChan <- job.Progress

	// Step 4: Copy media files if requested
	if job.Options.IncludeMedia {
		// Build map of source -> destination paths for all media
		mediaFiles := make(map[string]string)

		for _, post := range posts {
			if !post.HasMedia {
				continue
			}

			mediaList, err := storage.ListMediaForPost(db, post.URI)
			if err != nil {
				log.Printf("Warning: failed to list media for post %s: %v", post.URI, err)
				continue
			}

			job.Progress.MediaTotal += len(mediaList)

			for _, media := range mediaList {
				// Source path is from database
				srcPath := media.FilePath
				// Destination preserves the same filename
				dstPath := filepath.Join(mediaDir, filepath.Base(media.FilePath))
				mediaFiles[srcPath] = dstPath
			}
		}

		// Copy all media files with progress tracking
		mediaChan := make(chan int, 100)
		go func() {
			for count := range mediaChan {
				job.Progress.MediaCopied = count
				progressChan <- job.Progress
			}
		}()

		copiedCount, err := CopyMediaFiles(mediaFiles, mediaChan)
		if err != nil {
			log.Printf("Warning: media copy had errors: %v", err)
		}
		job.Progress.MediaCopied = copiedCount
	}

	// Step 5: Generate manifest
	files, _ := GetExportFiles(exportDir)
	manifest := GenerateManifest(
		job.Options.Format,
		len(posts),
		job.Progress.MediaCopied,
		job.Options.DateRange,
		version.GetVersion(),
		files,
	)

	manifestPath := filepath.Join(exportDir, "manifest.json")
	if err := WriteManifest(manifestPath, manifest); err != nil {
		log.Printf("Warning: failed to write manifest: %v", err)
		// Don't fail the export for manifest errors
	}

	// Step 6: Mark as complete
	now := time.Now()
	job.CompletedAt = &now
	job.Progress.Status = models.ExportStatusCompleted
	progressChan <- job.Progress

	log.Printf("Export completed successfully: %s", exportDir)
	return nil
}
