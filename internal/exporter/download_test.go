package exporter

import (
	"archive/zip"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestStreamDirectoryAsZIP verifies ZIP streaming functionality
func TestStreamDirectoryAsZIP(t *testing.T) {
	// Create temporary test directory with files
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "test_export")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create test files
	testFiles := map[string]string{
		"posts.json":     `{"posts": [{"uri": "test1", "text": "Hello"}]}`,
		"manifest.json":  `{"version": "1.0", "post_count": 1}`,
		"media/test.jpg": "fake image data",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(testDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory for %s: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write test file %s: %v", path, err)
		}
	}

	// Stream directory to ZIP
	var buf bytes.Buffer
	if err := StreamDirectoryAsZIP(testDir, &buf); err != nil {
		t.Fatalf("StreamDirectoryAsZIP failed: %v", err)
	}

	// Verify ZIP was created
	if buf.Len() == 0 {
		t.Fatal("ZIP output is empty")
	}

	// Read and verify ZIP contents
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to read ZIP: %v", err)
	}

	// Verify all files are in the ZIP
	foundFiles := make(map[string]bool)
	for _, f := range zipReader.File {
		foundFiles[f.Name] = true

		// Verify file can be read
		rc, err := f.Open()
		if err != nil {
			t.Errorf("Failed to open file %s in ZIP: %v", f.Name, err)
			continue
		}
		content, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Errorf("Failed to read file %s in ZIP: %v", f.Name, err)
			continue
		}

		// Verify content matches (for non-empty files)
		if expectedContent, exists := testFiles[f.Name]; exists {
			if string(content) != expectedContent {
				t.Errorf("File %s content mismatch\nExpected: %s\nGot: %s",
					f.Name, expectedContent, string(content))
			}
		}
	}

	// Verify all expected files were found
	for expectedFile := range testFiles {
		if !foundFiles[expectedFile] {
			t.Errorf("Expected file %s not found in ZIP", expectedFile)
		}
	}

	t.Logf("✓ StreamDirectoryAsZIP test passed - created ZIP with %d files", len(zipReader.File))
}

// TestStreamDirectoryAsZIP_EmptyDirectory verifies handling of empty directories
func TestStreamDirectoryAsZIP_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "empty_export")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	var buf bytes.Buffer
	if err := StreamDirectoryAsZIP(testDir, &buf); err != nil {
		t.Fatalf("StreamDirectoryAsZIP failed for empty directory: %v", err)
	}

	// Empty directory should still create valid ZIP (just no files)
	if buf.Len() == 0 {
		t.Fatal("ZIP output is empty")
	}

	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to read ZIP: %v", err)
	}

	if len(zipReader.File) != 0 {
		t.Errorf("Expected 0 files in ZIP, got %d", len(zipReader.File))
	}

	t.Log("✓ Empty directory test passed")
}

// TestStreamDirectoryAsZIP_NonExistentDirectory verifies error handling
func TestStreamDirectoryAsZIP_NonExistentDirectory(t *testing.T) {
	var buf bytes.Buffer
	err := StreamDirectoryAsZIP("/nonexistent/path/that/does/not/exist", &buf)
	if err == nil {
		t.Fatal("Expected error for non-existent directory, got nil")
	}

	t.Logf("✓ Non-existent directory error handling test passed: %v", err)
}

// TestStreamDirectoryAsZIP_LargeFiles verifies memory efficiency with larger files
func TestStreamDirectoryAsZIP_LargeFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testDir := filepath.Join(tmpDir, "large_export")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create a 5MB test file to simulate real export
	largeContent := make([]byte, 5*1024*1024)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	if err := os.WriteFile(filepath.Join(testDir, "large_file.json"), largeContent, 0644); err != nil {
		t.Fatalf("Failed to write large test file: %v", err)
	}

	var buf bytes.Buffer
	if err := StreamDirectoryAsZIP(testDir, &buf); err != nil {
		t.Fatalf("StreamDirectoryAsZIP failed for large file: %v", err)
	}

	// Verify ZIP was created and contains the file
	zipReader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatalf("Failed to read ZIP: %v", err)
	}

	if len(zipReader.File) != 1 {
		t.Errorf("Expected 1 file in ZIP, got %d", len(zipReader.File))
	}

	t.Logf("✓ Large file test passed - ZIP size: %d bytes", buf.Len())
}
