package exporter

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

// CopyMediaFile copies a single media file from source to destination using io.Copy
func CopyMediaFile(srcPath, dstPath string) error {
	// Open source file
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer src.Close()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Create destination file
	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	// Copy the file using io.Copy (kernel-optimized)
	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return nil
}

// CopyMediaFiles copies multiple media files with progress tracking
// Returns the number of successfully copied files
func CopyMediaFiles(mediaFiles map[string]string, progressChan chan<- int) (int, error) {
	copiedCount := 0

	for srcPath, dstPath := range mediaFiles {
		// Check if source file exists
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			// Log warning but continue - missing media files shouldn't fail the entire export
			log.Printf("Warning: media file not found: %s", srcPath)
			continue
		}

		// Copy the file
		if err := CopyMediaFile(srcPath, dstPath); err != nil {
			log.Printf("Warning: failed to copy %s: %v", srcPath, err)
			continue
		}

		copiedCount++

		// Send progress update if channel provided
		if progressChan != nil {
			select {
			case progressChan <- copiedCount:
			default:
				// Non-blocking send - skip if channel is full
			}
		}
	}

	return copiedCount, nil
}
