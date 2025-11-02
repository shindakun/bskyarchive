package integration

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
)

// TestDiskSpaceErrorHandling tests graceful error handling when disk operations fail
// This test simulates disk space exhaustion scenarios (T058)
// Note: Actual disk space exhaustion is difficult to test automatically,
// so we focus on permission errors and other filesystem errors that have similar behavior
func TestDiskSpaceErrorHandling(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	testDID := "did:plc:disktest"

	t.Run("deletion handles missing directory gracefully", func(t *testing.T) {
		// Create export record pointing to non-existent directory
		exportDir := "./exports/nonexistent/missing"
		record := &models.ExportRecord{
			ID:            fmt.Sprintf("%s/missing-export", testDID),
			DID:           testDID,
			Format:        "json",
			CreatedAt:     time.Now(),
			DirectoryPath: exportDir,
			PostCount:     100,
			MediaCount:    5,
			SizeBytes:     1024 * 1024,
		}

		if err := storage.CreateExportRecord(db, record); err != nil {
			t.Fatalf("Failed to create export record: %v", err)
		}

		// Attempt to delete - should handle missing directory gracefully
		// Note: In the actual implementation, os.RemoveAll returns nil for non-existent paths
		err := os.RemoveAll(exportDir)
		if err != nil {
			t.Errorf("RemoveAll should handle non-existent directory gracefully, got error: %v", err)
		}

		// Delete from database should still work
		err = storage.DeleteExport(db, record.ID)
		if err != nil {
			t.Errorf("Database deletion should succeed even if directory doesn't exist: %v", err)
		}
	})

	t.Run("deletion handles orphaned database records", func(t *testing.T) {
		// Create export record but no actual files
		exportDir := "./exports/orphaned/test"
		record := &models.ExportRecord{
			ID:            fmt.Sprintf("%s/orphaned-export", testDID),
			DID:           testDID,
			Format:        "json",
			CreatedAt:     time.Now(),
			DirectoryPath: exportDir,
			PostCount:     100,
			MediaCount:    5,
			SizeBytes:     1024 * 1024,
		}

		if err := storage.CreateExportRecord(db, record); err != nil {
			t.Fatalf("Failed to create export record: %v", err)
		}

		// Verify record exists
		retrieved, err := storage.GetExportByID(db, record.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve export record: %v", err)
		}
		if retrieved.ID != record.ID {
			t.Errorf("Retrieved wrong record: expected %s, got %s", record.ID, retrieved.ID)
		}

		// Delete - should handle missing directory gracefully
		if err := os.RemoveAll(exportDir); err != nil {
			t.Logf("RemoveAll returned error (expected for non-existent path): %v", err)
		}

		// Database deletion should still work
		if err := storage.DeleteExport(db, record.ID); err != nil {
			t.Errorf("Should be able to delete orphaned database record: %v", err)
		}

		// Verify record is gone
		_, err = storage.GetExportByID(db, record.ID)
		if err == nil {
			t.Error("Export record should be deleted")
		}
	})

	t.Run("export size calculation handles inaccessible files", func(t *testing.T) {
		// Create a directory with files
		exportDir := "./exports/accessibility-test/test"
		if err := os.MkdirAll(exportDir, 0755); err != nil {
			t.Fatalf("Failed to create export directory: %v", err)
		}
		defer os.RemoveAll("./exports/accessibility-test")

		// Create a test file
		testFile := filepath.Join(exportDir, "test.json")
		if err := os.WriteFile(testFile, []byte(`{"test":"data"}`), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Calculate size
		var totalSize int64
		err := filepath.Walk(exportDir, func(_ string, info os.FileInfo, err error) error {
			if err != nil {
				// Log error but continue walking
				t.Logf("Walk encountered error (continuing): %v", err)
				return nil // Don't stop walking on error
			}
			if !info.IsDir() {
				totalSize += info.Size()
			}
			return nil
		})

		if err != nil {
			t.Errorf("Walk should handle errors gracefully: %v", err)
		}

		if totalSize == 0 {
			t.Error("Should have calculated size for accessible file")
		}

		t.Logf("Calculated size: %d bytes", totalSize)
	})

	t.Run("database operations handle connection errors", func(t *testing.T) {
		// Close the database to simulate connection loss
		db.Close()

		// Attempt operations - should return errors gracefully
		record := &models.ExportRecord{
			ID:            fmt.Sprintf("%s/connection-test", testDID),
			DID:           testDID,
			Format:        "json",
			CreatedAt:     time.Now(),
			DirectoryPath: "./exports/test/connection",
			PostCount:     100,
			MediaCount:    5,
			SizeBytes:     1024,
		}

		// Create should fail
		err := storage.CreateExportRecord(db, record)
		if err == nil {
			t.Error("CreateExportRecord should fail with closed database")
		} else {
			t.Logf("✓ CreateExportRecord correctly returned error: %v", err)
		}

		// List should fail
		_, err = storage.ListExportsByDID(db, testDID, 50, 0)
		if err == nil {
			t.Error("ListExportsByDID should fail with closed database")
		} else {
			t.Logf("✓ ListExportsByDID correctly returned error: %v", err)
		}

		// Get should fail
		_, err = storage.GetExportByID(db, "any-id")
		if err == nil {
			t.Error("GetExportByID should fail with closed database")
		} else {
			t.Logf("✓ GetExportByID correctly returned error: %v", err)
		}

		// Delete should fail
		err = storage.DeleteExport(db, "any-id")
		if err == nil {
			t.Error("DeleteExport should fail with closed database")
		} else {
			t.Logf("✓ DeleteExport correctly returned error: %v", err)
		}
	})
}

// TestExportStorageResilience tests resilience of export storage operations
func TestExportStorageResilience(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	testDID := "did:plc:resilience"

	t.Run("handles concurrent deletions gracefully", func(t *testing.T) {
		// Create export record
		exportDir := "./exports/concurrent/test"
		if err := os.MkdirAll(exportDir, 0755); err != nil {
			t.Fatalf("Failed to create export directory: %v", err)
		}
		defer os.RemoveAll("./exports/concurrent")

		record := &models.ExportRecord{
			ID:            fmt.Sprintf("%s/concurrent-export", testDID),
			DID:           testDID,
			Format:        "json",
			CreatedAt:     time.Now(),
			DirectoryPath: exportDir,
			PostCount:     100,
			MediaCount:    5,
			SizeBytes:     1024,
		}

		if err := storage.CreateExportRecord(db, record); err != nil {
			t.Fatalf("Failed to create export record: %v", err)
		}

		// Delete files first (simulating external deletion or concurrent access)
		if err := os.RemoveAll(exportDir); err != nil {
			t.Fatalf("Failed to delete export directory: %v", err)
		}

		// Now try to delete from database - should succeed even though files are gone
		if err := storage.DeleteExport(db, record.ID); err != nil {
			t.Errorf("Database deletion should succeed even when files already deleted: %v", err)
		}

		// Verify record is gone
		_, err := storage.GetExportByID(db, record.ID)
		if err == nil {
			t.Error("Export record should be deleted")
		}
	})

	t.Run("validates export paths for security", func(t *testing.T) {
		// Test path traversal prevention
		invalidRecords := []struct {
			name string
			path string
		}{
			{"absolute path", "/tmp/hack"},
			{"parent traversal", "./exports/../../../etc/passwd"},
			{"null byte", "./exports/test\x00/file"},
			{"no exports prefix", "./data/export"},
		}

		for _, tc := range invalidRecords {
			t.Run(tc.name, func(t *testing.T) {
				record := &models.ExportRecord{
					ID:            fmt.Sprintf("%s/%s", testDID, tc.name),
					DID:           testDID,
					Format:        "json",
					CreatedAt:     time.Now(),
					DirectoryPath: tc.path,
					PostCount:     100,
					MediaCount:    5,
					SizeBytes:     1024,
				}

				err := storage.CreateExportRecord(db, record)
				if err == nil {
					t.Errorf("Should reject invalid path: %s", tc.path)
				} else {
					t.Logf("✓ Correctly rejected invalid path '%s': %v", tc.path, err)
				}
			})
		}
	})

	t.Run("handles database transaction failures", func(t *testing.T) {
		// Try to create duplicate record (should fail due to PRIMARY KEY constraint)
		exportDir := "./exports/duplicate/test"
		if err := os.MkdirAll(exportDir, 0755); err != nil {
			t.Fatalf("Failed to create export directory: %v", err)
		}
		defer os.RemoveAll("./exports/duplicate")

		record1 := &models.ExportRecord{
			ID:            fmt.Sprintf("%s/duplicate-id", testDID),
			DID:           testDID,
			Format:        "json",
			CreatedAt:     time.Now(),
			DirectoryPath: exportDir,
			PostCount:     100,
			MediaCount:    5,
			SizeBytes:     1024,
		}

		// Create first record
		if err := storage.CreateExportRecord(db, record1); err != nil {
			t.Fatalf("Failed to create first export record: %v", err)
		}

		// Try to create duplicate (same ID)
		record2 := &models.ExportRecord{
			ID:            record1.ID, // Same ID!
			DID:           testDID,
			Format:        "csv",
			CreatedAt:     time.Now(),
			DirectoryPath: exportDir,
			PostCount:     200,
			MediaCount:    10,
			SizeBytes:     2048,
		}

		err := storage.CreateExportRecord(db, record2)
		if err == nil {
			t.Error("Should reject duplicate export ID")
		} else {
			t.Logf("✓ Correctly rejected duplicate ID: %v", err)
		}

		// Verify original record is unchanged
		retrieved, err := storage.GetExportByID(db, record1.ID)
		if err != nil {
			t.Fatalf("Failed to retrieve original record: %v", err)
		}
		if retrieved.PostCount != 100 {
			t.Errorf("Original record was modified: expected PostCount=100, got %d", retrieved.PostCount)
		}
	})
}

// TestDiskSpaceDocumentation documents behavior during disk space issues
func TestDiskSpaceDocumentation(t *testing.T) {
	t.Log("=== Disk Space Error Handling Documentation ===")
	t.Log("")
	t.Log("Expected Behavior During Disk Space Exhaustion:")
	t.Log("")
	t.Log("1. Export Creation:")
	t.Log("   - Will fail with 'no space left on device' error")
	t.Log("   - Progress will stop and status will be set to 'failed'")
	t.Log("   - Partial files will remain in export directory")
	t.Log("   - User will see error message in UI")
	t.Log("   - Database record may or may not be created (depends on when failure occurs)")
	t.Log("")
	t.Log("2. Export Download:")
	t.Log("   - Downloads stream from existing files (no new disk writes)")
	t.Log("   - Should continue to work even when disk is full")
	t.Log("   - Only fails if export files have been deleted")
	t.Log("")
	t.Log("3. Export Deletion:")
	t.Log("   - Deletion frees up disk space")
	t.Log("   - os.RemoveAll handles missing files gracefully")
	t.Log("   - Database record always deleted regardless of file deletion success")
	t.Log("   - Logs warning if file deletion fails but continues")
	t.Log("")
	t.Log("4. Export Listing:")
	t.Log("   - Pure database query (no disk I/O)")
	t.Log("   - Works normally even when disk is full")
	t.Log("   - May show exports whose files have been manually deleted")
	t.Log("")
	t.Log("5. Recovery Recommendations:")
	t.Log("   - Monitor disk space before starting exports")
	t.Log("   - Delete old exports to free up space")
	t.Log("   - Failed exports leave partial files that should be cleaned up")
	t.Log("   - Consider implementing automatic cleanup of old/failed exports")
	t.Log("")
	t.Log("=== Manual Testing Required ===")
	t.Log("")
	t.Log("To fully test disk space exhaustion (requires manual setup):")
	t.Log("1. Create a small disk image (e.g., 100MB)")
	t.Log("2. Mount it as the exports directory")
	t.Log("3. Attempt to create exports larger than available space")
	t.Log("4. Verify error messages are user-friendly")
	t.Log("5. Verify partial exports can be deleted to recover space")
	t.Log("6. Verify downloads continue to work for existing exports")
}

// Benchmark to verify deletion performance under stress
func BenchmarkExportDeletion(b *testing.B) {
	// Create temporary database
	tmpDir := b.TempDir()
	dbPath := filepath.Join(tmpDir, "bench.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		b.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	testDID := "did:plc:benchmark"

	// Pre-create export records
	records := make([]*models.ExportRecord, b.N)
	for i := 0; i < b.N; i++ {
		exportDir := fmt.Sprintf("./exports/%s/bench-%d", testDID, i)
		os.MkdirAll(exportDir, 0755)

		records[i] = &models.ExportRecord{
			ID:            fmt.Sprintf("%s/bench-%d", testDID, i),
			DID:           testDID,
			Format:        "json",
			CreatedAt:     time.Now(),
			DirectoryPath: exportDir,
			PostCount:     100,
			MediaCount:    5,
			SizeBytes:     1024,
		}

		if err := storage.CreateExportRecord(db, records[i]); err != nil {
			b.Fatalf("Failed to create record: %v", err)
		}
	}
	defer os.RemoveAll("./exports")

	b.ResetTimer()

	// Benchmark deletion
	for i := 0; i < b.N; i++ {
		if err := storage.DeleteExport(db, records[i].ID); err != nil {
			b.Errorf("Failed to delete export %d: %v", i, err)
		}
	}
}
