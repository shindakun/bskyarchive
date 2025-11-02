package unit

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/shindakun/bskyarchive/internal/exporter"
)

// TestZIPStreamingIntegrity verifies ZIP archive structure and integrity
func TestZIPStreamingIntegrity(t *testing.T) {
	// Create test export directory structure
	tmpDir := t.TempDir()
	exportDir := filepath.Join(tmpDir, "test_export")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}

	// Create realistic export structure
	files := map[string][]byte{
		"posts.json": []byte(`{
			"posts": [
				{"uri": "at://did:plc:test/app.bsky.feed.post/test1", "text": "Hello world"},
				{"uri": "at://did:plc:test/app.bsky.feed.post/test2", "text": "Test post"}
			]
		}`),
		"manifest.json": []byte(`{
			"export_format": "json",
			"export_timestamp": "2025-11-01T12:00:00Z",
			"post_count": 2,
			"media_count": 1,
			"version": "1.0.0"
		}`),
		"media/test_image.jpg": []byte("fake image binary data"),
	}

	for path, content := range files {
		fullPath := filepath.Join(exportDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			t.Fatalf("Failed to write %s: %v", path, err)
		}
	}

	// Stream to ZIP
	var buf bytes.Buffer
	if err := exporter.StreamDirectoryAsZIP(exportDir, &buf); err != nil {
		t.Fatalf("Failed to stream directory as ZIP: %v", err)
	}

	// Verify ZIP structure
	zipData := buf.Bytes()
	zipReader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("Failed to read ZIP: %v", err)
	}

	// Test 1: Count files (excluding directory entries)
	fileCount := 0
	for _, zipFile := range zipReader.File {
		if !zipFile.FileInfo().IsDir() {
			fileCount++
		}
	}
	if fileCount != len(files) {
		t.Errorf("Expected %d files in ZIP, got %d", len(files), fileCount)
	}

	// Test 2: Verify each file's integrity
	for _, zipFile := range zipReader.File {
		// Skip directory entries
		if zipFile.FileInfo().IsDir() {
			continue
		}

		expectedContent, exists := files[zipFile.Name]
		if !exists {
			t.Errorf("Unexpected file in ZIP: %s", zipFile.Name)
			continue
		}

		// Open and read file from ZIP
		rc, err := zipFile.Open()
		if err != nil {
			t.Errorf("Failed to open %s from ZIP: %v", zipFile.Name, err)
			continue
		}

		actualContent, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Errorf("Failed to read %s from ZIP: %v", zipFile.Name, err)
			continue
		}

		// Verify content matches exactly
		if !bytes.Equal(actualContent, expectedContent) {
			t.Errorf("Content mismatch for %s\nExpected length: %d\nActual length: %d",
				zipFile.Name, len(expectedContent), len(actualContent))
		}

		// Verify uncompressed size matches
		if zipFile.UncompressedSize64 != uint64(len(expectedContent)) {
			t.Errorf("Uncompressed size mismatch for %s: expected %d, got %d",
				zipFile.Name, len(expectedContent), zipFile.UncompressedSize64)
		}
	}

	// Test 3: Verify directory structure preservation
	hasMediaDirectory := false
	for _, zipFile := range zipReader.File {
		if filepath.Dir(zipFile.Name) == "media" {
			hasMediaDirectory = true
			break
		}
	}
	if !hasMediaDirectory {
		t.Error("Expected media/ directory structure to be preserved in ZIP")
	}

	// Test 4: Verify ZIP is valid (can be extracted)
	extractDir := filepath.Join(tmpDir, "extracted")
	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("Failed to create extraction directory: %v", err)
	}

	for _, zipFile := range zipReader.File {
		extractPath := filepath.Join(extractDir, zipFile.Name)

		// Handle directory entries
		if zipFile.FileInfo().IsDir() {
			if err := os.MkdirAll(extractPath, 0755); err != nil {
				t.Fatalf("Failed to create directory: %v", err)
			}
			continue
		}

		// Handle files
		if err := os.MkdirAll(filepath.Dir(extractPath), 0755); err != nil {
			t.Fatalf("Failed to create extraction subdirectory: %v", err)
		}

		rc, err := zipFile.Open()
		if err != nil {
			t.Fatalf("Failed to open file for extraction: %v", err)
		}

		outFile, err := os.Create(extractPath)
		if err != nil {
			rc.Close()
			t.Fatalf("Failed to create extracted file: %v", err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()
		if err != nil {
			t.Fatalf("Failed to extract file: %v", err)
		}
	}

	// Verify extracted files match originals
	for path, expectedContent := range files {
		extractedPath := filepath.Join(extractDir, path)
		actualContent, err := os.ReadFile(extractedPath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", path, err)
			continue
		}

		if !bytes.Equal(actualContent, expectedContent) {
			t.Errorf("Extracted file %s does not match original", path)
		}
	}

	t.Logf("✓ ZIP streaming integrity test passed")
	t.Logf("✓ Verified %d files in archive", len(zipReader.File))
	t.Logf("✓ ZIP size: %d bytes", len(zipData))
}

// TestZIPStreamingWithSymlinks verifies handling of symbolic links
func TestZIPStreamingWithSymlinks(t *testing.T) {
	tmpDir := t.TempDir()
	exportDir := filepath.Join(tmpDir, "export_with_symlinks")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}

	// Create a regular file
	targetFile := filepath.Join(exportDir, "target.txt")
	if err := os.WriteFile(targetFile, []byte("target content"), 0644); err != nil {
		t.Fatalf("Failed to write target file: %v", err)
	}

	// Create a symlink to the file
	symlinkPath := filepath.Join(exportDir, "link.txt")
	if err := os.Symlink(targetFile, symlinkPath); err != nil {
		t.Skipf("Symlink creation not supported: %v", err)
	}

	// Stream to ZIP
	var buf bytes.Buffer
	if err := exporter.StreamDirectoryAsZIP(exportDir, &buf); err != nil {
		t.Fatalf("Failed to stream directory with symlinks: %v", err)
	}

	// Verify ZIP was created successfully
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to read ZIP: %v", err)
	}

	// Should contain at least the target file
	if len(zipReader.File) == 0 {
		t.Error("ZIP should contain at least the target file")
	}

	t.Log("✓ Symlink handling test passed")
}

// TestZIPStreamingCompression verifies compression is working
func TestZIPStreamingCompression(t *testing.T) {
	tmpDir := t.TempDir()
	exportDir := filepath.Join(tmpDir, "compressible_export")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}

	// Create highly compressible content (repeated text)
	compressibleContent := bytes.Repeat([]byte("This is highly compressible text. "), 1000)
	if err := os.WriteFile(filepath.Join(exportDir, "posts.json"), compressibleContent, 0644); err != nil {
		t.Fatalf("Failed to write compressible file: %v", err)
	}

	// Stream to ZIP
	var buf bytes.Buffer
	if err := exporter.StreamDirectoryAsZIP(exportDir, &buf); err != nil {
		t.Fatalf("Failed to stream directory: %v", err)
	}

	// Verify compression worked
	zipSize := buf.Len()
	originalSize := len(compressibleContent)
	compressionRatio := float64(originalSize) / float64(zipSize)

	if compressionRatio < 2.0 {
		t.Errorf("Expected significant compression (ratio > 2.0), got %.2f", compressionRatio)
	}

	t.Logf("✓ Compression test passed")
	t.Logf("✓ Original size: %d bytes, ZIP size: %d bytes, ratio: %.2fx",
		originalSize, zipSize, compressionRatio)
}
