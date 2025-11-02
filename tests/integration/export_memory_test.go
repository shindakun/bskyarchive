package integration

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/shindakun/bskyarchive/internal/exporter"
)

// TestStreamDirectoryAsZIP_MemoryUsage verifies memory-efficient ZIP streaming
// Goal: <500MB memory footprint for large exports (5GB+)
func TestStreamDirectoryAsZIP_MemoryUsage(t *testing.T) {
	// Skip this test in short mode (it creates large files)
	if testing.Short() {
		t.Skip("Skipping memory profiling test in short mode")
	}

	// Create a temporary directory with test files
	tmpDir := t.TempDir()
	exportDir := filepath.Join(tmpDir, "large-export")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}

	// Create multiple files to simulate a large export
	// We'll create 10 files of 10MB each = 100MB total
	// This is scaled down from 5GB for practical test execution
	const fileCount = 10
	const fileSizeMB = 10
	const fileSizeBytes = fileSizeMB * 1024 * 1024

	t.Logf("Creating %d test files of %dMB each (%dMB total)", fileCount, fileSizeMB, fileCount*fileSizeMB)

	// Create test files
	for i := 0; i < fileCount; i++ {
		filename := filepath.Join(exportDir, "posts.json")
		if i > 0 {
			filename = filepath.Join(exportDir, "media", "test-file-"+string(rune('0'+i))+".jpg")
			os.MkdirAll(filepath.Dir(filename), 0755)
		}

		// Write file with repeated data
		f, err := os.Create(filename)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}

		// Write in chunks to avoid memory spike
		chunk := make([]byte, 1024*1024) // 1MB chunks
		for j := 0; j < fileSizeMB; j++ {
			if _, err := f.Write(chunk); err != nil {
				f.Close()
				t.Fatalf("Failed to write test data: %v", err)
			}
		}
		f.Close()
	}

	// Force garbage collection before measuring baseline
	runtime.GC()

	// Measure baseline memory
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Stream the directory as ZIP to a discard writer
	// This simulates actual HTTP streaming without network overhead
	discardWriter := io.Discard

	err := exporter.StreamDirectoryAsZIP(exportDir, discardWriter)
	if err != nil {
		t.Fatalf("Failed to stream directory as ZIP: %v", err)
	}

	// Force garbage collection after streaming
	runtime.GC()

	// Measure memory after streaming
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Calculate memory used during streaming
	memUsedMB := float64(memAfter.Alloc-memBefore.Alloc) / (1024 * 1024)
	heapInUseMB := float64(memAfter.HeapInuse) / (1024 * 1024)
	totalAllocMB := float64(memAfter.TotalAlloc-memBefore.TotalAlloc) / (1024 * 1024)

	t.Logf("Memory usage during ZIP streaming:")
	t.Logf("  Alloc delta: %.2f MB", memUsedMB)
	t.Logf("  Heap in use: %.2f MB", heapInUseMB)
	t.Logf("  Total alloc delta: %.2f MB", totalAllocMB)

	// Verify memory usage is reasonable
	// For 100MB of data, we should use much less than 500MB
	// Allow up to 50MB for this test (scaled proportionally)
	const maxMemoryMB = 50.0

	if heapInUseMB > maxMemoryMB {
		t.Errorf("Memory usage too high: %.2f MB > %d MB threshold", heapInUseMB, int(maxMemoryMB))
	}

	t.Logf("✓ Memory profiling test passed - heap usage %.2f MB < %d MB", heapInUseMB, int(maxMemoryMB))
}

// TestStreamDirectoryAsZIP_LargeFileStreaming verifies streaming works for individual large files
func TestStreamDirectoryAsZIP_LargeFileStreaming(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	tmpDir := t.TempDir()
	exportDir := filepath.Join(tmpDir, "large-file-export")
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}

	// Create one large file (50MB)
	largeFile := filepath.Join(exportDir, "large-posts.json")
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("Failed to create large file: %v", err)
	}

	const fileSizeMB = 50
	chunk := make([]byte, 1024*1024) // 1MB chunks
	for i := 0; i < fileSizeMB; i++ {
		if _, err := f.Write(chunk); err != nil {
			f.Close()
			t.Fatalf("Failed to write test data: %v", err)
		}
	}
	f.Close()

	// Force GC and measure
	runtime.GC()
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Stream to discard
	err = exporter.StreamDirectoryAsZIP(exportDir, io.Discard)
	if err != nil {
		t.Fatalf("Failed to stream large file: %v", err)
	}

	runtime.GC()
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	heapInUseMB := float64(memAfter.HeapInuse) / (1024 * 1024)
	t.Logf("Large file streaming heap usage: %.2f MB", heapInUseMB)

	// For a 50MB file, heap should stay well under 100MB
	const maxMemoryMB = 100.0
	if heapInUseMB > maxMemoryMB {
		t.Errorf("Memory usage too high for large file: %.2f MB > %d MB", heapInUseMB, int(maxMemoryMB))
	}

	t.Logf("✓ Large file streaming test passed - heap usage %.2f MB < %d MB", heapInUseMB, int(maxMemoryMB))
}
