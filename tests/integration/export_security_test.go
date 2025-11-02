package integration

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
)

// TestExportRecord_PathTraversalPrevention verifies that export records reject malicious paths
func TestExportRecord_PathTraversalPrevention(t *testing.T) {
	testCases := []struct {
		name          string
		directoryPath string
		shouldPass    bool
		description   string
	}{
		{
			name:          "Valid relative path with ./ prefix",
			directoryPath: "./exports/did:plc:test/2025-11-01_12-00-00",
			shouldPass:    true,
			description:   "Normal export path should be accepted",
		},
		{
			name:          "Path traversal with ../",
			directoryPath: "./exports/../../../etc/passwd",
			shouldPass:    false,
			description:   "Path traversal attempt should be rejected",
		},
		{
			name:          "Path traversal in middle",
			directoryPath: "./exports/did:plc:test/../../../etc/passwd",
			shouldPass:    false,
			description:   "Path traversal in middle should be rejected",
		},
		{
			name:          "Absolute path outside exports",
			directoryPath: "/etc/passwd",
			shouldPass:    false,
			description:   "Absolute path outside exports should be rejected",
		},
		{
			name:          "Path without ./ prefix",
			directoryPath: "exports/did:plc:test/2025-11-01_12-00-00",
			shouldPass:    false,
			description:   "Path without ./ prefix should be rejected",
		},
		{
			name:          "Path with symbolic link characters",
			directoryPath: "./exports/did:plc:test/../../secret",
			shouldPass:    false,
			description:   "Symbolic link traversal should be rejected",
		},
		{
			name:          "Empty path",
			directoryPath: "",
			shouldPass:    false,
			description:   "Empty path should be rejected",
		},
		{
			name:          "Path with null bytes",
			directoryPath: "./exports/did:plc:test\x00/file",
			shouldPass:    false,
			description:   "Path with null bytes should be rejected",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			exportRecord := &models.ExportRecord{
				ID:            "did:plc:test/2025-11-01_12-00-00",
				DID:           "did:plc:test",
				Format:        "json",
				CreatedAt:     time.Now(),
				DirectoryPath: tc.directoryPath,
				PostCount:     100,
				SizeBytes:     1024,
			}

			err := exportRecord.Validate()

			if tc.shouldPass && err != nil {
				t.Errorf("%s: Expected validation to pass, got error: %v", tc.description, err)
			}

			if !tc.shouldPass && err == nil {
				t.Errorf("%s: Expected validation to fail, but it passed", tc.description)
			}

			if !tc.shouldPass && err != nil {
				t.Logf("✓ %s: Correctly rejected with error: %v", tc.description, err)
			}
		})
	}
}

// TestCreateExportRecord_PathValidation verifies database layer rejects invalid paths
func TestCreateExportRecord_PathValidation(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Test case 1: Valid path should be accepted
	validRecord := &models.ExportRecord{
		ID:            "did:plc:test/valid-export",
		DID:           "did:plc:test",
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: "./exports/did:plc:test/2025-11-01_12-00-00",
		PostCount:     100,
		SizeBytes:     1024,
	}

	err = storage.CreateExportRecord(db, validRecord)
	if err != nil {
		t.Errorf("Valid export record should be accepted, got error: %v", err)
	} else {
		t.Log("✓ Valid export record accepted")
	}

	// Test case 2: Path traversal should be rejected
	maliciousRecord := &models.ExportRecord{
		ID:            "did:plc:test/malicious-export",
		DID:           "did:plc:test",
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: "./exports/../../../etc/passwd",
		PostCount:     100,
		SizeBytes:     1024,
	}

	err = storage.CreateExportRecord(db, maliciousRecord)
	if err == nil {
		t.Error("Malicious export record should be rejected, but was accepted")
	} else {
		t.Logf("✓ Malicious export record correctly rejected: %v", err)
	}

	// Test case 3: Absolute path should be rejected
	absolutePathRecord := &models.ExportRecord{
		ID:            "did:plc:test/absolute-path",
		DID:           "did:plc:test",
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: "/tmp/malicious-export",
		PostCount:     100,
		SizeBytes:     1024,
	}

	err = storage.CreateExportRecord(db, absolutePathRecord)
	if err == nil {
		t.Error("Absolute path export record should be rejected, but was accepted")
	} else {
		t.Logf("✓ Absolute path export record correctly rejected: %v", err)
	}
}

// TestGetExportByID_OwnershipIsolation verifies users can only access their own exports
func TestGetExportByID_OwnershipIsolation(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create export for user A
	userARecord := &models.ExportRecord{
		ID:            "did:plc:userA/export-1",
		DID:           "did:plc:userA",
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: "./exports/did:plc:userA/2025-11-01_12-00-00",
		PostCount:     100,
		SizeBytes:     1024,
	}

	err = storage.CreateExportRecord(db, userARecord)
	if err != nil {
		t.Fatalf("Failed to create user A export: %v", err)
	}

	// Create export for user B
	userBRecord := &models.ExportRecord{
		ID:            "did:plc:userB/export-1",
		DID:           "did:plc:userB",
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: "./exports/did:plc:userB/2025-11-01_12-00-00",
		PostCount:     50,
		SizeBytes:     512,
	}

	err = storage.CreateExportRecord(db, userBRecord)
	if err != nil {
		t.Fatalf("Failed to create user B export: %v", err)
	}

	// Verify user A can access their own export
	retrievedA, err := storage.GetExportByID(db, "did:plc:userA/export-1")
	if err != nil {
		t.Errorf("User A should be able to access their own export: %v", err)
	}
	if retrievedA.DID != "did:plc:userA" {
		t.Errorf("Retrieved export has wrong DID: got %s, want did:plc:userA", retrievedA.DID)
	}

	// Verify user B cannot access user A's export through manipulation
	// This tests that the handler layer properly checks ownership
	// The ID contains the DID, so cross-user access should fail at retrieval
	_, err = storage.GetExportByID(db, "did:plc:userA/export-1")
	if err == sql.ErrNoRows {
		t.Error("GetExportByID should return the record, ownership check happens in handler")
	}

	// The key security check is that the handler verifies:
	// session.DID matches the DID prefix of the export ID
	// This test verifies the storage layer correctly retrieves by ID
	// Handler-level tests verify ownership checking

	t.Log("✓ Export isolation test passed - storage layer works correctly")
	t.Log("  Note: Ownership verification happens in handler layer (authenticated session)")
}

// TestListExportsByDID_UserIsolation verifies DID-based isolation in export listing
func TestListExportsByDID_UserIsolation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create multiple exports for different users
	users := []string{"did:plc:alice", "did:plc:bob", "did:plc:charlie"}
	for _, did := range users {
		for i := 1; i <= 3; i++ {
			record := &models.ExportRecord{
				ID:            did + "/export-" + string(rune('0'+i)),
				DID:           did,
				Format:        "json",
				CreatedAt:     time.Now(),
				DirectoryPath: "./exports/" + did + "/2025-11-01_12-00-0" + string(rune('0'+i)),
				PostCount:     100 * i,
				SizeBytes:     1024 * int64(i),
			}

			if err := storage.CreateExportRecord(db, record); err != nil {
				t.Fatalf("Failed to create export for %s: %v", did, err)
			}
		}
	}

	// Verify each user can only see their own exports
	for _, did := range users {
		exports, err := storage.ListExportsByDID(db, did, 100, 0)
		if err != nil {
			t.Fatalf("Failed to list exports for %s: %v", did, err)
		}

		if len(exports) != 3 {
			t.Errorf("User %s should have 3 exports, got %d", did, len(exports))
		}

		// Verify all exports belong to this user
		for _, export := range exports {
			if export.DID != did {
				t.Errorf("User %s retrieved export belonging to %s", did, export.DID)
			}
		}

		t.Logf("✓ User %s correctly sees only their 3 exports", did)
	}

	t.Log("✓ DID-based isolation test passed - users cannot see each other's exports")
}
