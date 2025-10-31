package unit

import (
	"database/sql"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	// Use in-memory database
	db, err := storage.InitDB(":memory:")
	if err != nil {
		t.Fatalf("Failed to initialize test database: %v", err)
	}

	return db
}

func TestSaveAndGetPost(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	// Create test post
	testPost := &models.Post{
		URI:         "at://did:plc:test123/app.bsky.feed.post/abc123",
		CID:         "bafytest123",
		DID:         "did:plc:test123",
		Text:        "This is a test post",
		CreatedAt:   time.Now().UTC(),
		IndexedAt:   time.Now().UTC(),
		ReplyCount:  5,
		LikeCount:   10,
		RepostCount: 2,
		QuoteCount:  1,
	}

	// Test SavePost
	err := storage.SavePost(db, testPost)
	if err != nil {
		t.Fatalf("SavePost failed: %v", err)
	}

	// Test GetPost
	retrieved, err := storage.GetPost(db, testPost.URI)
	if err != nil {
		t.Fatalf("GetPost failed: %v", err)
	}

	// Verify fields
	if retrieved.URI != testPost.URI {
		t.Errorf("URI mismatch: got %s, want %s", retrieved.URI, testPost.URI)
	}
	if retrieved.CID != testPost.CID {
		t.Errorf("CID mismatch: got %s, want %s", retrieved.CID, testPost.CID)
	}
	if retrieved.Text != testPost.Text {
		t.Errorf("Text mismatch: got %s, want %s", retrieved.Text, testPost.Text)
	}
	if retrieved.LikeCount != testPost.LikeCount {
		t.Errorf("LikeCount mismatch: got %d, want %d", retrieved.LikeCount, testPost.LikeCount)
	}
	if retrieved.QuoteCount != testPost.QuoteCount {
		t.Errorf("QuoteCount mismatch: got %d, want %d", retrieved.QuoteCount, testPost.QuoteCount)
	}
}

func TestListPosts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testDID := "did:plc:test123"

	// Create multiple test posts
	posts := []*models.Post{
		{
			URI:       "at://did:plc:test123/app.bsky.feed.post/post1",
			CID:       "cid1",
			DID:       testDID,
			Text:      "First post",
			CreatedAt: time.Now().UTC().Add(-2 * time.Hour),
			IndexedAt: time.Now().UTC(),
		},
		{
			URI:       "at://did:plc:test123/app.bsky.feed.post/post2",
			CID:       "cid2",
			DID:       testDID,
			Text:      "Second post",
			CreatedAt: time.Now().UTC().Add(-1 * time.Hour),
			IndexedAt: time.Now().UTC(),
		},
		{
			URI:       "at://did:plc:test123/app.bsky.feed.post/post3",
			CID:       "cid3",
			DID:       testDID,
			Text:      "Third post",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
	}

	// Save all posts
	for _, post := range posts {
		if err := storage.SavePost(db, post); err != nil {
			t.Fatalf("Failed to save post: %v", err)
		}
	}

	// Test ListPosts with pagination
	resp, err := storage.ListPosts(db, testDID, 10, 0)
	if err != nil {
		t.Fatalf("ListPosts failed: %v", err)
	}

	if resp.Total != 3 {
		t.Errorf("Expected 3 total posts, got %d", resp.Total)
	}

	if len(resp.Posts) != 3 {
		t.Errorf("Expected 3 posts in response, got %d", len(resp.Posts))
	}

	// Verify posts are ordered by created_at DESC (newest first)
	if resp.Posts[0].Text != "Third post" {
		t.Errorf("Expected newest post first, got: %s", resp.Posts[0].Text)
	}
}

func TestSearchPosts(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testDID := "did:plc:test123"

	// Create test posts with searchable content
	posts := []*models.Post{
		{
			URI:       "at://did:plc:test123/app.bsky.feed.post/search1",
			CID:       "cid1",
			DID:       testDID,
			Text:      "I love programming in Go",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
		{
			URI:       "at://did:plc:test123/app.bsky.feed.post/search2",
			CID:       "cid2",
			DID:       testDID,
			Text:      "Python is great for data science",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
		{
			URI:       "at://did:plc:test123/app.bsky.feed.post/search3",
			CID:       "cid3",
			DID:       testDID,
			Text:      "Go has excellent concurrency support",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
	}

	// Save all posts
	for _, post := range posts {
		if err := storage.SavePost(db, post); err != nil {
			t.Fatalf("Failed to save post: %v", err)
		}
	}

	// Test search for "Go"
	resp, err := storage.SearchPosts(db, testDID, "Go", 10, 0)
	if err != nil {
		t.Fatalf("SearchPosts failed: %v", err)
	}

	if resp.Total != 2 {
		t.Errorf("Expected 2 posts matching 'Go', got %d", resp.Total)
	}

	// Test search for "Python"
	resp, err = storage.SearchPosts(db, testDID, "Python", 10, 0)
	if err != nil {
		t.Fatalf("SearchPosts failed: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("Expected 1 post matching 'Python', got %d", resp.Total)
	}

	// Test search with no results
	resp, err = storage.SearchPosts(db, testDID, "Rust", 10, 0)
	if err != nil {
		t.Fatalf("SearchPosts failed: %v", err)
	}

	if resp.Total != 0 {
		t.Errorf("Expected 0 posts matching 'Rust', got %d", resp.Total)
	}
}

func TestSearchPostsByATURI(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testDID := "did:plc:test123"
	testURI := "at://did:plc:test123/app.bsky.feed.post/aturi123"

	// Create test post
	testPost := &models.Post{
		URI:       testURI,
		CID:       "cidtest",
		DID:       testDID,
		Text:      "Test post for AT URI search",
		CreatedAt: time.Now().UTC(),
		IndexedAt: time.Now().UTC(),
	}

	// Save post
	if err := storage.SavePost(db, testPost); err != nil {
		t.Fatalf("Failed to save post: %v", err)
	}

	// Test search by AT URI
	resp, err := storage.SearchPosts(db, testDID, testURI, 10, 0)
	if err != nil {
		t.Fatalf("SearchPosts by AT URI failed: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("Expected 1 post when searching by AT URI, got %d", resp.Total)
	}

	if len(resp.Posts) > 0 && resp.Posts[0].URI != testURI {
		t.Errorf("Expected URI %s, got %s", testURI, resp.Posts[0].URI)
	}
}

func TestPostUpsert(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testURI := "at://did:plc:test123/app.bsky.feed.post/upsert1"

	// Create initial post
	initialPost := &models.Post{
		URI:       testURI,
		CID:       "cid1",
		DID:       "did:plc:test123",
		Text:      "Original text",
		LikeCount: 5,
		CreatedAt: time.Now().UTC(),
		IndexedAt: time.Now().UTC(),
	}

	// Save initial post
	if err := storage.SavePost(db, initialPost); err != nil {
		t.Fatalf("Failed to save initial post: %v", err)
	}

	// Update post with new like count
	updatedPost := &models.Post{
		URI:       testURI,
		CID:       "cid1",
		DID:       "did:plc:test123",
		Text:      "Original text",
		LikeCount: 10, // Updated
		CreatedAt: initialPost.CreatedAt,
		IndexedAt: time.Now().UTC(),
	}

	// Save updated post (should upsert)
	if err := storage.SavePost(db, updatedPost); err != nil {
		t.Fatalf("Failed to upsert post: %v", err)
	}

	// Retrieve and verify
	retrieved, err := storage.GetPost(db, testURI)
	if err != nil {
		t.Fatalf("Failed to get updated post: %v", err)
	}

	if retrieved.LikeCount != 10 {
		t.Errorf("Expected LikeCount to be updated to 10, got %d", retrieved.LikeCount)
	}

	// Verify we still only have one post (not duplicated)
	resp, err := storage.ListPosts(db, "did:plc:test123", 10, 0)
	if err != nil {
		t.Fatalf("Failed to list posts: %v", err)
	}

	if resp.Total != 1 {
		t.Errorf("Expected 1 post after upsert, got %d", resp.Total)
	}
}
