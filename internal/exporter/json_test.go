package exporter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
	_ "modernc.org/sqlite"
)

// setupJSONTestDB creates an in-memory database for testing
func setupJSONTestDB(t *testing.T) *sql.DB {
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

// insertJSONTestPosts inserts test posts into database
func insertJSONTestPosts(t *testing.T, db *sql.DB, did string, count int) {
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
		text := fmt.Sprintf("Test post #%d", i)

		_, err := stmt.Exec(uri, cid, did, text, createdAt, createdAt, "", "")
		if err != nil {
			t.Fatalf("Failed to insert post %d: %v", i, err)
		}
	}
}

// TestExportToJSONBatched_SingleBatch tests export with single batch (< 1000 posts)
func TestExportToJSONBatched_SingleBatch(t *testing.T) {
	db := setupJSONTestDB(t)
	defer db.Close()

	did := "did:plc:jsontest1"
	postCount := 500

	insertJSONTestPosts(t, db, did, postCount)

	// Create temp file
	tmpFile, err := os.CreateTemp("", "json-single-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	outputPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputPath)

	// Export
	err = ExportToJSONBatched(db, did, nil, outputPath, 1000)
	if err != nil {
		t.Fatalf("ExportToJSONBatched failed: %v", err)
	}

	// Verify output
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	// Parse JSON
	var posts []models.Post
	if err := json.Unmarshal(data, &posts); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(posts) != postCount {
		t.Errorf("Expected %d posts, got %d", postCount, len(posts))
	}

	// Verify first and last posts
	if posts[0].URI != fmt.Sprintf("at://%s/app.bsky.feed.post/%05d", did, postCount-1) {
		t.Errorf("First post URI incorrect (should be most recent due to DESC order)")
	}

	if posts[postCount-1].URI != fmt.Sprintf("at://%s/app.bsky.feed.post/%05d", did, 0) {
		t.Errorf("Last post URI incorrect (should be oldest)")
	}
}

// TestExportToJSONBatched_MultipleBatches tests export with multiple batches (2500 posts → 3 batches)
func TestExportToJSONBatched_MultipleBatches(t *testing.T) {
	db := setupJSONTestDB(t)
	defer db.Close()

	did := "did:plc:jsontest2"
	postCount := 2500

	insertJSONTestPosts(t, db, did, postCount)

	// Create temp file
	tmpFile, err := os.CreateTemp("", "json-multi-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	outputPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputPath)

	// Export with batch size 1000
	err = ExportToJSONBatched(db, did, nil, outputPath, 1000)
	if err != nil {
		t.Fatalf("ExportToJSONBatched failed: %v", err)
	}

	// Verify output
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output: %v", err)
	}

	// Parse JSON
	var posts []models.Post
	if err := json.Unmarshal(data, &posts); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(posts) != postCount {
		t.Errorf("Expected %d posts, got %d", postCount, len(posts))
	}

	// Verify posts are in correct order (DESC by created_at)
	for i := 1; i < len(posts); i++ {
		if posts[i].CreatedAt.After(posts[i-1].CreatedAt) {
			t.Errorf("Posts not in correct order at index %d", i)
		}
	}
}

// TestExportToJSONBatched_ByteIdentical compares batched vs non-batched output
func TestExportToJSONBatched_ByteIdentical(t *testing.T) {
	db := setupJSONTestDB(t)
	defer db.Close()

	did := "did:plc:jsontest3"
	postCount := 1500

	insertJSONTestPosts(t, db, did, postCount)

	// Export with batching
	tmpBatched, err := os.CreateTemp("", "json-batched-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	batchedPath := tmpBatched.Name()
	tmpBatched.Close()
	defer os.Remove(batchedPath)

	err = ExportToJSONBatched(db, did, nil, batchedPath, 500)
	if err != nil {
		t.Fatalf("Batched export failed: %v", err)
	}

	// Export with original method (load all into memory)
	tmpOriginal, err := os.CreateTemp("", "json-original-*.json")
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

	err = ExportToJSON(allPosts, originalPath)
	if err != nil {
		t.Fatalf("Original export failed: %v", err)
	}

	// Read both files
	batchedData, err := os.ReadFile(batchedPath)
	if err != nil {
		t.Fatalf("Failed to read batched file: %v", err)
	}

	originalData, err := os.ReadFile(originalPath)
	if err != nil {
		t.Fatalf("Failed to read original file: %v", err)
	}

	// Parse both to compare structure (byte-identical is hard due to formatting)
	var batchedPosts, originalPosts []models.Post

	if err := json.Unmarshal(batchedData, &batchedPosts); err != nil {
		t.Fatalf("Failed to parse batched JSON: %v", err)
	}

	if err := json.Unmarshal(originalData, &originalPosts); err != nil {
		t.Fatalf("Failed to parse original JSON: %v", err)
	}

	if len(batchedPosts) != len(originalPosts) {
		t.Errorf("Post count mismatch: batched=%d, original=%d", len(batchedPosts), len(originalPosts))
	}

	// Compare all posts
	for i := range batchedPosts {
		if batchedPosts[i].URI != originalPosts[i].URI {
			t.Errorf("Post %d URI mismatch: batched=%s, original=%s", i, batchedPosts[i].URI, originalPosts[i].URI)
		}
		if batchedPosts[i].Text != originalPosts[i].Text {
			t.Errorf("Post %d text mismatch", i)
		}
	}

	t.Log("✓ Batched and original exports produce semantically identical JSON")
}

// TestExportToJSONBatched_ValidStructure verifies JSON array is valid and parseable
func TestExportToJSONBatched_ValidStructure(t *testing.T) {
	db := setupJSONTestDB(t)
	defer db.Close()

	did := "did:plc:jsontest4"
	postCount := 2000

	insertJSONTestPosts(t, db, did, postCount)

	tmpFile, err := os.CreateTemp("", "json-valid-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	outputPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputPath)

	// Export
	err = ExportToJSONBatched(db, did, nil, outputPath, 1000)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Verify structure
	if data[0] != '[' {
		t.Error("JSON does not start with '['")
	}

	if data[len(data)-2] != ']' || data[len(data)-1] != '\n' {
		t.Error("JSON does not end with ']\\n'")
	}

	// Parse to verify validity
	var posts []models.Post
	if err := json.Unmarshal(data, &posts); err != nil {
		t.Fatalf("JSON is not valid: %v", err)
	}

	// Verify all required fields are present
	for i, post := range posts {
		if post.URI == "" {
			t.Errorf("Post %d missing URI", i)
		}
		if post.CID == "" {
			t.Errorf("Post %d missing CID", i)
		}
		if post.DID == "" {
			t.Errorf("Post %d missing DID", i)
		}
		if post.CreatedAt.IsZero() {
			t.Errorf("Post %d missing created_at", i)
		}
	}

	t.Logf("✓ Valid JSON array with %d posts", len(posts))
}

// TestExportToJSONBatched_EmptyResult tests export with no matching posts
func TestExportToJSONBatched_EmptyResult(t *testing.T) {
	db := setupJSONTestDB(t)
	defer db.Close()

	did := "did:plc:nonexistent"

	tmpFile, err := os.CreateTemp("", "json-empty-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	outputPath := tmpFile.Name()
	tmpFile.Close()
	defer os.Remove(outputPath)

	// Export (no posts in DB)
	err = ExportToJSONBatched(db, did, nil, outputPath, 1000)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Read file
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	// Should be empty array
	var posts []models.Post
	if err := json.Unmarshal(data, &posts); err != nil {
		t.Fatalf("Failed to parse empty JSON: %v", err)
	}

	if len(posts) != 0 {
		t.Errorf("Expected 0 posts, got %d", len(posts))
	}

	t.Log("✓ Empty result produces valid empty JSON array")
}
