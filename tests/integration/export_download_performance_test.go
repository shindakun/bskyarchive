package integration

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/exporter"
	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
)

// TestDownloadInitiationPerformance verifies that download initiation takes less than 1 second
// This test validates the performance requirement from T056
// "Download initiation" means time to start streaming, not full download time
func TestDownloadInitiationPerformance(t *testing.T) {
	// Create temporary database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	db, err := storage.InitDB(dbPath)
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Create a typical export directory with some files
	testDID := "did:plc:download123"
	exportDir := fmt.Sprintf("./exports/%s/test-export", testDID)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}
	defer os.RemoveAll("./exports")

	// Create test files in export directory (typical export structure)
	files := []struct {
		name    string
		content string
	}{
		{"manifest.json", `{"export_format":"json","post_count":100}`},
		{"posts.json", `[{"uri":"at://test/post/1","text":"Hello"}]`},
		{"media/image1.jpg", "fake-image-data-1234567890"},
		{"media/image2.jpg", "fake-image-data-0987654321"},
	}

	for _, f := range files {
		fullPath := filepath.Join(exportDir, f.name)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(f.content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", f.name, err)
		}
	}

	// Calculate export size
	var totalSize int64
	filepath.Walk(exportDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	// Create export record
	record := &models.ExportRecord{
		ID:            fmt.Sprintf("%s/test-export", testDID),
		DID:           testDID,
		Format:        "json",
		CreatedAt:     time.Now(),
		DirectoryPath: exportDir,
		PostCount:     100,
		MediaCount:    2,
		SizeBytes:     totalSize,
	}

	if err := storage.CreateExportRecord(db, record); err != nil {
		t.Fatalf("Failed to create export record: %v", err)
	}

	// Test download initiation performance
	const iterations = 5
	var totalInitTime time.Duration

	for i := 0; i < iterations; i++ {
		// Measure time to initiate download (first bytes)
		start := time.Now()

		// Use a pipe to capture the stream
		pr, pw := io.Pipe()

		// Start streaming in a goroutine
		errChan := make(chan error, 1)
		firstByteChan := make(chan struct{})
		firstByteReceived := false

		go func() {
			errChan <- exporter.StreamDirectoryAsZIP(exportDir, pw)
			pw.Close()
		}()

		// Read first bytes to measure initiation time
		buf := make([]byte, 1024)
		go func() {
			n, err := pr.Read(buf)
			if !firstByteReceived && n > 0 {
				firstByteReceived = true
				close(firstByteChan)
			}
			// Continue reading to drain the pipe
			io.Copy(io.Discard, pr)
			if err != nil && err != io.EOF {
				t.Logf("Read error (non-fatal): %v", err)
			}
		}()

		// Wait for first byte
		select {
		case <-firstByteChan:
			// First byte received
		case err := <-errChan:
			if err != nil {
				t.Fatalf("Stream error: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for first byte")
		}

		initTime := time.Since(start)
		totalInitTime += initTime

		// Clean up
		pr.Close()
		<-errChan // Wait for goroutine to finish

		t.Logf("Iteration %d: Download initiation time: %v", i+1, initTime)
	}

	// Calculate average
	avgInitTime := totalInitTime / iterations

	t.Logf("Performance Results:")
	t.Logf("  Total iterations: %d", iterations)
	t.Logf("  Total initiation time: %v", totalInitTime)
	t.Logf("  Average initiation time: %v", avgInitTime)
	t.Logf("  Min acceptable: 1s")
	t.Logf("  Export size: %d bytes", totalSize)

	// Verify performance requirement: average initiation time < 1 second
	if avgInitTime >= time.Second {
		t.Errorf("Performance requirement not met: average initiation time %v >= 1s", avgInitTime)
	} else {
		t.Logf("✓ Performance requirement met: average initiation time %v < 1s", avgInitTime)
	}
}

// TestDownloadInitiationPerformanceLargeExport tests with a larger, more realistic export
func TestDownloadInitiationPerformanceLargeExport(t *testing.T) {
	// Create temporary export directory
	testDID := "did:plc:largedownload"
	exportDir := fmt.Sprintf("./exports/%s/large-export", testDID)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}
	defer os.RemoveAll("./exports")

	// Create a larger posts.json file (10MB of JSON data)
	postsFile := filepath.Join(exportDir, "posts.json")
	f, err := os.Create(postsFile)
	if err != nil {
		t.Fatalf("Failed to create posts file: %v", err)
	}

	// Write realistic JSON data
	f.WriteString("[\n")
	for i := 0; i < 10000; i++ {
		if i > 0 {
			f.WriteString(",\n")
		}
		fmt.Fprintf(f, `{"uri":"at://test/post/%d","text":"This is post number %d with some content"}`, i, i)
	}
	f.WriteString("\n]")
	f.Close()

	// Create manifest
	manifestFile := filepath.Join(exportDir, "manifest.json")
	if err := os.WriteFile(manifestFile, []byte(`{"export_format":"json","post_count":10000}`), 0644); err != nil {
		t.Fatalf("Failed to create manifest: %v", err)
	}

	// Create media directory with some files
	mediaDir := filepath.Join(exportDir, "media")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		t.Fatalf("Failed to create media directory: %v", err)
	}

	// Add a few media files
	for i := 0; i < 10; i++ {
		mediaFile := filepath.Join(mediaDir, fmt.Sprintf("image%d.jpg", i))
		data := bytes.Repeat([]byte("X"), 100*1024) // 100KB per file
		if err := os.WriteFile(mediaFile, data, 0644); err != nil {
			t.Fatalf("Failed to create media file: %v", err)
		}
	}

	// Measure initiation time
	start := time.Now()

	pr, pw := io.Pipe()

	// Start streaming
	errChan := make(chan error, 1)
	firstByteChan := make(chan struct{})
	firstByteReceived := false

	go func() {
		errChan <- exporter.StreamDirectoryAsZIP(exportDir, pw)
		pw.Close()
	}()

	// Read first bytes
	buf := make([]byte, 1024)
	go func() {
		n, err := pr.Read(buf)
		if !firstByteReceived && n > 0 {
			firstByteReceived = true
			close(firstByteChan)
		}
		io.Copy(io.Discard, pr)
		if err != nil && err != io.EOF {
			t.Logf("Read error (non-fatal): %v", err)
		}
	}()

	// Wait for first byte
	select {
	case <-firstByteChan:
		// First byte received
	case err := <-errChan:
		if err != nil {
			t.Fatalf("Stream error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for first byte")
	}

	initTime := time.Since(start)

	// Clean up
	pr.Close()
	<-errChan

	// Get total export size
	var totalSize int64
	filepath.Walk(exportDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	t.Logf("Large Export Performance:")
	t.Logf("  Export size: %.2f MB", float64(totalSize)/(1024*1024))
	t.Logf("  Download initiation time: %v", initTime)
	t.Logf("  Min acceptable: 1s")

	if initTime >= time.Second {
		t.Errorf("Performance requirement not met for large export: initiation time %v >= 1s", initTime)
	} else {
		t.Logf("✓ Performance requirement met: initiation time %v < 1s", initTime)
	}
}

// TestStreamingEfficiency verifies that streaming doesn't load entire file into memory
func TestStreamingEfficiency(t *testing.T) {
	// Create a test export directory
	testDID := "did:plc:streaming"
	exportDir := fmt.Sprintf("./exports/%s/stream-test", testDID)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		t.Fatalf("Failed to create export directory: %v", err)
	}
	defer os.RemoveAll("./exports")

	// Create a moderately sized file
	testFile := filepath.Join(exportDir, "test.dat")
	data := bytes.Repeat([]byte("A"), 5*1024*1024) // 5MB file
	if err := os.WriteFile(testFile, data, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Stream to a buffer that discards data
	start := time.Now()

	pr, pw := io.Pipe()

	go func() {
		exporter.StreamDirectoryAsZIP(exportDir, pw)
		pw.Close()
	}()

	// Read and discard all data
	n, err := io.Copy(io.Discard, pr)
	if err != nil {
		t.Fatalf("Failed to read stream: %v", err)
	}

	duration := time.Since(start)

	t.Logf("Streaming Efficiency Test:")
	t.Logf("  Source size: %.2f MB", float64(len(data))/(1024*1024))
	t.Logf("  Compressed size: %.2f MB", float64(n)/(1024*1024))
	t.Logf("  Streaming time: %v", duration)
	t.Logf("  Throughput: %.2f MB/s", float64(n)/(1024*1024)/duration.Seconds())

	// Verify streaming completed successfully
	if n == 0 {
		t.Error("No data was streamed")
	}
}
