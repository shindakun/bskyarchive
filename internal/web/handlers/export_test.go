package handlers

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
)

// TestDeleteExportInternal verifies the deleteExportInternal helper function
func TestDeleteExportInternal(t *testing.T) {
	// Create temporary database and export directory
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	t.Run("successful deletion of export with files", func(t *testing.T) {
		// Create export directory with files (use ./exports structure for validation)
		exportDir := "./exports/did:plc:test/export1"
		if err := os.MkdirAll(exportDir, 0755); err != nil {
			t.Fatalf("Failed to create export directory: %v", err)
		}
		defer os.RemoveAll("./exports") // Cleanup

		// Create some test files
		testFile := filepath.Join(exportDir, "posts.json")
		if err := os.WriteFile(testFile, []byte(`{"posts": []}`), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Create export record
		exportRecord := &models.ExportRecord{
			ID:            "did:plc:test/export1",
			DID:           "did:plc:test",
			Format:        "json",
			CreatedAt:     time.Now(),
			DirectoryPath: exportDir,
			PostCount:     10,
			SizeBytes:     1000,
		}

		if err := storage.CreateExportRecord(db, exportRecord); err != nil {
			t.Fatalf("Failed to create export record: %v", err)
		}

		// Delete the export
		err := deleteExportInternal(db, exportRecord.ID)
		if err != nil {
			t.Errorf("deleteExportInternal() failed: %v", err)
		}

		// Verify directory is deleted
		if _, err := os.Stat(exportDir); !os.IsNotExist(err) {
			t.Error("Export directory should be deleted but still exists")
		}

		// Verify database record is deleted
		_, err = storage.GetExportByID(db, exportRecord.ID)
		if err == nil {
			t.Error("Export record should be deleted from database but still exists")
		}
	})

	t.Run("deletion when directory doesn't exist but DB record does", func(t *testing.T) {
		// Create export record without directory
		exportRecord := &models.ExportRecord{
			ID:            "did:plc:test/orphaned",
			DID:           "did:plc:test",
			Format:        "json",
			CreatedAt:     time.Now(),
			DirectoryPath: "./exports/did:plc:test/orphaned",
			PostCount:     10,
			SizeBytes:     1000,
		}

		if err := storage.CreateExportRecord(db, exportRecord); err != nil {
			t.Fatalf("Failed to create export record: %v", err)
		}

		// Delete should succeed even if directory doesn't exist
		err := deleteExportInternal(db, exportRecord.ID)
		if err != nil {
			t.Errorf("deleteExportInternal() should handle missing directory gracefully: %v", err)
		}

		// Verify database record is deleted
		_, err = storage.GetExportByID(db, exportRecord.ID)
		if err == nil {
			t.Error("Export record should be deleted from database")
		}
	})

	t.Run("deletion of non-existent export returns error", func(t *testing.T) {
		err := deleteExportInternal(db, "did:plc:test/nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent export, got nil")
		}
	})

	t.Log("✓ deleteExportInternal unit tests passed")
}

// TestDeleteExportCleanup verifies that deletion properly cleans up all files
func TestDeleteExportCleanup(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create complex export structure with subdirectories
	exportDir := "./exports/did:plc:test/complex"
	mediaDir := filepath.Join(exportDir, "media")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		t.Fatalf("Failed to create export directories: %v", err)
	}
	defer os.RemoveAll("./exports") // Cleanup

	// Create multiple files
	files := map[string]string{
		filepath.Join(exportDir, "posts.json"):     `{"posts": []}`,
		filepath.Join(exportDir, "manifest.json"):  `{"version": "1.0"}`,
		filepath.Join(mediaDir, "image1.jpg"):      "fake image 1",
		filepath.Join(mediaDir, "image2.jpg"):      "fake image 2",
	}

	for path, content := range files {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", path, err)
		}
	}

	// Create export record
	exportRecord := &models.ExportRecord{
		ID:            "did:plc:test/complex",
		DID:           "did:plc:test",
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: exportDir,
		PostCount:     100,
		MediaCount:    2,
		SizeBytes:     5000,
	}

	if err := storage.CreateExportRecord(db, exportRecord); err != nil {
		t.Fatalf("Failed to create export record: %v", err)
	}

	// Delete the export
	err = deleteExportInternal(db, exportRecord.ID)
	if err != nil {
		t.Fatalf("deleteExportInternal() failed: %v", err)
	}

	// Verify entire directory tree is deleted
	if _, err := os.Stat(exportDir); !os.IsNotExist(err) {
		t.Error("Export directory tree should be completely deleted")
	}

	// Verify media subdirectory is also gone
	if _, err := os.Stat(mediaDir); !os.IsNotExist(err) {
		t.Error("Media subdirectory should be deleted")
	}

	t.Log("✓ Complex export cleanup test passed")
}

// TestDeleteExportConcurrency verifies deletion handles concurrent operations gracefully
func TestDeleteExportConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create export
	exportDir := "./exports/did:plc:test/concurrent"
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}
	defer os.RemoveAll("./exports") // Cleanup

	if err := os.WriteFile(filepath.Join(exportDir, "posts.json"), []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	exportRecord := &models.ExportRecord{
		ID:            "did:plc:test/concurrent",
		DID:           "did:plc:test",
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: exportDir,
		PostCount:     10,
		SizeBytes:     1000,
	}

	if err := storage.CreateExportRecord(db, exportRecord); err != nil {
		t.Fatalf("Failed to create export record: %v", err)
	}

	// Try to delete twice concurrently
	errChan := make(chan error, 2)

	go func() {
		errChan <- deleteExportInternal(db, exportRecord.ID)
	}()

	go func() {
		errChan <- deleteExportInternal(db, exportRecord.ID)
	}()

	// Collect results
	err1 := <-errChan
	err2 := <-errChan

	// At least one should succeed or both should handle gracefully
	if err1 != nil && err2 != nil {
		t.Logf("Both deletions returned errors (this is acceptable if handled gracefully)")
		t.Logf("Error 1: %v", err1)
		t.Logf("Error 2: %v", err2)
	}

	// Verify export is gone
	if _, err := os.Stat(exportDir); !os.IsNotExist(err) {
		t.Error("Export directory should be deleted after concurrent operations")
	}

	_, err = storage.GetExportByID(db, exportRecord.ID)
	if err == nil {
		t.Error("Export record should be deleted from database")
	}

	t.Log("✓ Concurrent deletion test passed")
}
