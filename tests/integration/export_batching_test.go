package integration

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/shindakun/bskyarchive/internal/exporter"
	"github.com/shindakun/bskyarchive/internal/models"
	_ "modernc.org/sqlite"
)

// TestExportBatching_JSON tests batched JSON export with 10k posts
func TestExportBatching_JSON(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Open the pre-generated test database
	dbPath := "/Users/steve/go/src/github.com/shindakun/bskyarchive/tests/fixtures/test_10k.db"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create output directory
	outputDir := tempBatchingDir(t)

	// Create export job
	job := &models.ExportJob{
		Options: models.ExportOptions{
			DID:       "did:plc:test123",
			Format:    models.ExportFormatJSON,
			OutputDir: outputDir,
		},
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel
	progressChan := make(chan models.ExportProgress, 20)

	// Run export
	err = exporter.Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify export directory was created
	if job.ExportDir == "" {
		t.Fatal("ExportDir not set")
	}

	// Find the JSON file
	entries, err := os.ReadDir(job.ExportDir)
	if err != nil {
		t.Fatalf("Failed to read export dir: %v", err)
	}

	var jsonFile string
	for _, entry := range entries {
		if entry.Name() == "posts.json" {
			jsonFile = filepath.Join(job.ExportDir, entry.Name())
			break
		}
	}

	if jsonFile == "" {
		t.Fatal("posts.json not found in export directory")
	}

	// Read and parse JSON
	data, err := os.ReadFile(jsonFile)
	if err != nil {
		t.Fatalf("Failed to read JSON file: %v", err)
	}

	var posts []models.Post
	if err := json.Unmarshal(data, &posts); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify post count
	expectedCount := 10000
	if len(posts) != expectedCount {
		t.Errorf("Expected %d posts in JSON, got %d", expectedCount, len(posts))
	}

	// Verify status
	if job.Progress.Status != models.ExportStatusCompleted {
		t.Errorf("Expected status=Completed, got %s", job.Progress.Status)
	}

	t.Logf("✓ Successfully exported and verified %d posts in JSON format", len(posts))
}

// TestExportBatching_CSV tests batched CSV export with 10k posts
func TestExportBatching_CSV(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Open the pre-generated test database
	dbPath := "/Users/steve/go/src/github.com/shindakun/bskyarchive/tests/fixtures/test_10k.db"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create output directory
	outputDir := tempBatchingDir(t)

	// Create export job
	job := &models.ExportJob{
		Options: models.ExportOptions{
			DID:       "did:plc:test123",
			Format:    models.ExportFormatCSV,
			OutputDir: outputDir,
		},
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel
	progressChan := make(chan models.ExportProgress, 20)

	// Run export
	err = exporter.Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Find the CSV file
	entries, err := os.ReadDir(job.ExportDir)
	if err != nil {
		t.Fatalf("Failed to read export dir: %v", err)
	}

	var csvFile string
	for _, entry := range entries {
		if entry.Name() == "posts.csv" {
			csvFile = filepath.Join(job.ExportDir, entry.Name())
			break
		}
	}

	if csvFile == "" {
		t.Fatal("posts.csv not found in export directory")
	}

	// Read and count CSV rows
	file, err := os.Open(csvFile)
	if err != nil {
		t.Fatalf("Failed to open CSV file: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read CSV: %v", err)
	}

	// Verify row count (subtract 1 for header)
	expectedCount := 10000
	actualCount := len(records) - 1 // -1 for header row
	if actualCount != expectedCount {
		t.Errorf("Expected %d rows in CSV, got %d", expectedCount, actualCount)
	}

	// Verify status
	if job.Progress.Status != models.ExportStatusCompleted {
		t.Errorf("Expected status=Completed, got %s", job.Progress.Status)
	}

	t.Logf("✓ Successfully exported and verified %d rows in CSV format", actualCount)
}

// TestExportBatching_Progress tests progress update frequency
func TestExportBatching_Progress(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Open the pre-generated test database
	dbPath := "/Users/steve/go/src/github.com/shindakun/bskyarchive/tests/fixtures/test_10k.db"
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create output directory
	outputDir := tempBatchingDir(t)

	// Create export job
	job := &models.ExportJob{
		Options: models.ExportOptions{
			DID:       "did:plc:test123",
			Format:    models.ExportFormatJSON,
			OutputDir: outputDir,
		},
		Progress: models.ExportProgress{
			Status: models.ExportStatusQueued,
		},
	}

	// Create progress channel with buffer
	progressChan := make(chan models.ExportProgress, 50)

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
	err = exporter.Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
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

	// Verify PostsTotal was set to 10000
	foundPostsTotal := false
	for _, update := range progressUpdates {
		if update.PostsTotal == 10000 {
			foundPostsTotal = true
			break
		}
	}
	if !foundPostsTotal {
		t.Error("Expected to find PostsTotal=10000 in progress updates")
	}

	// Verify PostsProcessed reached 10000
	foundPostsProcessed := false
	for _, update := range progressUpdates {
		if update.PostsProcessed == 10000 {
			foundPostsProcessed = true
			break
		}
	}
	if !foundPostsProcessed {
		t.Error("Expected to find PostsProcessed=10000 in progress updates")
	}

	t.Logf("✓ Received %d progress updates with correct transitions", len(progressUpdates))
	t.Logf("  - First status: %s", progressUpdates[0].Status)
	t.Logf("  - Last status: %s", lastUpdate.Status)
	t.Logf("  - PostsTotal: %d", lastUpdate.PostsTotal)
	t.Logf("  - PostsProcessed: %d", lastUpdate.PostsProcessed)
}

// setupBatchingTestDB creates an in-memory test database with sample posts
// Note: Different from setupTestDB in export_integration_test.go to avoid conflicts
func setupBatchingTestDB(t *testing.T, postCount int) *sql.DB {
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

	// Insert test posts (will be implemented when tests are activated)
	// For now, just return the empty database structure

	return db
}

// cleanupBatchingTestFiles removes test output files
func cleanupBatchingTestFiles(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		if err := os.RemoveAll(path); err != nil && !os.IsNotExist(err) {
			t.Logf("Warning: Failed to cleanup %s: %v", path, err)
		}
	}
}

// tempBatchingDir creates a temporary directory for test outputs
func tempBatchingDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "bskyarchive-batching-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

// verifyBatchingFileExists checks if a file exists and returns its path
func verifyBatchingFileExists(t *testing.T, dir, filename string) string {
	t.Helper()
	path := filepath.Join(dir, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("Expected file does not exist: %s", path)
	}
	return path
}
