package exporter

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// StreamDirectoryAsZIP creates a ZIP archive of the given directory and streams it to the writer
// This function uses memory-efficient streaming to handle large exports without loading
// the entire archive into memory. It walks the directory tree and adds each file to the ZIP.
//
// Parameters:
//   - dirPath: Full path to the directory to archive
//   - w: Writer to stream ZIP data to (typically http.ResponseWriter or io.Pipe)
//
// Returns error if directory doesn't exist, can't be read, or ZIP creation fails
func StreamDirectoryAsZIP(dirPath string, w io.Writer) error {
	// Verify directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("failed to stat directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dirPath)
	}

	// Create ZIP writer
	zipWriter := zip.NewWriter(w)
	defer zipWriter.Close()

	// Walk directory tree
	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		// Skip the root directory itself
		if path == dirPath {
			return nil
		}

		// Get relative path for ZIP entry
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Normalize path separators to forward slashes for ZIP compatibility
		zipPath := filepath.ToSlash(relPath)

		// Handle directories
		if info.IsDir() {
			// Create directory entry in ZIP (must end with /)
			_, err := zipWriter.Create(zipPath + "/")
			if err != nil {
				return fmt.Errorf("failed to create directory entry %s: %w", zipPath, err)
			}
			return nil
		}

		// Handle symbolic links
		if info.Mode()&os.ModeSymlink != 0 {
			// Follow symlink and add the target file
			target, err := os.Readlink(path)
			if err != nil {
				// Skip broken symlinks
				return nil
			}

			// If symlink target is relative, make it absolute
			if !filepath.IsAbs(target) {
				target = filepath.Join(filepath.Dir(path), target)
			}

			// Check if target exists
			targetInfo, err := os.Stat(target)
			if err != nil {
				// Skip broken symlinks
				return nil
			}

			// If target is a file, read it
			if !targetInfo.IsDir() {
				return addFileToZIP(zipWriter, target, zipPath)
			}

			// Skip symlinked directories to avoid loops
			return nil
		}

		// Handle regular files
		return addFileToZIP(zipWriter, path, zipPath)
	})

	if err != nil {
		return fmt.Errorf("failed to walk directory: %w", err)
	}

	// Close ZIP writer to finalize archive
	if err := zipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close ZIP writer: %w", err)
	}

	return nil
}

// addFileToZIP adds a single file to the ZIP archive
func addFileToZIP(zipWriter *zip.Writer, filePath, zipPath string) error {
	// Open source file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Get file info for metadata
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file %s: %w", filePath, err)
	}

	// Create ZIP file header
	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return fmt.Errorf("failed to create ZIP header for %s: %w", filePath, err)
	}

	// Set name to relative path (with forward slashes)
	header.Name = zipPath

	// Use DEFLATE compression for better compression ratios
	header.Method = zip.Deflate

	// Create entry in ZIP
	writer, err := zipWriter.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("failed to create ZIP entry for %s: %w", zipPath, err)
	}

	// Copy file contents to ZIP (streaming, no memory buffering)
	_, err = io.Copy(writer, file)
	if err != nil {
		return fmt.Errorf("failed to write file %s to ZIP: %w", filePath, err)
	}

	return nil
}

// StreamExportAsZIP is a convenience wrapper around StreamDirectoryAsZIP that takes an export ID
// and constructs the directory path. It's designed to be called from HTTP handlers.
//
// Parameters:
//   - exportID: Export ID in format "{did}/{timestamp}"
//   - baseExportDir: Base directory for all exports (typically "./exports")
//   - w: Writer to stream ZIP data to
//
// Returns error if export directory doesn't exist or streaming fails
func StreamExportAsZIP(exportID, baseExportDir string, w io.Writer) error {
	// Validate export ID format
	parts := strings.Split(exportID, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid export ID format: expected 'did/timestamp', got '%s'", exportID)
	}

	// Construct full directory path
	dirPath := filepath.Join(baseExportDir, exportID)

	// Stream the directory
	return StreamDirectoryAsZIP(dirPath, w)
}
