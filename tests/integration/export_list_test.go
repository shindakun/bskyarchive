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

// TestExportListDIDIsolation verifies that users can only see their own exports
func TestExportListDIDIsolation(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create exports for three different users
	user1 := "did:plc:user1"
	user2 := "did:plc:user2"
	user3 := "did:plc:user3"

	// User 1: 3 exports
	user1Exports := []models.ExportRecord{
		{
			ID:            fmt.Sprintf("%s/export1", user1),
			DID:           user1,
			Format:        "json",
			CreatedAt:     time.Now().Add(-3 * time.Hour),
			DirectoryPath: "./exports/user1/export1",
			PostCount:     100,
			MediaCount:    50,
			SizeBytes:     1024000,
		},
		{
			ID:            fmt.Sprintf("%s/export2", user1),
			DID:           user1,
			Format:        "csv",
			CreatedAt:     time.Now().Add(-2 * time.Hour),
			DirectoryPath: "./exports/user1/export2",
			PostCount:     200,
			MediaCount:    75,
			SizeBytes:     2048000,
		},
		{
			ID:            fmt.Sprintf("%s/export3", user1),
			DID:           user1,
			Format:        "json",
			CreatedAt:     time.Now().Add(-1 * time.Hour),
			DirectoryPath: "./exports/user1/export3",
			PostCount:     150,
			MediaCount:    60,
			SizeBytes:     1536000,
		},
	}

	// User 2: 2 exports
	user2Exports := []models.ExportRecord{
		{
			ID:            fmt.Sprintf("%s/export1", user2),
			DID:           user2,
			Format:        "json",
			CreatedAt:     time.Now().Add(-4 * time.Hour),
			DirectoryPath: "./exports/user2/export1",
			PostCount:     50,
			MediaCount:    25,
			SizeBytes:     512000,
		},
		{
			ID:            fmt.Sprintf("%s/export2", user2),
			DID:           user2,
			Format:        "json",
			CreatedAt:     time.Now().Add(-1 * time.Minute),
			DirectoryPath: "./exports/user2/export2",
			PostCount:     75,
			MediaCount:    30,
			SizeBytes:     768000,
		},
	}

	// User 3: 1 export
	user3Exports := []models.ExportRecord{
		{
			ID:            fmt.Sprintf("%s/export1", user3),
			DID:           user3,
			Format:        "csv",
			CreatedAt:     time.Now().Add(-5 * time.Hour),
			DirectoryPath: "./exports/user3/export1",
			PostCount:     300,
			MediaCount:    150,
			SizeBytes:     3072000,
		},
	}

	// Insert all exports
	allExports := append(user1Exports, append(user2Exports, user3Exports...)...)
	for _, exp := range allExports {
		if err := storage.CreateExportRecord(db, &exp); err != nil {
			t.Fatalf("Failed to create export %s: %v", exp.ID, err)
		}
	}

	// Test 1: User 1 should only see their 3 exports
	t.Run("User1 sees only their exports", func(t *testing.T) {
		exports, err := storage.ListExportsByDID(db, user1, 50, 0)
		if err != nil {
			t.Fatalf("Failed to list exports for user1: %v", err)
		}

		if len(exports) != 3 {
			t.Errorf("Expected 3 exports for user1, got %d", len(exports))
		}

		// Verify all exports belong to user1
		for _, exp := range exports {
			if exp.DID != user1 {
				t.Errorf("Found export with wrong DID: %s (expected %s)", exp.DID, user1)
			}
		}

		// Verify newest first ordering
		if len(exports) >= 2 {
			if exports[0].CreatedAt.Before(exports[1].CreatedAt) {
				t.Error("Exports not sorted by newest first")
			}
		}
	})

	// Test 2: User 2 should only see their 2 exports
	t.Run("User2 sees only their exports", func(t *testing.T) {
		exports, err := storage.ListExportsByDID(db, user2, 50, 0)
		if err != nil {
			t.Fatalf("Failed to list exports for user2: %v", err)
		}

		if len(exports) != 2 {
			t.Errorf("Expected 2 exports for user2, got %d", len(exports))
		}

		// Verify all exports belong to user2
		for _, exp := range exports {
			if exp.DID != user2 {
				t.Errorf("Found export with wrong DID: %s (expected %s)", exp.DID, user2)
			}
		}
	})

	// Test 3: User 3 should only see their 1 export
	t.Run("User3 sees only their export", func(t *testing.T) {
		exports, err := storage.ListExportsByDID(db, user3, 50, 0)
		if err != nil {
			t.Fatalf("Failed to list exports for user3: %v", err)
		}

		if len(exports) != 1 {
			t.Errorf("Expected 1 export for user3, got %d", len(exports))
		}

		if len(exports) > 0 && exports[0].DID != user3 {
			t.Errorf("Found export with wrong DID: %s (expected %s)", exports[0].DID, user3)
		}
	})

	// Test 4: Non-existent user should see zero exports
	t.Run("Non-existent user sees no exports", func(t *testing.T) {
		exports, err := storage.ListExportsByDID(db, "did:plc:nonexistent", 50, 0)
		if err != nil {
			t.Fatalf("Failed to list exports for nonexistent user: %v", err)
		}

		if len(exports) != 0 {
			t.Errorf("Expected 0 exports for nonexistent user, got %d", len(exports))
		}
	})

	// Test 5: Verify pagination works correctly
	t.Run("Pagination respects limits", func(t *testing.T) {
		// Get first 2 exports for user1
		exports, err := storage.ListExportsByDID(db, user1, 2, 0)
		if err != nil {
			t.Fatalf("Failed to list exports with limit: %v", err)
		}

		if len(exports) != 2 {
			t.Errorf("Expected 2 exports with limit=2, got %d", len(exports))
		}

		// Get next 2 exports (should only get 1 more)
		exports2, err := storage.ListExportsByDID(db, user1, 2, 2)
		if err != nil {
			t.Fatalf("Failed to list exports with offset: %v", err)
		}

		if len(exports2) != 1 {
			t.Errorf("Expected 1 export with offset=2, got %d", len(exports2))
		}

		// Verify no overlap
		if len(exports) > 0 && len(exports2) > 0 {
			if exports[0].ID == exports2[0].ID {
				t.Error("Pagination returned duplicate exports")
			}
		}
	})

	// Test 6: Verify total count across all users
	t.Run("Total exports across all users", func(t *testing.T) {
		// Count exports manually from database
		var total int
		err := db.QueryRow("SELECT COUNT(*) FROM exports").Scan(&total)
		if err != nil {
			t.Fatalf("Failed to count total exports: %v", err)
		}

		expectedTotal := len(user1Exports) + len(user2Exports) + len(user3Exports)
		if total != expectedTotal {
			t.Errorf("Expected %d total exports in database, got %d", expectedTotal, total)
		}
	})

	t.Log("✓ Export listing DID isolation test passed")
}

// TestExportListOrdering verifies exports are sorted by creation time (newest first)
func TestExportListOrdering(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	did := "did:plc:testuser"
	baseTime := time.Now()

	// Create exports with known timestamps
	exports := []models.ExportRecord{
		{
			ID:            fmt.Sprintf("%s/oldest", did),
			DID:           did,
			Format:        "json",
			CreatedAt:     baseTime.Add(-10 * time.Hour),
			DirectoryPath: "./exports/oldest",
			PostCount:     10,
			SizeBytes:     1000,
		},
		{
			ID:            fmt.Sprintf("%s/middle", did),
			DID:           did,
			Format:        "json",
			CreatedAt:     baseTime.Add(-5 * time.Hour),
			DirectoryPath: "./exports/middle",
			PostCount:     20,
			SizeBytes:     2000,
		},
		{
			ID:            fmt.Sprintf("%s/newest", did),
			DID:           did,
			Format:        "json",
			CreatedAt:     baseTime,
			DirectoryPath: "./exports/newest",
			PostCount:     30,
			SizeBytes:     3000,
		},
	}

	// Insert in random order
	for _, exp := range exports {
		if err := storage.CreateExportRecord(db, &exp); err != nil {
			t.Fatalf("Failed to create export: %v", err)
		}
	}

	// Retrieve and verify ordering
	retrieved, err := storage.ListExportsByDID(db, did, 50, 0)
	if err != nil {
		t.Fatalf("Failed to list exports: %v", err)
	}

	if len(retrieved) != 3 {
		t.Fatalf("Expected 3 exports, got %d", len(retrieved))
	}

	// Verify newest first
	if retrieved[0].ID != fmt.Sprintf("%s/newest", did) {
		t.Errorf("First export should be newest, got %s", retrieved[0].ID)
	}
	if retrieved[1].ID != fmt.Sprintf("%s/middle", did) {
		t.Errorf("Second export should be middle, got %s", retrieved[1].ID)
	}
	if retrieved[2].ID != fmt.Sprintf("%s/oldest", did) {
		t.Errorf("Third export should be oldest, got %s", retrieved[2].ID)
	}

	// Verify timestamps are in descending order
	for i := 0; i < len(retrieved)-1; i++ {
		if retrieved[i].CreatedAt.Before(retrieved[i+1].CreatedAt) {
			t.Errorf("Exports not sorted correctly: export[%d] is older than export[%d]", i, i+1)
		}
	}

	t.Log("✓ Export listing ordering test passed")
}

// Helper function to check if database file exists
func dbExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Helper function to count records in a table
func countRecords(db *sql.DB, table string) (int, error) {
	var count int
	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&count)
	return count, err
}
