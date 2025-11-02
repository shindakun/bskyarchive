package exporter

import (
	"os"
	"strings"
	"testing"
)

// TestCreateExportDirectory_PreservesDotSlash verifies that ./ prefix is preserved
// This ensures paths pass validation in models.ExportRecord
func TestCreateExportDirectory_PreservesDotSlash(t *testing.T) {
	// Create temporary directory for testing
	tmpDir := t.TempDir()
	testBaseDir := "./test_exports"

	t.Run("preserves ./ prefix for relative paths", func(t *testing.T) {
		exportDir, err := CreateExportDirectory(testBaseDir, "did:plc:test")
		if err != nil {
			t.Fatalf("CreateExportDirectory() error = %v", err)
		}

		if !strings.HasPrefix(exportDir, "./") {
			t.Errorf("Expected path to start with ./, got: %s", exportDir)
		}

		if !strings.HasPrefix(exportDir, testBaseDir+"/") {
			t.Errorf("Expected path to start with %s/, got: %s", testBaseDir, exportDir)
		}

		// Cleanup
		os.RemoveAll("./test_exports")
	})

	t.Run("handles path without ./ prefix", func(t *testing.T) {
		exportDir, err := CreateExportDirectory("exports_no_dot", "did:plc:test")
		if err != nil {
			t.Fatalf("CreateExportDirectory() error = %v", err)
		}

		// Should NOT add ./ prefix if baseDir didn't have it
		if strings.HasPrefix(exportDir, "./") {
			t.Errorf("Expected path to NOT start with ./ for baseDir without prefix, got: %s", exportDir)
		}

		// Cleanup
		os.RemoveAll("exports_no_dot")
	})

	t.Run("handles absolute paths correctly", func(t *testing.T) {
		exportDir, err := CreateExportDirectory(tmpDir, "did:plc:test")
		if err != nil {
			t.Fatalf("CreateExportDirectory() error = %v", err)
		}

		// Absolute paths should NOT get ./ prefix
		if strings.HasPrefix(exportDir, "./") {
			t.Errorf("Expected absolute path to NOT start with ./, got: %s", exportDir)
		}

		if !strings.HasPrefix(exportDir, tmpDir) {
			t.Errorf("Expected path to start with %s, got: %s", tmpDir, exportDir)
		}
	})

	t.Log("✓ Path prefix preservation tests passed")
}
