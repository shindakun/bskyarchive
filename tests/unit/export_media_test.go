package unit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/shindakun/bskyarchive/internal/exporter"
)

// TestCopyMediaFile tests single file copy functionality
func TestCopyMediaFile(t *testing.T) {
	// Create temporary directories for source and destination
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create test file with content
	srcFile := filepath.Join(srcDir, "test_media.jpg")
	testContent := []byte("This is test media file content")
	if err := os.WriteFile(srcFile, testContent, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Copy the file
	dstFile := filepath.Join(dstDir, "media", "test_media.jpg")
	err := exporter.CopyMediaFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("CopyMediaFile failed: %v", err)
	}

	// Verify destination file exists
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Fatalf("Destination file was not created: %s", dstFile)
	}

	// Verify file content matches
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != string(testContent) {
		t.Errorf("File content mismatch: got %s, want %s", string(dstContent), string(testContent))
	}
}

// TestCopyMediaFileCreatesDirectory tests that destination directory is created
func TestCopyMediaFileCreatesDirectory(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create test file
	srcFile := filepath.Join(srcDir, "test.jpg")
	if err := os.WriteFile(srcFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Copy to nested directory that doesn't exist yet
	dstFile := filepath.Join(dstDir, "media", "nested", "subdir", "test.jpg")
	err := exporter.CopyMediaFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("CopyMediaFile failed: %v", err)
	}

	// Verify nested directory was created
	dstDirPath := filepath.Dir(dstFile)
	if _, err := os.Stat(dstDirPath); os.IsNotExist(err) {
		t.Fatalf("Destination directory was not created: %s", dstDirPath)
	}

	// Verify file exists
	if _, err := os.Stat(dstFile); os.IsNotExist(err) {
		t.Fatalf("Destination file was not created: %s", dstFile)
	}
}

// TestCopyMediaFileInvalidSource tests error handling for missing source file
func TestCopyMediaFileInvalidSource(t *testing.T) {
	dstDir := t.TempDir()

	// Try to copy non-existent file
	srcFile := "/nonexistent/path/media.jpg"
	dstFile := filepath.Join(dstDir, "media.jpg")

	err := exporter.CopyMediaFile(srcFile, dstFile)
	if err == nil {
		t.Fatal("Expected error for non-existent source file, got nil")
	}
}

// TestCopyMediaFiles tests batch file copying with progress tracking
func TestCopyMediaFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create multiple test media files
	testFiles := map[string][]byte{
		"image1.jpg": []byte("Image 1 content"),
		"image2.png": []byte("Image 2 content"),
		"video.mp4":  []byte("Video content"),
	}

	mediaMap := make(map[string]string)
	for filename, content := range testFiles {
		srcPath := filepath.Join(srcDir, filename)
		if err := os.WriteFile(srcPath, content, 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
		mediaMap[srcPath] = filepath.Join(dstDir, "media", filename)
	}

	// Create progress channel
	progressChan := make(chan int, 10)

	// Copy files
	copiedCount, err := exporter.CopyMediaFiles(mediaMap, progressChan)
	if err != nil {
		t.Fatalf("CopyMediaFiles failed: %v", err)
	}
	close(progressChan)

	// Verify all files were copied
	if copiedCount != len(testFiles) {
		t.Errorf("Expected %d files copied, got %d", len(testFiles), copiedCount)
	}

	// Verify each file exists and has correct content
	for srcPath, dstPath := range mediaMap {
		// Check file exists
		if _, err := os.Stat(dstPath); os.IsNotExist(err) {
			t.Errorf("Destination file was not created: %s", dstPath)
			continue
		}

		// Read and compare content
		srcContent, _ := os.ReadFile(srcPath)
		dstContent, err := os.ReadFile(dstPath)
		if err != nil {
			t.Errorf("Failed to read destination file %s: %v", dstPath, err)
			continue
		}

		if string(dstContent) != string(srcContent) {
			t.Errorf("Content mismatch for %s", dstPath)
		}
	}

	// Verify progress updates were sent
	progressUpdates := 0
	for range progressChan {
		progressUpdates++
	}

	if progressUpdates == 0 {
		t.Error("Expected progress updates, got none")
	}
}

// TestCopyMediaFilesWithMissingFiles tests that missing source files don't fail entire operation
func TestCopyMediaFilesWithMissingFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create map with mix of existing and non-existing files
	mediaMap := map[string]string{
		filepath.Join(srcDir, "exists.jpg"):     filepath.Join(dstDir, "exists.jpg"),
		"/nonexistent/missing.jpg":              filepath.Join(dstDir, "missing.jpg"),
		filepath.Join(srcDir, "also_exists.png"): filepath.Join(dstDir, "also_exists.png"),
	}

	// Create the existing files
	if err := os.WriteFile(filepath.Join(srcDir, "exists.jpg"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "also_exists.png"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Copy files (should not fail even with missing file)
	copiedCount, err := exporter.CopyMediaFiles(mediaMap, nil)
	if err != nil {
		t.Fatalf("CopyMediaFiles failed: %v", err)
	}

	// Should have copied 2 files (skipped the missing one)
	if copiedCount != 2 {
		t.Errorf("Expected 2 files copied (skipping missing), got %d", copiedCount)
	}

	// Verify existing files were copied
	if _, err := os.Stat(filepath.Join(dstDir, "exists.jpg")); os.IsNotExist(err) {
		t.Error("Expected exists.jpg to be copied")
	}
	if _, err := os.Stat(filepath.Join(dstDir, "also_exists.png")); os.IsNotExist(err) {
		t.Error("Expected also_exists.png to be copied")
	}

	// Verify missing file was not created
	if _, err := os.Stat(filepath.Join(dstDir, "missing.jpg")); err == nil {
		t.Error("Missing file should not have been created")
	}
}

// TestCopyMediaFilesNoProgress tests copying without progress channel
func TestCopyMediaFilesNoProgress(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create test file
	srcFile := filepath.Join(srcDir, "test.jpg")
	if err := os.WriteFile(srcFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	mediaMap := map[string]string{
		srcFile: filepath.Join(dstDir, "test.jpg"),
	}

	// Copy without progress channel (nil)
	copiedCount, err := exporter.CopyMediaFiles(mediaMap, nil)
	if err != nil {
		t.Fatalf("CopyMediaFiles failed: %v", err)
	}

	if copiedCount != 1 {
		t.Errorf("Expected 1 file copied, got %d", copiedCount)
	}

	// Verify file was copied
	if _, err := os.Stat(filepath.Join(dstDir, "test.jpg")); os.IsNotExist(err) {
		t.Error("File was not copied")
	}
}

// TestCopyMediaFileLargeFile tests copying a larger file
func TestCopyMediaFileLargeFile(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a larger test file (1MB)
	srcFile := filepath.Join(srcDir, "large.bin")
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	if err := os.WriteFile(srcFile, largeContent, 0644); err != nil {
		t.Fatalf("Failed to create large test file: %v", err)
	}

	// Copy the file
	dstFile := filepath.Join(dstDir, "large.bin")
	err := exporter.CopyMediaFile(srcFile, dstFile)
	if err != nil {
		t.Fatalf("CopyMediaFile failed: %v", err)
	}

	// Verify file size matches
	srcInfo, _ := os.Stat(srcFile)
	dstInfo, err := os.Stat(dstFile)
	if err != nil {
		t.Fatalf("Destination file not found: %v", err)
	}

	if dstInfo.Size() != srcInfo.Size() {
		t.Errorf("File size mismatch: got %d, want %d", dstInfo.Size(), srcInfo.Size())
	}

	// Verify content matches
	dstContent, err := os.ReadFile(dstFile)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if len(dstContent) != len(largeContent) {
		t.Errorf("Content length mismatch: got %d, want %d", len(dstContent), len(largeContent))
	}

	// Spot check some bytes
	for i := 0; i < 100; i++ {
		idx := i * 1000
		if idx < len(dstContent) && dstContent[idx] != largeContent[idx] {
			t.Errorf("Content mismatch at byte %d: got %d, want %d", idx, dstContent[idx], largeContent[idx])
		}
	}
}
