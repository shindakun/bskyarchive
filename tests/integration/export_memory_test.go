package integration

import (
	"database/sql"
	"os"
	"runtime"
	"testing"

	"github.com/shindakun/bskyarchive/internal/exporter"
	"github.com/shindakun/bskyarchive/internal/models"
	_ "modernc.org/sqlite"
)

// TestExportBatching_Memory tests memory usage during large export
// This test verifies that memory usage stays below 500MB for 10k posts
// Note: For production validation with 50k+ posts, run manually with larger dataset
func TestExportBatching_Memory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory profiling test in short mode")
	}

	// Check if test database exists (using 10k for faster automated testing)
	dbPath := "/Users/steve/go/src/github.com/shindakun/bskyarchive/tests/fixtures/test_10k.db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Skipf("Test database not found at %s. Run: go run tests/fixtures/generate_test_db.go -posts 10000 -output tests/fixtures/test_10k.db", dbPath)
	}

	// Open the test database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}
	defer db.Close()

	// Create output directory
	outputDir := tempBatchingDir(t)

	// Force garbage collection and get baseline memory
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

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
	progressChan := make(chan models.ExportProgress, 50)

	// Collect progress in background with logging
	done := make(chan bool)
	go func() {
		for progress := range progressChan {
			// Log progress updates to track export status
			t.Logf("Progress: Status=%s, PostsTotal=%d, PostsProcessed=%d",
				progress.Status, progress.PostsTotal, progress.PostsProcessed)
		}
		done <- true
	}()

	// Track peak memory during export
	peakMemory := uint64(0)
	memSampler := make(chan bool)
	go func() {
		ticker := runtime.MemStats{}
		for {
			select {
			case <-memSampler:
				return
			default:
				runtime.ReadMemStats(&ticker)
				allocMB := ticker.Alloc / 1024 / 1024
				if allocMB > peakMemory {
					peakMemory = allocMB
				}
				// Sleep to avoid busy-waiting
				runtime.Gosched()
			}
		}
	}()

	// Run export
	err = exporter.Run(db, job, progressChan)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Stop memory sampling
	close(memSampler)

	// Wait for progress collection
	<-done

	// Get final memory stats
	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate memory usage
	allocatedMB := memAfter.Alloc / 1024 / 1024
	peakAllocMB := memAfter.TotalAlloc / 1024 / 1024

	// Verify export completed
	if job.Progress.Status != models.ExportStatusCompleted {
		t.Errorf("Expected status=Completed, got %s", job.Progress.Status)
	}

	// Verify post count
	expectedCount := 10000
	if job.Progress.PostsTotal != expectedCount {
		t.Errorf("Expected PostsTotal=%d, got %d", expectedCount, job.Progress.PostsTotal)
	}

	// Memory threshold: 500MB
	memoryThresholdMB := uint64(500)

	// Report memory usage
	t.Logf("Memory Statistics:")
	t.Logf("  - Current allocated: %d MB", allocatedMB)
	t.Logf("  - Peak allocated: %d MB", peakMemory)
	t.Logf("  - Total allocated: %d MB", peakAllocMB)
	t.Logf("  - Memory threshold: %d MB", memoryThresholdMB)

	// Verify memory usage is below threshold
	if peakMemory > memoryThresholdMB {
		t.Errorf("Peak memory usage %d MB exceeds threshold of %d MB", peakMemory, memoryThresholdMB)
	} else {
		t.Logf("✓ Memory usage %d MB is below threshold of %d MB", peakMemory, memoryThresholdMB)
	}

	// Verify all posts were exported
	if job.Progress.PostsProcessed != expectedCount {
		t.Errorf("Expected PostsProcessed=%d, got %d", expectedCount, job.Progress.PostsProcessed)
	} else {
		t.Logf("✓ Successfully exported %d posts", job.Progress.PostsProcessed)
	}
}
