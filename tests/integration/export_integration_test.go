package integration

import (
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/exporter"
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

// TestFullExportWorkflow tests the complete end-to-end export process
func TestFullExportWorkflow(t *testing.T) {
	// Setup: Create database and populate with test data
	db := setupTestDB(t)
	defer db.Close()

	testDID := "did:plc:test123"

	// Create test posts
	testPosts := []*models.Post{
		{
			URI:         "at://did:plc:test123/app.bsky.feed.post/post1",
			CID:         "bafytest1",
			DID:         testDID,
			Text:        "First test post for export",
			CreatedAt:   time.Now().UTC().Add(-3 * time.Hour),
			IndexedAt:   time.Now().UTC(),
			LikeCount:   15,
			RepostCount: 5,
			ReplyCount:  3,
			QuoteCount:  1,
			HasMedia:    false,
		},
		{
			URI:         "at://did:plc:test123/app.bsky.feed.post/post2",
			CID:         "bafytest2",
			DID:         testDID,
			Text:        "Second post with emoji 🚀 and unicode",
			CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
			IndexedAt:   time.Now().UTC(),
			LikeCount:   25,
			RepostCount: 10,
			ReplyCount:  5,
			QuoteCount:  2,
			HasMedia:    true,
			EmbedType:   "images",
		},
		{
			URI:         "at://did:plc:test123/app.bsky.feed.post/post3",
			CID:         "bafytest3",
			DID:         testDID,
			Text:        "Third post for testing",
			CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
			IndexedAt:   time.Now().UTC(),
			LikeCount:   5,
			RepostCount: 2,
			ReplyCount:  0,
			QuoteCount:  0,
			HasMedia:    false,
		},
	}

	// Save all posts to database
	for _, post := range testPosts {
		if err := storage.SavePost(db, post); err != nil {
			t.Fatalf("Failed to save test post: %v", err)
		}
	}

	// Create temporary directory for export
	tmpDir := t.TempDir()

	// Setup export job
	exportOptions := models.ExportOptions{
		Format:       models.ExportFormatJSON,
		OutputDir:    tmpDir,
		IncludeMedia: false, // Skip media for this test
		DID:          testDID,
		DateRange:    nil,
	}

	job := &models.ExportJob{
		ID:        "test-job-1",
		Options:   exportOptions,
		CreatedAt: time.Now(),
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel to track export progress
	progressChan := make(chan models.ExportProgress, 100)
	var progressUpdates []models.ExportProgress

	// Collect progress updates in background
	go func() {
		for progress := range progressChan {
			progressUpdates = append(progressUpdates, progress)
		}
	}()

	// Execute: Run the export
	err := exporter.Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Close progress channel and wait for collection to finish
	time.Sleep(100 * time.Millisecond) // Give time for progress updates to process

	// Verify: Check export results

	// 1. Verify export directory was created
	if job.ExportDir == "" {
		t.Fatal("ExportDir was not set on job")
	}

	exportDir := job.ExportDir
	if _, err := os.Stat(exportDir); os.IsNotExist(err) {
		t.Fatalf("Export directory was not created: %s", exportDir)
	}

	// 2. Verify posts.json was created and contains correct data
	postsPath := filepath.Join(exportDir, "posts.json")
	if _, err := os.Stat(postsPath); os.IsNotExist(err) {
		t.Fatalf("posts.json was not created: %s", postsPath)
	}

	// Read and parse posts.json
	postsData, err := os.ReadFile(postsPath)
	if err != nil {
		t.Fatalf("Failed to read posts.json: %v", err)
	}

	var exportedPosts []models.Post
	if err := json.Unmarshal(postsData, &exportedPosts); err != nil {
		t.Fatalf("Failed to unmarshal posts.json: %v", err)
	}

	if len(exportedPosts) != len(testPosts) {
		t.Errorf("Expected %d posts in export, got %d", len(testPosts), len(exportedPosts))
	}

	// Verify post content
	for i, post := range exportedPosts {
		if post.DID != testDID {
			t.Errorf("Post %d: Expected DID %s, got %s", i, testDID, post.DID)
		}
		if post.Text == "" {
			t.Errorf("Post %d: Text is empty", i)
		}
	}

	// 3. Verify manifest.json was created
	manifestPath := filepath.Join(exportDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Fatalf("manifest.json was not created: %s", manifestPath)
	}

	// Read and parse manifest.json
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest.json: %v", err)
	}

	var manifest models.ExportManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("Failed to unmarshal manifest.json: %v", err)
	}

	// Verify manifest contents
	if manifest.ExportFormat != string(models.ExportFormatJSON) {
		t.Errorf("Expected export format %s, got %s", models.ExportFormatJSON, manifest.ExportFormat)
	}

	if manifest.PostCount != len(testPosts) {
		t.Errorf("Expected PostCount %d, got %d", len(testPosts), manifest.PostCount)
	}

	if len(manifest.Files) == 0 {
		t.Error("Manifest Files list is empty")
	}

	// 4. Verify job progress was updated correctly
	if job.Progress.Status != models.ExportStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.ExportStatusCompleted, job.Progress.Status)
	}

	if job.Progress.PostsTotal != len(testPosts) {
		t.Errorf("Expected PostsTotal %d, got %d", len(testPosts), job.Progress.PostsTotal)
	}

	if job.Progress.PostsProcessed != len(testPosts) {
		t.Errorf("Expected PostsProcessed %d, got %d", len(testPosts), job.Progress.PostsProcessed)
	}

	// 5. Verify progress updates were sent
	if len(progressUpdates) == 0 {
		t.Error("Expected progress updates, got none")
	}
}

// TestFullExportWithMedia tests export workflow including media file copying
func TestFullExportWithMedia(t *testing.T) {
	// Setup database
	db := setupTestDB(t)
	defer db.Close()

	testDID := "did:plc:test456"

	// Create temporary media directory
	mediaSrcDir := t.TempDir()

	// Create test media files
	mediaFile1 := filepath.Join(mediaSrcDir, "bafyhash1.jpg")
	mediaFile2 := filepath.Join(mediaSrcDir, "bafyhash2.png")

	if err := os.WriteFile(mediaFile1, []byte("Image 1 content"), 0644); err != nil {
		t.Fatalf("Failed to create test media file: %v", err)
	}
	if err := os.WriteFile(mediaFile2, []byte("Image 2 content"), 0644); err != nil {
		t.Fatalf("Failed to create test media file: %v", err)
	}

	// Create test post with media
	testPost := &models.Post{
		URI:         "at://did:plc:test456/app.bsky.feed.post/media-post",
		CID:         "bafymedia",
		DID:         testDID,
		Text:        "Post with media attachments",
		CreatedAt:   time.Now().UTC(),
		IndexedAt:   time.Now().UTC(),
		HasMedia:    true,
		EmbedType:   "images",
		LikeCount:   10,
		RepostCount: 5,
	}

	if err := storage.SavePost(db, testPost); err != nil {
		t.Fatalf("Failed to save test post: %v", err)
	}

	// Create temporary export directory
	tmpDir := t.TempDir()

	// Setup export job with media
	exportOptions := models.ExportOptions{
		Format:       models.ExportFormatJSON,
		OutputDir:    tmpDir,
		IncludeMedia: true,
		DID:          testDID,
		DateRange:    nil,
	}

	job := &models.ExportJob{
		ID:        "test-media-job",
		Options:   exportOptions,
		CreatedAt: time.Now(),
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Note: Since we don't have actual media in the database for this test,
	// the media copying step will be skipped (no media records).
	// This test verifies the workflow still completes successfully.

	progressChan := make(chan models.ExportProgress, 100)

	// Run export
	err := exporter.Run(db, job, progressChan)
	// Note: exporter.Run closes the progressChan, no need to close it here

	if err != nil {
		t.Fatalf("Export with media failed: %v", err)
	}

	// Verify export completed
	if job.Progress.Status != models.ExportStatusCompleted {
		t.Errorf("Expected status %s, got %s", models.ExportStatusCompleted, job.Progress.Status)
	}

	// Verify posts.json was created
	postsPath := filepath.Join(job.ExportDir, "posts.json")
	if _, err := os.Stat(postsPath); os.IsNotExist(err) {
		t.Errorf("posts.json was not created: %s", postsPath)
	}

	// Verify manifest.json was created
	manifestPath := filepath.Join(job.ExportDir, "manifest.json")
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Errorf("manifest.json was not created: %s", manifestPath)
	}
}

// TestExportWithDateRange tests export with date range filtering (Phase 5 feature)
func TestExportWithDateRange(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	testDID := "did:plc:test789"
	now := time.Now().UTC()

	// Create posts across different dates
	oldPost := &models.Post{
		URI:       "at://did:plc:test789/app.bsky.feed.post/old",
		CID:       "bafyold",
		DID:       testDID,
		Text:      "Old post from 10 days ago",
		CreatedAt: now.Add(-10 * 24 * time.Hour),
		IndexedAt: now,
	}

	recentPost := &models.Post{
		URI:       "at://did:plc:test789/app.bsky.feed.post/recent",
		CID:       "bafyrecent",
		DID:       testDID,
		Text:      "Recent post from 2 days ago",
		CreatedAt: now.Add(-2 * 24 * time.Hour),
		IndexedAt: now,
	}

	if err := storage.SavePost(db, oldPost); err != nil {
		t.Fatalf("Failed to save old post: %v", err)
	}
	if err := storage.SavePost(db, recentPost); err != nil {
		t.Fatalf("Failed to save recent post: %v", err)
	}

	// Export with date range (last 3 days)
	tmpDir := t.TempDir()
	startDate := now.Add(-3 * 24 * time.Hour)

	exportOptions := models.ExportOptions{
		Format:    models.ExportFormatJSON,
		OutputDir: tmpDir,
		DID:       testDID,
		DateRange: &models.DateRange{
			StartDate: startDate,
			EndDate:   now,
		},
	}

	job := &models.ExportJob{
		ID:        "test-daterange-job",
		Options:   exportOptions,
		CreatedAt: time.Now(),
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	progressChan := make(chan models.ExportProgress, 100)
	err := exporter.Run(db, job, progressChan)
	// Note: exporter.Run closes the progressChan, no need to close it here

	if err != nil {
		t.Fatalf("Export with date range failed: %v", err)
	}

	// Read exported posts
	postsPath := filepath.Join(job.ExportDir, "posts.json")
	postsData, err := os.ReadFile(postsPath)
	if err != nil {
		t.Fatalf("Failed to read posts.json: %v", err)
	}

	var exportedPosts []models.Post
	if err := json.Unmarshal(postsData, &exportedPosts); err != nil {
		t.Fatalf("Failed to unmarshal posts.json: %v", err)
	}

	// Should only have the recent post (within date range)
	if len(exportedPosts) != 1 {
		t.Errorf("Expected 1 post within date range, got %d", len(exportedPosts))
	}

	if len(exportedPosts) > 0 && exportedPosts[0].URI != recentPost.URI {
		t.Errorf("Expected recent post, got %s", exportedPosts[0].URI)
	}

	// Verify manifest includes date range info
	manifestPath := filepath.Join(job.ExportDir, "manifest.json")
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("Failed to read manifest.json: %v", err)
	}

	var manifest models.ExportManifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("Failed to unmarshal manifest.json: %v", err)
	}

	if manifest.DateRange == nil {
		t.Error("Expected DateRange in manifest, got nil")
	}

	if manifest.PostCount != 1 {
		t.Errorf("Expected PostCount 1 in manifest, got %d", manifest.PostCount)
	}
}
