package exporter

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
	"github.com/shindakun/bskyarchive/internal/version"
)

// CreateExportDirectory creates a per-user timestamped export directory
// Returns the full path to the created directory
// The directory structure is: baseDir/{did}/{timestamp}
func CreateExportDirectory(baseDir string, did string) (string, error) {
	// Preserve ./ prefix for relative paths
	// filepath.Join normalizes paths and can strip ./ prefix
	preserveDotSlash := strings.HasPrefix(baseDir, "./")

	// Create per-user subdirectory structure
	userDir := filepath.Join(baseDir, did)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create user export directory: %w", err)
	}

	// Generate timestamp in filesystem-safe format (YYYY-MM-DD_HH-MM-SS)
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	exportDir := filepath.Join(userDir, timestamp)

	// Create the timestamped directory with read/write/execute for owner, read/execute for group and others
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create export directory: %w", err)
	}

	// Restore ./ prefix if it was present in baseDir and got stripped by filepath.Join
	if preserveDotSlash && !strings.HasPrefix(exportDir, "./") && !filepath.IsAbs(exportDir) {
		exportDir = "./" + exportDir
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

	// Error recovery: cleanup partial export on failure
	var exportSucceeded bool
	defer func() {
		if !exportSucceeded && job.ExportDir != "" {
			// Export failed - clean up partial export directory
			log.Printf("Export failed, cleaning up partial export at: %s", job.ExportDir)
			if err := os.RemoveAll(job.ExportDir); err != nil {
				log.Printf("Warning: Failed to cleanup partial export: %v", err)
			} else {
				log.Printf("Partial export directory removed successfully")
			}
		}
	}()

	// Update status to running
	job.Progress.Status = models.ExportStatusRunning
	progressChan <- job.Progress

	// Step 1: Create export directory with per-user isolation
	exportDir, err := CreateExportDirectory(job.Options.OutputDir, job.Options.DID)
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

	// Step 2: Count total posts for progress tracking
	log.Printf("Counting posts for export (DID: %s)", job.Options.DID)

	// Build count query with same filters as export
	countQuery := "SELECT COUNT(*) FROM posts WHERE did = ?"
	args := []interface{}{job.Options.DID}

	// Add date range filters if specified
	if job.Options.DateRange != nil {
		if !job.Options.DateRange.StartDate.IsZero() {
			countQuery += " AND created_at >= ?"
			args = append(args, job.Options.DateRange.StartDate)
		}
		if !job.Options.DateRange.EndDate.IsZero() {
			countQuery += " AND created_at <= ?"
			args = append(args, job.Options.DateRange.EndDate)
		}
	}

	var totalPosts int
	if err := db.QueryRow(countQuery, args...).Scan(&totalPosts); err != nil {
		job.Progress.Status = models.ExportStatusFailed
		job.Progress.Error = fmt.Sprintf("Failed to count posts: %v", err)
		progressChan <- job.Progress
		return err
	}

	job.Progress.PostsTotal = totalPosts
	if totalPosts == 0 {
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

		exportSucceeded = true // Mark as successful to prevent cleanup
		return nil // Not an error - just empty
	}

	// Step 3: Export posts to JSON or CSV using batched streaming
	var dataFile string
	const batchSize = 1000 // Process 1000 posts at a time

	if job.Options.Format == models.ExportFormatJSON {
		dataFile = filepath.Join(exportDir, "posts.json")
		log.Printf("Starting batched JSON export (batch size: %d)", batchSize)
		if err := ExportToJSONBatched(db, job.Options.DID, job.Options.DateRange, dataFile, batchSize); err != nil {
			job.Progress.Status = models.ExportStatusFailed
			job.Progress.Error = fmt.Sprintf("Failed to export JSON: %v", err)
			progressChan <- job.Progress
			return err
		}
	} else if job.Options.Format == models.ExportFormatCSV {
		dataFile = filepath.Join(exportDir, "posts.csv")
		log.Printf("Starting batched CSV export (batch size: %d)", batchSize)
		if err := ExportToCSVBatched(db, job.Options.DID, job.Options.DateRange, dataFile, batchSize); err != nil {
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

	// Update progress after export completes
	job.Progress.PostsProcessed = totalPosts
	progressChan <- job.Progress

	// Step 4: Copy media files if requested
	if job.Options.IncludeMedia {
		// Build map of source -> destination paths for all media
		mediaFiles := make(map[string]string)

		// Fetch posts with media in batches to avoid loading all into memory
		offset := 0
		for {
			posts, err := storage.ListPostsWithDateRange(db, job.Options.DID, job.Options.DateRange, batchSize, offset)
			if err != nil {
				log.Printf("Warning: failed to fetch posts for media processing: %v", err)
				break
			}
			if len(posts) == 0 {
				break
			}

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

			offset += len(posts)
			if len(posts) < batchSize {
				break
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
		totalPosts,
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

	// Step 5.5: Calculate total export size and track in database
	var totalSize int64
	filepath.Walk(exportDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	// Create export record for tracking
	exportRecord := &models.ExportRecord{
		ID:            fmt.Sprintf("%s/%s", job.Options.DID, filepath.Base(exportDir)),
		DID:           job.Options.DID,
		Format:        string(job.Options.Format),
		CreatedAt:     job.CreatedAt,
		DirectoryPath: exportDir,
		PostCount:     totalPosts,
		MediaCount:    job.Progress.MediaCopied,
		SizeBytes:     totalSize,
		ManifestPath:  manifestPath,
	}

	if job.Options.DateRange != nil {
		if !job.Options.DateRange.StartDate.IsZero() {
			exportRecord.DateRangeStart = &job.Options.DateRange.StartDate
		}
		if !job.Options.DateRange.EndDate.IsZero() {
			exportRecord.DateRangeEnd = &job.Options.DateRange.EndDate
		}
	}

	// Save export record to database
	if err := storage.CreateExportRecord(db, exportRecord); err != nil {
		log.Printf("Warning: Failed to save export record to database: %v", err)
		// Don't fail the export - this is metadata only
		// User can still use the export, just won't see it in the list
	} else {
		log.Printf("Export record saved to database: %s (size: %d bytes)", exportRecord.ID, totalSize)
	}

	// Step 6: Mark as complete
	now := time.Now()
	job.CompletedAt = &now
	job.Progress.Status = models.ExportStatusCompleted
	progressChan <- job.Progress

	exportSucceeded = true // Mark as successful to prevent cleanup
	log.Printf("Export completed successfully: %s", exportDir)
	return nil
}
