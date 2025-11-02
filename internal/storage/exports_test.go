package storage

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
)

// TestCreateExportRecord verifies export record creation
func TestCreateExportRecord(t *testing.T) {
	// Setup test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create test export record
	now := time.Now()
	record := &models.ExportRecord{
		ID:            "did:plc:test123/2025-11-01_10-00-00",
		DID:           "did:plc:test123",
		Format:        "json",
		CreatedAt:     now,
		DirectoryPath: "./exports/did:plc:test123/2025-11-01_10-00-00",
		PostCount:     100,
		MediaCount:    50,
		SizeBytes:     1024 * 1024, // 1MB
		ManifestPath:  "./exports/did:plc:test123/2025-11-01_10-00-00/manifest.json",
	}

	// Test creation
	err = CreateExportRecord(db, record)
	if err != nil {
		t.Fatalf("Failed to create export record: %v", err)
	}

	// Verify record was created
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM exports WHERE id = ?", record.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count records: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 record, got %d", count)
	}

	t.Log("✓ CreateExportRecord test passed")
}

// TestCreateExportRecordWithDateRange verifies creation with date range filter
func TestCreateExportRecordWithDateRange(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	now := time.Now()
	startDate := now.AddDate(0, -1, 0) // 1 month ago
	endDate := now

	record := &models.ExportRecord{
		ID:             "did:plc:test456/2025-11-01_11-00-00",
		DID:            "did:plc:test456",
		Format:         "csv",
		CreatedAt:      now,
		DirectoryPath:  "./exports/did:plc:test456/2025-11-01_11-00-00",
		PostCount:      50,
		MediaCount:     25,
		SizeBytes:      512 * 1024,
		DateRangeStart: &startDate,
		DateRangeEnd:   &endDate,
	}

	err = CreateExportRecord(db, record)
	if err != nil {
		t.Fatalf("Failed to create export with date range: %v", err)
	}

	// Retrieve and verify
	retrieved, err := GetExportByID(db, record.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve export: %v", err)
	}

	if retrieved.DateRangeStart == nil {
		t.Error("Expected DateRangeStart to be set")
	}
	if retrieved.DateRangeEnd == nil {
		t.Error("Expected DateRangeEnd to be set")
	}

	t.Log("✓ CreateExportRecordWithDateRange test passed")
}

// TestCreateExportRecordValidation verifies validation is enforced
func TestCreateExportRecordValidation(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	tests := []struct {
		name    string
		record  *models.ExportRecord
		wantErr bool
	}{
		{
			name: "missing ID",
			record: &models.ExportRecord{
				DID:           "did:plc:test",
				Format:        "json",
				DirectoryPath: "./exports/test",
				PostCount:     10,
			},
			wantErr: true,
		},
		{
			name: "missing DID",
			record: &models.ExportRecord{
				ID:            "test/123",
				Format:        "json",
				DirectoryPath: "./exports/test",
				PostCount:     10,
			},
			wantErr: true,
		},
		{
			name: "invalid format",
			record: &models.ExportRecord{
				ID:            "test/123",
				DID:           "did:plc:test",
				Format:        "xml", // invalid
				DirectoryPath: "./exports/test",
				PostCount:     10,
			},
			wantErr: true,
		},
		{
			name: "invalid directory path",
			record: &models.ExportRecord{
				ID:            "test/123",
				DID:           "did:plc:test",
				Format:        "json",
				DirectoryPath: "/tmp/exports", // must start with ./exports/
				PostCount:     10,
			},
			wantErr: true,
		},
		{
			name: "negative post count",
			record: &models.ExportRecord{
				ID:            "test/123",
				DID:           "did:plc:test",
				Format:        "json",
				DirectoryPath: "./exports/test",
				PostCount:     -1, // invalid
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CreateExportRecord(db, tt.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateExportRecord() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	t.Log("✓ CreateExportRecordValidation test passed")
}

// TestGetExportByID verifies retrieval of export records
func TestGetExportByID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create test record
	now := time.Now()
	record := &models.ExportRecord{
		ID:            "did:plc:test789/2025-11-01_12-00-00",
		DID:           "did:plc:test789",
		Format:        "json",
		CreatedAt:     now,
		DirectoryPath: "./exports/did:plc:test789/2025-11-01_12-00-00",
		PostCount:     200,
		MediaCount:    100,
		SizeBytes:     2048 * 1024,
	}

	err = CreateExportRecord(db, record)
	if err != nil {
		t.Fatalf("Failed to create record: %v", err)
	}

	// Test retrieval
	retrieved, err := GetExportByID(db, record.ID)
	if err != nil {
		t.Fatalf("Failed to retrieve record: %v", err)
	}

	// Verify fields
	if retrieved.ID != record.ID {
		t.Errorf("ID mismatch: got %s, want %s", retrieved.ID, record.ID)
	}
	if retrieved.DID != record.DID {
		t.Errorf("DID mismatch: got %s, want %s", retrieved.DID, record.DID)
	}
	if retrieved.Format != record.Format {
		t.Errorf("Format mismatch: got %s, want %s", retrieved.Format, record.Format)
	}
	if retrieved.PostCount != record.PostCount {
		t.Errorf("PostCount mismatch: got %d, want %d", retrieved.PostCount, record.PostCount)
	}
	if retrieved.MediaCount != record.MediaCount {
		t.Errorf("MediaCount mismatch: got %d, want %d", retrieved.MediaCount, record.MediaCount)
	}
	if retrieved.SizeBytes != record.SizeBytes {
		t.Errorf("SizeBytes mismatch: got %d, want %d", retrieved.SizeBytes, record.SizeBytes)
	}

	// Test non-existent record
	_, err = GetExportByID(db, "nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent record, got nil")
	}

	t.Log("✓ GetExportByID test passed")
}

// TestListExportsByDID verifies listing exports for a user
func TestListExportsByDID(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	did := "did:plc:testlist"
	now := time.Now()

	// Create multiple exports for the same user
	for i := 0; i < 5; i++ {
		record := &models.ExportRecord{
			ID:            fmt.Sprintf("%s/2025-11-01_1%d-00-00", did, i),
			DID:           did,
			Format:        "json",
			CreatedAt:     now.Add(time.Duration(i) * time.Hour), // Different times
			DirectoryPath: fmt.Sprintf("./exports/%s/2025-11-01_1%d-00-00", did, i),
			PostCount:     100 * (i + 1),
			MediaCount:    50 * (i + 1),
			SizeBytes:     int64(1024 * (i + 1)),
		}
		err = CreateExportRecord(db, record)
		if err != nil {
			t.Fatalf("Failed to create record %d: %v", i, err)
		}
	}

	// Create export for different user (should not be included)
	otherRecord := &models.ExportRecord{
		ID:            "did:plc:other/2025-11-01_20-00-00",
		DID:           "did:plc:other",
		Format:        "csv",
		CreatedAt:     now,
		DirectoryPath: "./exports/did:plc:other/2025-11-01_20-00-00",
		PostCount:     10,
		MediaCount:    5,
		SizeBytes:     1024,
	}
	err = CreateExportRecord(db, otherRecord)
	if err != nil {
		t.Fatalf("Failed to create other record: %v", err)
	}

	// Test listing
	exports, err := ListExportsByDID(db, did, 10, 0)
	if err != nil {
		t.Fatalf("Failed to list exports: %v", err)
	}

	// Verify count
	if len(exports) != 5 {
		t.Errorf("Expected 5 exports, got %d", len(exports))
	}

	// Verify ordering (newest first)
	for i := 1; i < len(exports); i++ {
		if exports[i].CreatedAt.After(exports[i-1].CreatedAt) {
			t.Error("Exports not sorted by created_at DESC")
		}
	}

	// Test pagination
	page1, err := ListExportsByDID(db, did, 2, 0)
	if err != nil {
		t.Fatalf("Failed to get page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("Expected 2 exports in page 1, got %d", len(page1))
	}

	page2, err := ListExportsByDID(db, did, 2, 2)
	if err != nil {
		t.Fatalf("Failed to get page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("Expected 2 exports in page 2, got %d", len(page2))
	}

	// Verify pagination returns different records
	if page1[0].ID == page2[0].ID {
		t.Error("Pagination returned duplicate records")
	}

	t.Log("✓ ListExportsByDID test passed")
}

// TestDeleteExport verifies export record deletion
func TestDeleteExport(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create test record
	record := &models.ExportRecord{
		ID:            "did:plc:testdel/2025-11-01_15-00-00",
		DID:           "did:plc:testdel",
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: "./exports/did:plc:testdel/2025-11-01_15-00-00",
		PostCount:     100,
		MediaCount:    50,
		SizeBytes:     1024,
	}

	err = CreateExportRecord(db, record)
	if err != nil {
		t.Fatalf("Failed to create record: %v", err)
	}

	// Verify record exists
	_, err = GetExportByID(db, record.ID)
	if err != nil {
		t.Fatalf("Record should exist before deletion: %v", err)
	}

	// Test deletion
	err = DeleteExport(db, record.ID)
	if err != nil {
		t.Fatalf("Failed to delete export: %v", err)
	}

	// Verify record is gone
	_, err = GetExportByID(db, record.ID)
	if err == nil {
		t.Error("Record should not exist after deletion")
	}

	// Test deleting non-existent record
	err = DeleteExport(db, "nonexistent")
	if err == nil {
		t.Error("Expected error when deleting non-existent record")
	}

	t.Log("✓ DeleteExport test passed")
}

// TestExportRecordHelperMethods tests the helper methods on ExportRecord
func TestExportRecordHelperMethods(t *testing.T) {
	now := time.Now()
	startDate := now.AddDate(0, -1, 0)
	endDate := now

	tests := []struct {
		name          string
		record        models.ExportRecord
		expectedSize  string
		expectedRange string
	}{
		{
			name: "bytes only",
			record: models.ExportRecord{
				SizeBytes: 512,
			},
			expectedSize: "512 B",
		},
		{
			name: "kilobytes",
			record: models.ExportRecord{
				SizeBytes: 1024 * 5, // 5KB
			},
			expectedSize: "5.0 KB",
		},
		{
			name: "megabytes",
			record: models.ExportRecord{
				SizeBytes: 1024 * 1024 * 10, // 10MB
			},
			expectedSize: "10.0 MB",
		},
		{
			name: "no date range",
			record: models.ExportRecord{
				DateRangeStart: nil,
				DateRangeEnd:   nil,
			},
			expectedRange: "All posts",
		},
		{
			name: "with date range",
			record: models.ExportRecord{
				DateRangeStart: &startDate,
				DateRangeEnd:   &endDate,
			},
			expectedRange: fmt.Sprintf("%s to %s",
				startDate.Format("2006-01-02"),
				endDate.Format("2006-01-02")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedSize != "" {
				got := tt.record.HumanSize()
				if got != tt.expectedSize {
					t.Errorf("HumanSize() = %v, want %v", got, tt.expectedSize)
				}
			}
			if tt.expectedRange != "" {
				got := tt.record.DateRangeString()
				if got != tt.expectedRange {
					t.Errorf("DateRangeString() = %v, want %v", got, tt.expectedRange)
				}
			}
		})
	}

	t.Log("✓ ExportRecordHelperMethods test passed")
}
