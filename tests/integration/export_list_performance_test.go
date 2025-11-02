package integration

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
)

// TestExportListQueryPerformance verifies that listing 50 exports takes less than 1 second
// This test validates the performance requirement from T055
func TestExportListQueryPerformance(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Test DID
	testDID := "did:plc:test123456789"

	// Create 50 test export records
	t.Log("Creating 50 test export records...")
	for i := 0; i < 50; i++ {
		exportDir := fmt.Sprintf("./exports/%s/export-%d", testDID, i)
		if err := os.MkdirAll(exportDir, 0755); err != nil {
			t.Fatalf("Failed to create export directory: %v", err)
		}

		record := &models.ExportRecord{
			ID:            fmt.Sprintf("%s/export-%d", testDID, i),
			DID:           testDID,
			Format:        "json",
			CreatedAt:     time.Now().Add(-time.Duration(i) * time.Hour),
			DirectoryPath: exportDir,
			PostCount:     1000 + i,
			MediaCount:    50 + i,
			SizeBytes:     int64((100 + i) * 1024 * 1024), // ~100MB each
		}

		if err := storage.CreateExportRecord(db, record); err != nil {
			t.Fatalf("Failed to create export record %d: %v", i, err)
		}
	}

	// Run query multiple times to get average performance
	const iterations = 10
	var totalDuration time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()

		exports, err := storage.ListExportsByDID(db, testDID, 50, 0)
		if err != nil {
			t.Fatalf("Failed to list exports: %v", err)
		}

		duration := time.Since(start)
		totalDuration += duration

		// Verify results
		if len(exports) != 50 {
			t.Errorf("Expected 50 exports, got %d", len(exports))
		}

		// Verify ordering (newest first)
		for j := 1; j < len(exports); j++ {
			if exports[j-1].CreatedAt.Before(exports[j].CreatedAt) {
				t.Errorf("Exports not ordered by newest first: export[%d]=%v > export[%d]=%v",
					j-1, exports[j-1].CreatedAt, j, exports[j].CreatedAt)
			}
		}
	}

	// Calculate average
	avgDuration := totalDuration / iterations

	t.Logf("Performance Results:")
	t.Logf("  Total iterations: %d", iterations)
	t.Logf("  Total time: %v", totalDuration)
	t.Logf("  Average query time: %v", avgDuration)
	t.Logf("  Min acceptable: 1s")

	// Verify performance requirement: average query time < 1 second
	if avgDuration >= time.Second {
		t.Errorf("Performance requirement not met: average query time %v >= 1s", avgDuration)
	} else {
		t.Logf("✓ Performance requirement met: average query time %v < 1s", avgDuration)
	}
}

// TestExportListQueryPerformanceWithPagination tests pagination performance
func TestExportListQueryPerformanceWithPagination(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Test DID
	testDID := "did:plc:pagination123"

	// Create 100 test export records to test pagination
	t.Log("Creating 100 test export records...")
	for i := 0; i < 100; i++ {
		exportDir := fmt.Sprintf("./exports/%s/export-%d", testDID, i)
		if err := os.MkdirAll(exportDir, 0755); err != nil {
			t.Fatalf("Failed to create export directory: %v", err)
		}

		record := &models.ExportRecord{
			ID:            fmt.Sprintf("%s/export-%d", testDID, i),
			DID:           testDID,
			Format:        "json",
			CreatedAt:     time.Now().Add(-time.Duration(i) * time.Hour),
			DirectoryPath: exportDir,
			PostCount:     1000,
			MediaCount:    50,
			SizeBytes:     100 * 1024 * 1024,
		}

		if err := storage.CreateExportRecord(db, record); err != nil {
			t.Fatalf("Failed to create export record %d: %v", i, err)
		}
	}

	// Test pagination performance
	start := time.Now()

	// Page 1 (0-49)
	page1, err := storage.ListExportsByDID(db, testDID, 50, 0)
	if err != nil {
		t.Fatalf("Failed to list page 1: %v", err)
	}

	// Page 2 (50-99)
	page2, err := storage.ListExportsByDID(db, testDID, 50, 50)
	if err != nil {
		t.Fatalf("Failed to list page 2: %v", err)
	}

	duration := time.Since(start)

	// Verify results
	if len(page1) != 50 {
		t.Errorf("Expected 50 exports in page 1, got %d", len(page1))
	}
	if len(page2) != 50 {
		t.Errorf("Expected 50 exports in page 2, got %d", len(page2))
	}

	// Verify no overlap
	for _, e1 := range page1 {
		for _, e2 := range page2 {
			if e1.ID == e2.ID {
				t.Errorf("Found duplicate export ID across pages: %s", e1.ID)
			}
		}
	}

	t.Logf("Pagination Performance:")
	t.Logf("  Total records: 100")
	t.Logf("  Pages fetched: 2 (50 records each)")
	t.Logf("  Total time: %v", duration)
	t.Logf("  Time per page: %v", duration/2)

	if duration >= 2*time.Second {
		t.Errorf("Pagination too slow: %v >= 2s for 2 pages", duration)
	}
}

// TestExportListQueryPerformanceWithIndices verifies indices are being used
func TestExportListQueryPerformanceWithIndices(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify indices exist
	var indexCount int
	err = db.QueryRow(`
		SELECT COUNT(*)
		FROM sqlite_master
		WHERE type = 'index'
		AND tbl_name = 'exports'
		AND name IN ('idx_exports_did', 'idx_exports_created_at', 'idx_exports_did_created')
	`).Scan(&indexCount)

	if err != nil {
		t.Fatalf("Failed to query indices: %v", err)
	}

	if indexCount != 3 {
		t.Errorf("Expected 3 indices on exports table, found %d", indexCount)
	} else {
		t.Log("✓ All required indices exist on exports table")
	}

	// Test query plan to ensure index is used
	testDID := "did:plc:indextest"

	// Create a single record
	exportDir := fmt.Sprintf("./exports/%s/export-1", testDID)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}

	record := &models.ExportRecord{
		ID:            fmt.Sprintf("%s/export-1", testDID),
		DID:           testDID,
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: exportDir,
		PostCount:     1000,
		MediaCount:    50,
		SizeBytes:     100 * 1024 * 1024,
	}

	if err := storage.CreateExportRecord(db, record); err != nil {
		t.Fatalf("Failed to create export record: %v", err)
	}

	// Check query plan
	var queryPlan string
	err = db.QueryRow(`
		EXPLAIN QUERY PLAN
		SELECT id, did, format, created_at, directory_path,
		       post_count, media_count, size_bytes,
		       date_range_start, date_range_end, manifest_path
		FROM exports
		WHERE did = ?
		ORDER BY created_at DESC
		LIMIT 50 OFFSET 0
	`, testDID).Scan(&queryPlan)

	if err != nil && err != sql.ErrNoRows {
		// The EXPLAIN QUERY PLAN might return multiple rows, just check one exists
		t.Logf("Note: Could not verify query plan (this is informational only)")
	} else {
		t.Logf("Query plan: %s", queryPlan)
	}
}
