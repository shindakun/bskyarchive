package exporter

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
	_ "modernc.org/sqlite"
)

// setupRunTestDB creates an in-memory database for testing Run() function
func setupRunTestDB(t *testing.T) *sql.DB {
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

// insertRunTestPosts inserts test posts into database
func insertRunTestPosts(t *testing.T, db *sql.DB, did string, count int) {
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
			t.Fatalf("Failed to insert test post %d: %v", i, err)
		}
	}
}

// tempRunTestDir creates a temporary directory for test exports
func tempRunTestDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "exporter_run_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	t.Cleanup(func() {
		os.RemoveAll(dir)
	})

	return dir
}

// TestRun_CountQuery verifies that the COUNT query works correctly
// and that the total post count is properly tracked
func TestRun_CountQuery(t *testing.T) {
	db := setupRunTestDB(t)
	defer db.Close()

	did := "did:plc:test123"
	postCount := 2500

	// Insert test posts
	insertRunTestPosts(t, db, did, postCount)

	// Create export job
	outputDir := tempRunTestDir(t)
	job := &models.ExportJob{
		Options: models.ExportOptions{
			DID:       did,
			Format:    models.ExportFormatJSON,
			OutputDir: outputDir,
		},
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel
	progressChan := make(chan models.ExportProgress, 10)

	// Run export
	err := Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify PostsTotal was set correctly from COUNT query
	if job.Progress.PostsTotal != postCount {
		t.Errorf("Expected PostsTotal=%d, got %d", postCount, job.Progress.PostsTotal)
	}

	// Verify PostsProcessed matches total
	if job.Progress.PostsProcessed != postCount {
		t.Errorf("Expected PostsProcessed=%d, got %d", postCount, job.Progress.PostsProcessed)
	}

	t.Logf("✓ COUNT query correctly identified %d posts", postCount)
}

// TestRun_BatchedExportJSON verifies that batched JSON export works end-to-end
func TestRun_BatchedExportJSON(t *testing.T) {
	db := setupRunTestDB(t)
	defer db.Close()

	did := "did:plc:test123"
	postCount := 3500 // More than 3 batches (1000 each)

	// Insert test posts
	insertRunTestPosts(t, db, did, postCount)

	// Create export job
	outputDir := tempRunTestDir(t)
	job := &models.ExportJob{
		Options: models.ExportOptions{
			DID:       did,
			Format:    models.ExportFormatJSON,
			OutputDir: outputDir,
		},
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel
	progressChan := make(chan models.ExportProgress, 10)

	// Run export
	err := Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify export directory was created
	if job.ExportDir == "" {
		t.Fatal("ExportDir not set")
	}

	// Verify JSON file exists
	jsonFile := filepath.Join(job.ExportDir, "posts.json")
	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		t.Fatalf("Expected JSON file at %s, but it doesn't exist", jsonFile)
	}

	// Verify status is completed
	if job.Progress.Status != models.ExportStatusCompleted {
		t.Errorf("Expected status=%s, got %s", models.ExportStatusCompleted, job.Progress.Status)
	}

	t.Logf("✓ Batched JSON export completed successfully with %d posts", postCount)
}

// TestRun_BatchedExportCSV verifies that batched CSV export works end-to-end
func TestRun_BatchedExportCSV(t *testing.T) {
	db := setupRunTestDB(t)
	defer db.Close()

	did := "did:plc:test123"
	postCount := 3500

	// Insert test posts
	insertRunTestPosts(t, db, did, postCount)

	// Create export job
	outputDir := tempRunTestDir(t)
	job := &models.ExportJob{
		Options: models.ExportOptions{
			DID:       did,
			Format:    models.ExportFormatCSV,
			OutputDir: outputDir,
		},
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel
	progressChan := make(chan models.ExportProgress, 10)

	// Run export
	err := Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify CSV file exists
	csvFile := filepath.Join(job.ExportDir, "posts.csv")
	if _, err := os.Stat(csvFile); os.IsNotExist(err) {
		t.Fatalf("Expected CSV file at %s, but it doesn't exist", csvFile)
	}

	// Verify status is completed
	if job.Progress.Status != models.ExportStatusCompleted {
		t.Errorf("Expected status=%s, got %s", models.ExportStatusCompleted, job.Progress.Status)
	}

	t.Logf("✓ Batched CSV export completed successfully with %d posts", postCount)
}

// TestRun_ProgressUpdates verifies that progress updates are sent correctly
func TestRun_ProgressUpdates(t *testing.T) {
	db := setupRunTestDB(t)
	defer db.Close()

	did := "did:plc:test123"
	postCount := 2500

	// Insert test posts
	insertRunTestPosts(t, db, did, postCount)

	// Create export job
	outputDir := tempRunTestDir(t)
	job := &models.ExportJob{
		Options: models.ExportOptions{
			DID:       did,
			Format:    models.ExportFormatJSON,
			OutputDir: outputDir,
		},
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel with buffer
	progressChan := make(chan models.ExportProgress, 20)

	// Collect progress updates in a goroutine
	progressUpdates := []models.ExportProgress{}
	done := make(chan bool)
	go func() {
		for progress := range progressChan {
			progressUpdates = append(progressUpdates, progress)
		}
		done <- true
	}()

	// Run export
	err := Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Wait for progress collection to complete
	<-done

	// Verify we received progress updates
	if len(progressUpdates) == 0 {
		t.Fatal("Expected progress updates, but received none")
	}

	// Verify first update is Running status
	if progressUpdates[0].Status != models.ExportStatusRunning {
		t.Errorf("Expected first status=Running, got %s", progressUpdates[0].Status)
	}

	// Verify last update is Completed status
	lastUpdate := progressUpdates[len(progressUpdates)-1]
	if lastUpdate.Status != models.ExportStatusCompleted {
		t.Errorf("Expected last status=Completed, got %s", lastUpdate.Status)
	}

	// Verify PostsTotal was set
	foundPostsTotal := false
	for _, update := range progressUpdates {
		if update.PostsTotal == postCount {
			foundPostsTotal = true
			break
		}
	}
	if !foundPostsTotal {
		t.Errorf("Expected to find PostsTotal=%d in progress updates", postCount)
	}

	// Verify PostsProcessed was updated
	foundPostsProcessed := false
	for _, update := range progressUpdates {
		if update.PostsProcessed == postCount {
			foundPostsProcessed = true
			break
		}
	}
	if !foundPostsProcessed {
		t.Errorf("Expected to find PostsProcessed=%d in progress updates", postCount)
	}

	t.Logf("✓ Received %d progress updates with correct status transitions", len(progressUpdates))
}

// TestRun_EmptyArchive verifies handling of empty archives
func TestRun_EmptyArchive(t *testing.T) {
	db := setupRunTestDB(t)
	defer db.Close()

	did := "did:plc:test123"
	// Don't insert any posts

	// Create export job
	outputDir := tempRunTestDir(t)
	job := &models.ExportJob{
		Options: models.ExportOptions{
			DID:       did,
			Format:    models.ExportFormatJSON,
			OutputDir: outputDir,
		},
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel
	progressChan := make(chan models.ExportProgress, 10)

	// Run export
	err := Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify PostsTotal is 0
	if job.Progress.PostsTotal != 0 {
		t.Errorf("Expected PostsTotal=0, got %d", job.Progress.PostsTotal)
	}

	// Verify status is completed (not an error)
	if job.Progress.Status != models.ExportStatusCompleted {
		t.Errorf("Expected status=Completed, got %s", job.Progress.Status)
	}

	t.Logf("✓ Empty archive handled gracefully")
}

// TestRun_DateRangeFilter verifies COUNT query respects date range filters
func TestRun_DateRangeFilter(t *testing.T) {
	db := setupRunTestDB(t)
	defer db.Close()

	did := "did:plc:test123"

	// Insert posts with different dates
	baseTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	stmt, err := db.Prepare(`
		INSERT INTO posts (uri, cid, did, text, created_at, indexed_at, reply_parent, embed_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		t.Fatalf("Failed to prepare insert: %v", err)
	}
	defer stmt.Close()

	// Insert 1000 posts in January
	for i := 0; i < 1000; i++ {
		createdAt := baseTime.Add(time.Duration(i) * time.Hour)
		uri := fmt.Sprintf("at://%s/app.bsky.feed.post/%05d", did, i)
		_, err := stmt.Exec(uri, fmt.Sprintf("cid%d", i), did, fmt.Sprintf("Post %d", i), createdAt, createdAt, "", "")
		if err != nil {
			t.Fatalf("Failed to insert post: %v", err)
		}
	}

	// Insert 1000 posts in February
	febTime := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	for i := 1000; i < 2000; i++ {
		createdAt := febTime.Add(time.Duration(i-1000) * time.Hour)
		uri := fmt.Sprintf("at://%s/app.bsky.feed.post/%05d", did, i)
		_, err := stmt.Exec(uri, fmt.Sprintf("cid%d", i), did, fmt.Sprintf("Post %d", i), createdAt, createdAt, "", "")
		if err != nil {
			t.Fatalf("Failed to insert post: %v", err)
		}
	}

	// Create export job with date range filter (January only)
	outputDir := tempRunTestDir(t)
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC)

	job := &models.ExportJob{
		Options: models.ExportOptions{
			DID:       did,
			Format:    models.ExportFormatJSON,
			OutputDir: outputDir,
			DateRange: &models.DateRange{
				StartDate: startDate,
				EndDate:   endDate,
			},
		},
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel
	progressChan := make(chan models.ExportProgress, 10)

	// Run export
	err = Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Run() failed: %v", err)
	}

	// Verify only January posts were counted (should be around 744 posts = 31 days * 24 hours)
	expectedCount := 31 * 24 // 744 posts
	if job.Progress.PostsTotal < expectedCount-10 || job.Progress.PostsTotal > expectedCount+10 {
		t.Errorf("Expected PostsTotal around %d, got %d", expectedCount, job.Progress.PostsTotal)
	}

	t.Logf("✓ Date range filter correctly limited export to %d posts", job.Progress.PostsTotal)
}
