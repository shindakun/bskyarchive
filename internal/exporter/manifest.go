package exporter

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
)

// WriteManifest generates and writes a manifest.json file describing the export contents
func WriteManifest(manifestPath string, manifest *models.ExportManifest) error {
	// Create the manifest file
	file, err := os.Create(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to create manifest file: %w", err)
	}
	defer file.Close()

	// Create JSON encoder with pretty-printing
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	// Write the manifest
	if err := encoder.Encode(manifest); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

// GenerateManifest creates a manifest struct from export metadata
func GenerateManifest(format models.ExportFormat, postCount, mediaCount int, dateRange *models.DateRange, version string, files []string) *models.ExportManifest {
	return &models.ExportManifest{
		ExportFormat:    string(format),
		ExportTimestamp: time.Now(),
		PostCount:       postCount,
		MediaCount:      mediaCount,
		DateRange:       dateRange,
		Version:         version,
		Files:           files,
	}
}

// GetExportFiles lists all files in the export directory
func GetExportFiles(exportDir string) ([]string, error) {
	var files []string

	entries, err := os.ReadDir(exportDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read export directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		} else if entry.Name() == "media" {
			// Count media files separately
			mediaPath := filepath.Join(exportDir, "media")
			mediaEntries, err := os.ReadDir(mediaPath)
			if err == nil {
				files = append(files, fmt.Sprintf("media/ (%d files)", len(mediaEntries)))
			}
		}
	}

	return files, nil
}
