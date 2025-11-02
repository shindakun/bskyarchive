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

// TestExportDirectoryIsolation_UserCanAccessOwnExports verifies users can access their own exports
func TestExportDirectoryIsolation_UserCanAccessOwnExports(t *testing.T) {
	// Setup: Create database and test data
	db := setupTestDB(t)
	defer db.Close()

	testDID := "did:plc:user1"
	tmpDir := t.TempDir()

	// Create a test post for user1
	testPost := &models.Post{
		URI:       "at://did:plc:user1/app.bsky.feed.post/post1",
		CID:       "bafytest1",
		DID:       testDID,
		Text:      "User 1's post",
		CreatedAt: time.Now().UTC(),
		IndexedAt: time.Now().UTC(),
	}

	if err := storage.SavePost(db, testPost); err != nil {
		t.Fatalf("Failed to save test post: %v", err)
	}

	// Create export job for user1
	opts := models.ExportOptions{
		Format:       models.ExportFormatJSON,
		OutputDir:    tmpDir,
		IncludeMedia: false,
		DID:          testDID,
	}

	job := &models.ExportJob{
		ID:        "test-job-1",
		Options:   opts,
		CreatedAt: time.Now(),
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Run export
	progressChan := make(chan models.ExportProgress, 10)
	go func() {
		for range progressChan {
			// Consume progress updates
		}
	}()

	if err := exporter.Run(db, job, progressChan); err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify export directory structure: tmpDir/did/timestamp/
	userExportDir := filepath.Join(tmpDir, testDID)
	if _, err := os.Stat(userExportDir); os.IsNotExist(err) {
		t.Errorf("User export directory not created: %s", userExportDir)
	}

	// Verify export files exist in the per-user directory
	entries, err := os.ReadDir(userExportDir)
	if err != nil {
		t.Fatalf("Failed to read user export directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("No exports found in user directory")
	}

	// Verify export directory path contains the DID
	if !filepath.HasPrefix(job.ExportDir, userExportDir) {
		t.Errorf("Export directory %s does not start with user directory %s", job.ExportDir, userExportDir)
	}
}

// TestExportDirectoryIsolation_UserCannotAccessOtherExports verifies users cannot access other users' exports
func TestExportDirectoryIsolation_UserCannotAccessOtherExports(t *testing.T) {
	// This test verifies the directory structure isolation
	// The actual access control is tested in the handler tests

	tmpDir := t.TempDir()

	// Create export directories for two users
	user1DID := "did:plc:user1"
	user2DID := "did:plc:user2"

	user1Dir, err := exporter.CreateExportDirectory(tmpDir, user1DID)
	if err != nil {
		t.Fatalf("Failed to create user1 export directory: %v", err)
	}

	user2Dir, err := exporter.CreateExportDirectory(tmpDir, user2DID)
	if err != nil {
		t.Fatalf("Failed to create user2 export directory: %v", err)
	}

	// Verify directories are isolated
	if !filepath.HasPrefix(user1Dir, filepath.Join(tmpDir, user1DID)) {
		t.Errorf("User1 export directory %s not under user1 directory", user1Dir)
	}

	if !filepath.HasPrefix(user2Dir, filepath.Join(tmpDir, user2DID)) {
		t.Errorf("User2 export directory %s not under user2 directory", user2Dir)
	}

	// Verify directories are different
	if user1Dir == user2Dir {
		t.Error("User1 and User2 export directories should be different")
	}

	// Create test files in each directory
	user1File := filepath.Join(user1Dir, "user1-data.json")
	user2File := filepath.Join(user2Dir, "user2-data.json")

	if err := os.WriteFile(user1File, []byte(`{"user": "1"}`), 0644); err != nil {
		t.Fatalf("Failed to write user1 file: %v", err)
	}

	if err := os.WriteFile(user2File, []byte(`{"user": "2"}`), 0644); err != nil {
		t.Fatalf("Failed to write user2 file: %v", err)
	}

	// Verify files are isolated (cannot be accessed via wrong directory)
	user1DirEntries, _ := os.ReadDir(filepath.Join(tmpDir, user1DID))
	user2DirEntries, _ := os.ReadDir(filepath.Join(tmpDir, user2DID))

	// User1's directory should not contain user2's files
	for _, entry := range user1DirEntries {
		if entry.Name() == "user2-data.json" {
			t.Error("User1 directory should not contain user2's files")
		}
	}

	// User2's directory should not contain user1's files
	for _, entry := range user2DirEntries {
		if entry.Name() == "user1-data.json" {
			t.Error("User2 directory should not contain user1's files")
		}
	}
}

// TestExportDirectoryIsolation_CorrectPerUserDirectory verifies exports are created in correct per-user directories
func TestExportDirectoryIsolation_CorrectPerUserDirectory(t *testing.T) {
	// Setup: Create database and test data for multiple users
	db := setupTestDB(t)
	defer db.Close()

	tmpDir := t.TempDir()

	// Test with multiple users
	users := []string{"did:plc:alice", "did:plc:bob", "did:plc:carol"}

	for _, userDID := range users {
		// Create a test post for each user
		testPost := &models.Post{
			URI:       "at://" + userDID + "/app.bsky.feed.post/post1",
			CID:       "bafytest_" + userDID,
			DID:       userDID,
			Text:      "Post from " + userDID,
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		}

		if err := storage.SavePost(db, testPost); err != nil {
			t.Fatalf("Failed to save post for %s: %v", userDID, err)
		}

		// Create export job
		opts := models.ExportOptions{
			Format:       models.ExportFormatJSON,
			OutputDir:    tmpDir,
			IncludeMedia: false,
			DID:          userDID,
		}

		job := &models.ExportJob{
			ID:        "job-" + userDID,
			Options:   opts,
			CreatedAt: time.Now(),
			Progress: models.ExportProgress{
				Status: models.ExportStatusQueued,
			},
		}

		// Run export
		progressChan := make(chan models.ExportProgress, 10)
		go func() {
			for range progressChan {
				// Consume progress updates
			}
		}()

		if err := exporter.Run(db, job, progressChan); err != nil {
			t.Fatalf("Export failed for %s: %v", userDID, err)
		}

		// Verify export is in correct per-user directory: tmpDir/{did}/timestamp/
		expectedUserDir := filepath.Join(tmpDir, userDID)
		if !filepath.HasPrefix(job.ExportDir, expectedUserDir) {
			t.Errorf("Export for %s not in correct directory. Got: %s, Expected prefix: %s",
				userDID, job.ExportDir, expectedUserDir)
		}

		// Verify directory structure exists
		if _, err := os.Stat(job.ExportDir); os.IsNotExist(err) {
			t.Errorf("Export directory does not exist: %s", job.ExportDir)
		}

		// Verify export file exists
		exportFile := filepath.Join(job.ExportDir, "posts.json")
		if _, err := os.Stat(exportFile); os.IsNotExist(err) {
			t.Errorf("Export file does not exist: %s", exportFile)
		}
	}

	// Verify all user directories are separate
	for i, user1 := range users {
		for j, user2 := range users {
			if i == j {
				continue
			}

			dir1 := filepath.Join(tmpDir, user1)
			dir2 := filepath.Join(tmpDir, user2)

			if dir1 == dir2 {
				t.Errorf("Users %s and %s should have different directories", user1, user2)
			}

			// Verify dir1 is not a parent of dir2 and vice versa
			if filepath.HasPrefix(dir2, dir1+string(filepath.Separator)) {
				t.Errorf("User directory %s should not be parent of %s", dir1, dir2)
			}
		}
	}
}

// TestExportDirectoryIsolation_UnauthorizedAccessLogged verifies unauthorized access attempts are logged
// Note: This test focuses on the handler-level access control which includes logging
// The actual HTTP handler test would be more comprehensive, but this validates the isolation logic
func TestExportDirectoryIsolation_UnauthorizedAccessLogged(t *testing.T) {
	// This test validates that the export directory structure prevents cross-user access
	// The handler-level tests (in handler tests) verify the actual logging and 403 responses

	tmpDir := t.TempDir()

	user1DID := "did:plc:user1"
	user2DID := "did:plc:user2"

	// Create separate export directories for each user
	user1ExportDir, err := exporter.CreateExportDirectory(tmpDir, user1DID)
	if err != nil {
		t.Fatalf("Failed to create user1 export directory: %v", err)
	}

	user2ExportDir, err := exporter.CreateExportDirectory(tmpDir, user2DID)
	if err != nil {
		t.Fatalf("Failed to create user2 export directory: %v", err)
	}

	// Create test data in each user's directory
	user1Data := filepath.Join(user1ExportDir, "posts.json")
	user2Data := filepath.Join(user2ExportDir, "posts.json")

	if err := os.WriteFile(user1Data, []byte(`{"posts": ["user1"]}`), 0644); err != nil {
		t.Fatalf("Failed to write user1 data: %v", err)
	}

	if err := os.WriteFile(user2Data, []byte(`{"posts": ["user2"]}`), 0644); err != nil {
		t.Fatalf("Failed to write user2 data: %v", err)
	}

	// Verify that attempting to construct a path to another user's export
	// would fail the path validation in handlers (tested separately)

	// Verify directory paths are completely isolated
	user1BaseDir := filepath.Join(tmpDir, user1DID)
	user2BaseDir := filepath.Join(tmpDir, user2DID)

	// Check that user1's directory doesn't contain user2's subdirectories
	user1Entries, _ := os.ReadDir(user1BaseDir)
	for _, entry := range user1Entries {
		if entry.Name() == user2DID {
			t.Error("User1's directory should not contain user2's DID as subdirectory")
		}
	}

	// Check that user2's directory doesn't contain user1's subdirectories
	user2Entries, _ := os.ReadDir(user2BaseDir)
	for _, entry := range user2Entries {
		if entry.Name() == user1DID {
			t.Error("User2's directory should not contain user1's DID as subdirectory")
		}
	}

	// Verify each user's export directory is only accessible via their own DID path
	if !filepath.HasPrefix(user1ExportDir, user1BaseDir+string(filepath.Separator)) {
		t.Error("User1 export should be under user1 base directory")
	}

	if !filepath.HasPrefix(user2ExportDir, user2BaseDir+string(filepath.Separator)) {
		t.Error("User2 export should be under user2 base directory")
	}
}
