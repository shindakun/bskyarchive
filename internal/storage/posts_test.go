package storage

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	_ "modernc.org/sqlite"
)

// setupTestDB creates an in-memory database for testing
func setupTestDB(t *testing.T) *sql.DB {
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

// insertTestPosts inserts test posts into the database
func insertTestPosts(t *testing.T, db *sql.DB, did string, count int) []models.Post {
	t.Helper()

	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	posts := make([]models.Post, count)

	stmt, err := db.Prepare(`
		INSERT INTO posts (uri, cid, did, text, created_at, indexed_at, reply_parent, embed_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		t.Fatalf("Failed to prepare insert statement: %v", err)
	}
	defer stmt.Close()

	for i := 0; i < count; i++ {
		// Create timestamp - spread posts over time
		// To test tie-breaking, some posts will have identical timestamps
		createdAt := baseTime.Add(time.Duration(i/10) * time.Hour) // Group by 10s
		uri := fmt.Sprintf("at://%s/app.bsky.feed.post/%05d", did, i)
		cid := fmt.Sprintf("bafyrei%015d", i)
		text := fmt.Sprintf("Test post #%d", i)

		// Use empty strings instead of NULL for string fields to avoid scan errors
		_, err := stmt.Exec(uri, cid, did, text, createdAt, createdAt, "", "")
		if err != nil {
			t.Fatalf("Failed to insert test post %d: %v", i, err)
		}

		posts[i] = models.Post{
			URI:       uri,
			CID:       cid,
			DID:       did,
			Text:      text,
			CreatedAt: createdAt,
			IndexedAt: createdAt,
		}
	}

	return posts
}

// TestListPostsWithDateRange_Pagination tests basic pagination functionality
func TestListPostsWithDateRange_Pagination(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	did := "did:plc:test123"
	postCount := 2500

	// Insert test posts
	insertTestPosts(t, db, did, postCount)

	tests := []struct {
		name          string
		limit         int
		offset        int
		expectedCount int
	}{
		{
			name:          "First batch (offset 0)",
			limit:         1000,
			offset:        0,
			expectedCount: 1000,
		},
		{
			name:          "Second batch (offset 1000)",
			limit:         1000,
			offset:        1000,
			expectedCount: 1000,
		},
		{
			name:          "Third batch (offset 2000)",
			limit:         1000,
			offset:        2000,
			expectedCount: 500, // Only 500 posts remaining
		},
		{
			name:          "Beyond total count (offset 3000)",
			limit:         1000,
			offset:        3000,
			expectedCount: 0, // No posts remaining
		},
		{
			name:          "Small batch",
			limit:         100,
			offset:        0,
			expectedCount: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			posts, err := ListPostsWithDateRange(db, did, nil, tt.limit, tt.offset)
			if err != nil {
				t.Fatalf("ListPostsWithDateRange failed: %v", err)
			}

			if len(posts) != tt.expectedCount {
				t.Errorf("Expected %d posts, got %d", tt.expectedCount, len(posts))
			}

			// Verify posts are ordered correctly (DESC by created_at, then ASC by uri)
			for i := 1; i < len(posts); i++ {
				prev := posts[i-1]
				curr := posts[i]

				// If same timestamp, uri should be in ascending order
				if prev.CreatedAt.Equal(curr.CreatedAt) {
					if prev.URI >= curr.URI {
						t.Errorf("Posts not sorted correctly: prev.URI=%s >= curr.URI=%s at same timestamp",
							prev.URI, curr.URI)
					}
				} else {
					// Otherwise created_at should be descending
					if prev.CreatedAt.Before(curr.CreatedAt) {
						t.Errorf("Posts not sorted correctly: prev.CreatedAt=%v < curr.CreatedAt=%v",
							prev.CreatedAt, curr.CreatedAt)
					}
				}
			}
		})
	}
}

// TestListPostsWithDateRange_LastBatchHandling tests handling of last partial batch
func TestListPostsWithDateRange_LastBatchHandling(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	did := "did:plc:test456"

	tests := []struct {
		name           string
		totalPosts     int
		batchSize      int
		expectedBatches []int // Expected sizes for each batch
	}{
		{
			name:           "Exact multiple of batch size (3000 posts, 1000 per batch)",
			totalPosts:     3000,
			batchSize:      1000,
			expectedBatches: []int{1000, 1000, 1000, 0}, // Last query returns 0
		},
		{
			name:           "Partial last batch (2500 posts, 1000 per batch)",
			totalPosts:     2500,
			batchSize:      1000,
			expectedBatches: []int{1000, 1000, 500, 0},
		},
		{
			name:           "Single partial batch (500 posts, 1000 per batch)",
			totalPosts:     500,
			batchSize:      1000,
			expectedBatches: []int{500, 0},
		},
		{
			name:           "Small batch size (2500 posts, 100 per batch)",
			totalPosts:     2500,
			batchSize:      100,
			expectedBatches: []int{100, 100, 100}, // Just test first few
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear database and insert new posts
			db.Exec("DELETE FROM posts")
			insertTestPosts(t, db, did, tt.totalPosts)

			// Fetch batches
			offset := 0
			for batchNum, expectedSize := range tt.expectedBatches {
				posts, err := ListPostsWithDateRange(db, did, nil, tt.batchSize, offset)
				if err != nil {
					t.Fatalf("Batch %d failed: %v", batchNum, err)
				}

				if len(posts) != expectedSize {
					t.Errorf("Batch %d: expected %d posts, got %d", batchNum, expectedSize, len(posts))
				}

				// If this was a partial batch or empty batch, we're done
				if len(posts) < tt.batchSize {
					break
				}

				offset += tt.batchSize

				// For small batch size test, only test first 3 batches
				if batchNum >= 2 && tt.batchSize == 100 {
					break
				}
			}
		})
	}
}

// TestListPostsWithDateRange_OrderByDeterminism tests that identical queries return identical results
func TestListPostsWithDateRange_OrderByDeterminism(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	did := "did:plc:test789"

	// Insert posts with many identical timestamps to test tie-breaking
	baseTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	stmt, err := db.Prepare(`
		INSERT INTO posts (uri, cid, did, text, created_at, indexed_at, reply_parent, embed_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		t.Fatalf("Failed to prepare statement: %v", err)
	}
	defer stmt.Close()

	// Insert 100 posts with only 10 unique timestamps (10 posts per timestamp)
	for i := 0; i < 100; i++ {
		timestamp := baseTime.Add(time.Duration(i/10) * time.Hour)
		uri := fmt.Sprintf("at://%s/app.bsky.feed.post/%05d", did, i)
		cid := fmt.Sprintf("bafyrei%015d", i)
		text := fmt.Sprintf("Post %d", i)

		_, err := stmt.Exec(uri, cid, did, text, timestamp, timestamp, "", "")
		if err != nil {
			t.Fatalf("Failed to insert post %d: %v", i, err)
		}
	}

	// Query multiple times with same parameters
	const iterations = 5
	const limit = 50
	const offset = 25

	var results [][]string // Store URIs from each query

	for i := 0; i < iterations; i++ {
		posts, err := ListPostsWithDateRange(db, did, nil, limit, offset)
		if err != nil {
			t.Fatalf("Query iteration %d failed: %v", i, err)
		}

		uris := make([]string, len(posts))
		for j, post := range posts {
			uris[j] = post.URI
		}
		results = append(results, uris)
	}

	// Verify all results are identical
	for i := 1; i < iterations; i++ {
		if len(results[i]) != len(results[0]) {
			t.Fatalf("Iteration %d returned %d posts, expected %d", i, len(results[i]), len(results[0]))
		}

		for j := range results[i] {
			if results[i][j] != results[0][j] {
				t.Errorf("Iteration %d, position %d: got URI %s, expected %s",
					i, j, results[i][j], results[0][j])
			}
		}
	}

	t.Logf("✓ Determinism verified: %d identical queries returned identical results", iterations)
}

// TestListPostsWithDateRange_DefaultLimit tests that limit=0 uses default value
func TestListPostsWithDateRange_DefaultLimit(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	did := "did:plc:testdefault"
	insertTestPosts(t, db, did, 1500)

	// Call with limit=0, should use default (1000)
	posts, err := ListPostsWithDateRange(db, did, nil, 0, 0)
	if err != nil {
		t.Fatalf("Failed to query with limit=0: %v", err)
	}

	// Should return 1000 (default limit) not 1500
	if len(posts) != 1000 {
		t.Errorf("Expected default limit of 1000, got %d posts", len(posts))
	}
}

// TestListPostsWithDateRange_NegativeOffset tests that negative offset is treated as 0
func TestListPostsWithDateRange_NegativeOffset(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	did := "did:plc:testnegative"
	insertTestPosts(t, db, did, 100)

	// Call with negative offset
	posts, err := ListPostsWithDateRange(db, did, nil, 50, -10)
	if err != nil {
		t.Fatalf("Failed to query with negative offset: %v", err)
	}

	// Should return first 50 posts (offset treated as 0)
	if len(posts) != 50 {
		t.Errorf("Expected 50 posts, got %d", len(posts))
	}

	// Verify these are the first 50 posts (most recent due to DESC order)
	postsFromZero, _ := ListPostsWithDateRange(db, did, nil, 50, 0)
	if posts[0].URI != postsFromZero[0].URI {
		t.Errorf("Negative offset did not behave like offset=0")
	}
}
