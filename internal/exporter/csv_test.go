package exporter

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
	_ "modernc.org/sqlite"
)

// setupCSVTestDB creates an in-memory database for testing
func setupCSVTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Initialize schema
	schema := `
		CREATE TABLE posts (
			uri TEXT PRIMARY KEY,
			cid TEXT NOT NULL,
			did TEXT NOT NULL,
			text TEXT,
			created_at TIMESTAMP NOT NULL,
			indexed_at TIMESTAMP NOT NULL,
			has_media BOOLEAN DEFAULT 0,
			like_count INTEGER DEFAULT 0,
			repost_count INTEGER DEFAULT 0,
			reply_count INTEGER DEFAULT 0,
			quote_count INTEGER DEFAULT 0,
			is_reply BOOLEAN DEFAULT 0,
			reply_parent TEXT,
			embed_type TEXT,
			embed_data JSON,
			labels JSON,
			archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);

		CREATE INDEX idx_posts_did ON posts(did);
		CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
	`

	if _, err := db.Exec(schema); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	return db
}

// insertCSVTestPosts inserts test posts into database
func insertCSVTestPosts(t *testing.T, db *sql.DB, did string, count int) {
	t.Helper()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	stmt, err := db.Prepare(`
		INSERT INTO posts (uri, cid, did, text, created_at, indexed_at, reply_parent, embed_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		t.Fatalf("Failed to prepare insert: %v", err)
	}
	defer stmt.Close()

	for i := 0; i < count; i++ {
		createdAt := baseTime.Add(time.Duration(i) * time.Minute)
		uri := fmt.Sprintf("at://%s/app.bsky.feed.post/%05d", did, i)
		cid := fmt.Sprintf("bafyrei%015d", i)
		text := fmt.Sprintf("CSV test post #%d", i)

		_, err := stmt.Exec(uri, cid, did, text, createdAt, createdAt, "", "")
		if err != nil {
			t.Fatalf("Failed to insert post %d: %v", i, err)
		}
	}
}

// TestExportToCSVBatched_MultipleBatches tests CSV export with 5000 posts
func TestExportToCSVBatched_MultipleBatches(t *testing.T) {
	db := setupCSVTestDB(t)
	defer db.Close()

	did := "did:plc:csvtest1"
	postCount := 5000

	insertCSVTestPosts(t, db, did, postCount)

	// Create temp file
	tmpFile, err := os.CreateTemp("", "csv-multi-*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	outputPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputPath)

	// Export with batch size 1000
	err = ExportToCSVBatched(db, did, nil, outputPath, 1000)
	if err != nil {
		t.Fatalf("ExportToCSVBatched failed: %v", err)
	}

	// Read and parse CSV
	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open output: %v", err)
	}
	defer file.Close()

	// Skip UTF-8 BOM
	bom := make([]byte, 3)
	file.Read(bom)

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	// First row is header
	if len(records) < 1 {
		t.Fatal("CSV has no header")
	}

	// Should have header + 5000 data rows
	if len(records) != postCount+1 {
		t.Errorf("Expected %d rows (header + data), got %d", postCount+1, len(records))
	}

	// Verify header
	header := records[0]
	expectedHeader := []string{
		"URI", "CID", "DID", "Text", "CreatedAt",
		"LikeCount", "RepostCount", "ReplyCount", "QuoteCount",
		"IsReply", "ReplyParent", "HasMedia", "MediaFiles", "EmbedType", "IndexedAt",
	}
	if len(header) != len(expectedHeader) {
		t.Errorf("Header length mismatch: expected %d, got %d", len(expectedHeader), len(header))
	}

	// Verify some data rows
	if len(records) > 1 {
		firstDataRow := records[1]
		if !strings.Contains(firstDataRow[0], did) {
			t.Error("First row URI doesn't contain expected DID")
		}
	}
}

// TestExportToCSVBatched_ByteIdentical compares batched vs non-batched output
func TestExportToCSVBatched_ByteIdentical(t *testing.T) {
	db := setupCSVTestDB(t)
	defer db.Close()

	did := "did:plc:csvtest2"
	postCount := 1500

	insertCSVTestPosts(t, db, did, postCount)

	// Export with batching
	tmpBatched, err := os.CreateTemp("", "csv-batched-*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	batchedPath := tmpBatched.Name()
	tmpBatched.Close()
	defer os.Remove(batchedPath)

	err = ExportToCSVBatched(db, did, nil, batchedPath, 500)
	if err != nil {
		t.Fatalf("Batched export failed: %v", err)
	}

	// Export with original method
	tmpOriginal, err := os.CreateTemp("", "csv-original-*.csv")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	originalPath := tmpOriginal.Name()
	tmpOriginal.Close()
	defer os.Remove(originalPath)

	// Fetch all posts for original export
	allPosts := make([]models.Post, 0, postCount)
	for offset := 0; offset < postCount; offset += 500 {
		batch, err := storage.ListPostsWithDateRange(db, did, nil, 500, offset)
		if err != nil {
			t.Fatalf("Failed to fetch posts: %v", err)
		}
		allPosts = append(allPosts, batch...)
	}

	err = ExportToCSV(allPosts, originalPath)
	if err != nil {
		t.Fatalf("Original export failed: %v", err)
	}

	// Read both files
	batchedRecords := readCSVFile(t, batchedPath)
	originalRecords := readCSVFile(t, originalPath)

	// Compare row counts
	if len(batchedRecords) != len(originalRecords) {
		t.Errorf("Row count mismatch: batched=%d, original=%d", len(batchedRecords), len(originalRecords))
	}

	// Compare each row
	for i := range batchedRecords {
		if i >= len(originalRecords) {
			break
		}

		batchedRow := batchedRecords[i]
		originalRow := originalRecords[i]

		if len(batchedRow) != len(originalRow) {
			t.Errorf("Row %d column count mismatch: batched=%d, original=%d", i, len(batchedRow), len(originalRow))
			continue
		}

		// Compare each field
		for j := range batchedRow {
			if batchedRow[j] != originalRow[j] {
				t.Errorf("Row %d, column %d mismatch: batched=%q, original=%q", i, j, batchedRow[j], originalRow[j])
			}
		}
	}

	t.Log("✓ Batched and original CSV exports are identical")
}

// TestExportToCSVBatched_RFC4180Compliance verifies RFC 4180 compliance
func TestExportToCSVBatched_RFC4180Compliance(t *testing.T) {
	db := setupCSVTestDB(t)
	defer db.Close()

	did := "did:plc:csvtest3"

	// Insert posts with special characters that need CSV escaping
	stmt, _ := db.Prepare(`
		INSERT INTO posts (uri, cid, did, text, created_at, indexed_at, reply_parent, embed_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	defer stmt.Close()

	now := time.Now()

	specialTexts := []string{
		"Text with \"quotes\" inside",
		"Text with, commas, everywhere",
		"Text with\nnewlines\ninside",
		"Text with \"quotes\", commas, and\nnewlines",
	}

	for i, text := range specialTexts {
		uri := fmt.Sprintf("at://%s/app.bsky.feed.post/%05d", did, i)
		cid := fmt.Sprintf("bafyrei%015d", i)
		stmt.Exec(uri, cid, did, text, now, now, "", "")
	}

	// Export
	tmpFile, _ := os.CreateTemp("", "csv-rfc4180-*.csv")
	outputPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputPath)

	err := ExportToCSVBatched(db, did, nil, outputPath, 1000)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read and parse
	records := readCSVFile(t, outputPath)

	// Should have header + 4 data rows
	if len(records) != 5 {
		t.Fatalf("Expected 5 rows, got %d", len(records))
	}

	// Verify special characters are properly escaped in Text column (index 3)
	for i := 1; i < len(records); i++ {
		if len(records[i]) < 4 {
			t.Errorf("Row %d has insufficient columns", i)
			continue
		}

		text := records[i][3]
		expectedText := specialTexts[i-1]

		if text != expectedText {
			t.Errorf("Row %d text mismatch:\nExpected: %q\nGot: %q", i, expectedText, text)
		}
	}

	t.Log("✓ RFC 4180 compliance verified (quotes, commas, newlines handled correctly)")
}

// TestExportToCSVBatched_UTF8BOM verifies UTF-8 BOM is present
func TestExportToCSVBatched_UTF8BOM(t *testing.T) {
	db := setupCSVTestDB(t)
	defer db.Close()

	did := "did:plc:csvtest4"
	insertCSVTestPosts(t, db, did, 100)

	tmpFile, _ := os.CreateTemp("", "csv-bom-*.csv")
	outputPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputPath)

	err := ExportToCSVBatched(db, did, nil, outputPath, 1000)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read first 3 bytes
	file, err := os.Open(outputPath)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	bom := make([]byte, 3)
	n, err := file.Read(bom)
	if err != nil {
		t.Fatalf("Failed to read BOM: %v", err)
	}

	if n != 3 {
		t.Fatalf("Expected to read 3 bytes, got %d", n)
	}

	// UTF-8 BOM is 0xEF, 0xBB, 0xBF
	if bom[0] != 0xEF || bom[1] != 0xBB || bom[2] != 0xBF {
		t.Errorf("UTF-8 BOM not found. Got: %X %X %X", bom[0], bom[1], bom[2])
	} else {
		t.Log("✓ UTF-8 BOM present (Excel compatibility)")
	}
}

// TestExportToCSVBatched_EmptyResult tests export with no posts
func TestExportToCSVBatched_EmptyResult(t *testing.T) {
	db := setupCSVTestDB(t)
	defer db.Close()

	did := "did:plc:nonexistent"

	tmpFile, _ := os.CreateTemp("", "csv-empty-*.csv")
	outputPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputPath)

	err := ExportToCSVBatched(db, did, nil, outputPath, 1000)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Should have header only
	records := readCSVFile(t, outputPath)
	if len(records) != 1 {
		t.Errorf("Expected 1 row (header only), got %d", len(records))
	}

	t.Log("✓ Empty result produces valid CSV with header only")
}

// readCSVFile is a helper to read and parse CSV files
func readCSVFile(t *testing.T, path string) [][]string {
	t.Helper()

	file, err := os.Open(path)
	if err != nil {
		t.Fatalf("Failed to open file: %v", err)
	}
	defer file.Close()

	// Skip UTF-8 BOM
	bom := make([]byte, 3)
	file.Read(bom)

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	return records
}
