package storage

import (
	"path/filepath"
	"testing"
)

// TestExportsTableMigration verifies that the exports table migration works correctly
func TestExportsTableMigration(t *testing.T) {
	// Create temporary test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Initialize database (will run all migrations including exports table)
	db, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Verify exports table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='exports'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Exports table not found: %v", err)
	}
	if tableName != "exports" {
		t.Errorf("Expected table name 'exports', got '%s'", tableName)
	}

	// Verify indices exist
	expectedIndices := map[string]bool{
		"idx_exports_did":         false,
		"idx_exports_created_at":  false,
		"idx_exports_did_created": false,
	}

	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='exports'")
	if err != nil {
		t.Fatalf("Failed to query indices: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var indexName string
		if err := rows.Scan(&indexName); err != nil {
			t.Fatalf("Failed to scan index name: %v", err)
		}
		if _, exists := expectedIndices[indexName]; exists {
			expectedIndices[indexName] = true
		}
	}

	// Verify all expected indices were found
	for indexName, found := range expectedIndices {
		if !found {
			t.Errorf("Expected index '%s' not found", indexName)
		}
	}

	// Verify schema version is at least 3
	var version int
	err = db.QueryRow("SELECT version FROM schema_version ORDER BY version DESC LIMIT 1").Scan(&version)
	if err != nil {
		t.Fatalf("Failed to get schema version: %v", err)
	}
	if version < 3 {
		t.Errorf("Expected schema version >= 3, got %d", version)
	}

	// Verify table structure by trying to insert a record
	_, err = db.Exec(`
		INSERT INTO exports (id, did, format, created_at, directory_path, post_count, media_count, size_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, "test_id", "did:test:123", "json", 1234567890, "./exports/test", 10, 5, 1024)
	if err != nil {
		t.Fatalf("Failed to insert test record: %v", err)
	}

	// Verify record was inserted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM exports WHERE id = 'test_id'").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query test record: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 record, got %d", count)
	}

	// Verify constraints work (negative post_count should fail)
	_, err = db.Exec(`
		INSERT INTO exports (id, did, format, created_at, directory_path, post_count)
		VALUES (?, ?, ?, ?, ?, ?)
	`, "test_id_2", "did:test:456", "csv", 1234567890, "./exports/test2", -1)
	if err == nil {
		t.Error("Expected constraint violation for negative post_count, but insert succeeded")
	}

	t.Log("✓ Migration test passed")
	t.Logf("✓ Table 'exports' created successfully")
	t.Logf("✓ All indices created successfully")
	t.Logf("✓ Schema version: %d", version)
	t.Logf("✓ Constraints working correctly")
}

// TestExportsTableIdempotent verifies migration can run multiple times safely
func TestExportsTableIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Run initialization twice
	db1, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("First initialization failed: %v", err)
	}
	db1.Close()

	// Second initialization should work without errors
	db2, err := InitDB(dbPath)
	if err != nil {
		t.Fatalf("Second initialization failed: %v", err)
	}
	defer db2.Close()

	// Verify table still exists and is functional
	var count int
	err = db2.QueryRow("SELECT COUNT(*) FROM exports").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query exports table: %v", err)
	}

	t.Log("✓ Idempotent migration test passed")
}
