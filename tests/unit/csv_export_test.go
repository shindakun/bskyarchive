package unit

import (
	"bytes"
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/exporter"
	"github.com/shindakun/bskyarchive/internal/models"
)

// TestExportToCSV tests basic CSV export functionality
func TestExportToCSV(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_export.csv")

	// Create test posts with various data
	testPosts := []models.Post{
		{
			URI:         "at://did:plc:test123/app.bsky.feed.post/post1",
			CID:         "bafytest123",
			DID:         "did:plc:test123",
			Text:        "First test post for CSV export",
			CreatedAt:   time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC),
			IndexedAt:   time.Date(2025, 1, 15, 10, 35, 0, 0, time.UTC),
			LikeCount:   10,
			RepostCount: 5,
			ReplyCount:  2,
			QuoteCount:  1,
			HasMedia:    false,
			IsReply:     false,
		},
		{
			URI:         "at://did:plc:test123/app.bsky.feed.post/post2",
			CID:         "bafytest456",
			DID:         "did:plc:test123",
			Text:        "Post with emoji 🚀 and unicode ✨",
			CreatedAt:   time.Date(2025, 1, 16, 14, 20, 0, 0, time.UTC),
			IndexedAt:   time.Date(2025, 1, 16, 14, 25, 0, 0, time.UTC),
			LikeCount:   25,
			RepostCount: 10,
			ReplyCount:  5,
			QuoteCount:  3,
			HasMedia:    true,
			EmbedType:   "images",
			IsReply:     false,
		},
		{
			URI:         "at://did:plc:test123/app.bsky.feed.post/post3",
			CID:         "bafytest789",
			DID:         "did:plc:test123",
			Text:        "Post with \"quotes\" and, commas",
			CreatedAt:   time.Date(2025, 1, 17, 9, 15, 0, 0, time.UTC),
			IndexedAt:   time.Date(2025, 1, 17, 9, 20, 0, 0, time.UTC),
			LikeCount:   5,
			RepostCount: 2,
			ReplyCount:  0,
			QuoteCount:  0,
			HasMedia:    false,
			IsReply:     true,
			ReplyParent: "at://did:plc:other/app.bsky.feed.post/parent",
		},
	}

	// Export to CSV
	err := exporter.ExportToCSV(testPosts, outputPath)
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("CSV file was not created: %s", outputPath)
	}

	// Read and parse the CSV file
	fileData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	// Verify UTF-8 BOM is present
	if !bytes.HasPrefix(fileData, []byte{0xEF, 0xBB, 0xBF}) {
		t.Error("CSV file is missing UTF-8 BOM")
	}

	// Remove BOM for parsing
	csvData := bytes.TrimPrefix(fileData, []byte{0xEF, 0xBB, 0xBF})

	// Parse CSV
	reader := csv.NewReader(bytes.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Verify header row (15 columns)
	if len(records) < 1 {
		t.Fatal("CSV has no header row")
	}

	header := records[0]
	expectedHeaders := []string{
		"URI", "CID", "DID", "Text", "CreatedAt",
		"LikeCount", "RepostCount", "ReplyCount", "QuoteCount",
		"IsReply", "ReplyParent", "HasMedia", "MediaFiles", "EmbedType", "IndexedAt",
	}

	if len(header) != len(expectedHeaders) {
		t.Errorf("Expected %d columns, got %d", len(expectedHeaders), len(header))
	}

	for i, expected := range expectedHeaders {
		if i < len(header) && header[i] != expected {
			t.Errorf("Column %d: expected %s, got %s", i, expected, header[i])
		}
	}

	// Verify data rows (should have 3 posts + 1 header = 4 rows total)
	if len(records) != 4 {
		t.Errorf("Expected 4 rows (1 header + 3 data), got %d", len(records))
	}

	// Verify first post data
	if len(records) > 1 {
		row1 := records[1]
		if len(row1) >= 4 {
			if row1[0] != testPosts[0].URI {
				t.Errorf("Row 1 URI: expected %s, got %s", testPosts[0].URI, row1[0])
			}
			if row1[1] != testPosts[0].CID {
				t.Errorf("Row 1 CID: expected %s, got %s", testPosts[0].CID, row1[1])
			}
			if row1[3] != testPosts[0].Text {
				t.Errorf("Row 1 Text: expected %s, got %s", testPosts[0].Text, row1[3])
			}
		}
	}

	// Verify second post has emoji preserved
	if len(records) > 2 {
		row2 := records[2]
		if len(row2) >= 4 {
			if !strings.Contains(row2[3], "🚀") {
				t.Error("Emoji not preserved in CSV")
			}
			if !strings.Contains(row2[3], "✨") {
				t.Error("Unicode character not preserved in CSV")
			}
		}
	}

	// Verify third post has special characters properly escaped
	if len(records) > 3 {
		row3 := records[3]
		if len(row3) >= 4 {
			// CSV library should handle quotes and commas automatically
			if !strings.Contains(row3[3], "quotes") || !strings.Contains(row3[3], "commas") {
				t.Error("Special characters not properly escaped in CSV")
			}
			// Verify reply parent is formatted correctly
			if len(row3) >= 11 {
				if row3[9] != "true" {
					t.Errorf("IsReply should be 'true', got %s", row3[9])
				}
				if !strings.Contains(row3[10], "parent") {
					t.Error("ReplyParent not formatted correctly")
				}
			}
		}
	}
}

// TestExportToCSVEmpty tests CSV export with empty post list
func TestExportToCSVEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty_export.csv")

	emptyPosts := []models.Post{}

	err := exporter.ExportToCSV(emptyPosts, outputPath)
	if err != nil {
		t.Fatalf("ExportToCSV with empty posts failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("CSV file was not created: %s", outputPath)
	}

	// Read and verify it has headers
	fileData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	// Remove BOM
	csvData := bytes.TrimPrefix(fileData, []byte{0xEF, 0xBB, 0xBF})

	reader := csv.NewReader(bytes.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	// Should have exactly 1 row (header only)
	if len(records) != 1 {
		t.Errorf("Expected 1 row (header only), got %d", len(records))
	}
}

// TestExportToCSVSpecialCharacters tests RFC 4180 compliance
func TestExportToCSVSpecialCharacters(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "special_chars.csv")

	// Create posts with various special characters that need escaping
	testPosts := []models.Post{
		{
			URI:       "at://test/post1",
			CID:       "cid1",
			DID:       "did:test",
			Text:      "Text with \"double quotes\"",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
		{
			URI:       "at://test/post2",
			CID:       "cid2",
			DID:       "did:test",
			Text:      "Text with, commas, everywhere",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
		{
			URI:       "at://test/post3",
			CID:       "cid3",
			DID:       "did:test",
			Text:      "Text with\nnewlines\nin it",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
		{
			URI:       "at://test/post4",
			CID:       "cid4",
			DID:       "did:test",
			Text:      "Text with \"quotes\", commas, and\nnewlines all together",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
	}

	err := exporter.ExportToCSV(testPosts, outputPath)
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	// Read and parse
	fileData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	csvData := bytes.TrimPrefix(fileData, []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV with special characters: %v", err)
	}

	// Verify all posts were exported (4 data rows + 1 header = 5 total)
	if len(records) != 5 {
		t.Errorf("Expected 5 rows, got %d", len(records))
	}

	// Verify each special character is preserved
	if len(records) > 1 && len(records[1]) >= 4 {
		if !strings.Contains(records[1][3], "\"double quotes\"") {
			t.Error("Double quotes not preserved correctly")
		}
	}

	if len(records) > 2 && len(records[2]) >= 4 {
		if !strings.Contains(records[2][3], "commas, everywhere") {
			t.Error("Commas not preserved correctly")
		}
	}

	if len(records) > 3 && len(records[3]) >= 4 {
		if !strings.Contains(records[3][3], "newlines") {
			t.Error("Newlines not preserved correctly")
		}
	}
}

// TestCSVUTF8BOM tests that UTF-8 BOM is correctly added
func TestCSVUTF8BOM(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "bom_test.csv")

	testPosts := []models.Post{
		{
			URI:       "at://test/post",
			CID:       "cid",
			DID:       "did:test",
			Text:      "Test",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
	}

	err := exporter.ExportToCSV(testPosts, outputPath)
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	// Read raw file bytes
	fileData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	// Check for UTF-8 BOM (0xEF 0xBB 0xBF)
	if len(fileData) < 3 {
		t.Fatal("File too short to contain BOM")
	}

	if fileData[0] != 0xEF || fileData[1] != 0xBB || fileData[2] != 0xBF {
		t.Errorf("UTF-8 BOM not found. Got bytes: %X %X %X", fileData[0], fileData[1], fileData[2])
	}
}

// TestCSVTimestampFormat tests ISO 8601 timestamp formatting
func TestCSVTimestampFormat(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "timestamp_test.csv")

	// Use specific time for testing
	testTime := time.Date(2025, 1, 31, 15, 30, 45, 0, time.UTC)

	testPosts := []models.Post{
		{
			URI:       "at://test/post",
			CID:       "cid",
			DID:       "did:test",
			Text:      "Test",
			CreatedAt: testTime,
			IndexedAt: testTime.Add(5 * time.Minute),
		},
	}

	err := exporter.ExportToCSV(testPosts, outputPath)
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	// Read and parse CSV
	fileData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	csvData := bytes.TrimPrefix(fileData, []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) < 2 {
		t.Fatal("CSV should have header and data row")
	}

	row := records[1]
	if len(row) < 15 {
		t.Fatalf("Row should have 15 columns, got %d", len(row))
	}

	// Verify CreatedAt timestamp (column 4)
	createdAt := row[4]
	if !strings.Contains(createdAt, "2025-01-31") || !strings.Contains(createdAt, "15:30:45") {
		t.Errorf("CreatedAt timestamp not in ISO 8601 format: %s", createdAt)
	}

	// Verify IndexedAt timestamp (column 14)
	indexedAt := row[14]
	if !strings.Contains(indexedAt, "2025-01-31") || !strings.Contains(indexedAt, "15:35:45") {
		t.Errorf("IndexedAt timestamp not in ISO 8601 format: %s", indexedAt)
	}
}

// TestExportToCSVInvalidPath tests error handling for invalid paths
func TestExportToCSVInvalidPath(t *testing.T) {
	invalidPath := "/nonexistent/directory/export.csv"

	testPosts := []models.Post{
		{
			URI:       "at://test/post",
			CID:       "cid",
			DID:       "did:test",
			Text:      "Test",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
	}

	err := exporter.ExportToCSV(testPosts, invalidPath)
	if err == nil {
		t.Fatal("Expected error for invalid path, got nil")
	}
}

// TestCSVMediaFilesExtraction tests that media hashes are correctly extracted from embed_data
func TestCSVMediaFilesExtraction(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "media_test.csv")

	// Create test posts with different embed types
	testPosts := []models.Post{
		{
			URI:       "at://test/post1",
			CID:       "cid1",
			DID:       "did:test",
			Text:      "Post with multiple images",
			CreatedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			IndexedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			HasMedia:  true,
			EmbedType: "images",
			EmbedData: []byte(`{
				"$type": "app.bsky.embed.images#view",
				"images": [
					{
						"alt": "First image",
						"fullsize": "https://cdn.bsky.app/img/feed_fullsize/plain/did:plc:test/bafkreiabc123@jpeg",
						"thumb": "https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:test/bafkreiabc123@jpeg"
					},
					{
						"alt": "Second image",
						"fullsize": "https://cdn.bsky.app/img/feed_fullsize/plain/did:plc:test/bafkreixyz789@jpeg",
						"thumb": "https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:test/bafkreixyz789@jpeg"
					}
				]
			}`),
		},
		{
			URI:       "at://test/post2",
			CID:       "cid2",
			DID:       "did:test",
			Text:      "Post with external link and thumbnail",
			CreatedAt: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
			IndexedAt: time.Date(2025, 1, 15, 11, 0, 0, 0, time.UTC),
			HasMedia:  true,
			EmbedType: "external",
			EmbedData: []byte(`{
				"$type": "app.bsky.embed.external#view",
				"external": {
					"description": "Test link",
					"thumb": "https://cdn.bsky.app/img/feed_thumbnail/plain/did:plc:test/bafkreiexternal@jpeg",
					"title": "Test Title",
					"uri": "https://example.com"
				}
			}`),
		},
		{
			URI:       "at://test/post3",
			CID:       "cid3",
			DID:       "did:test",
			Text:      "Post without media",
			CreatedAt: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
			IndexedAt: time.Date(2025, 1, 15, 12, 0, 0, 0, time.UTC),
			HasMedia:  false,
		},
	}

	// Export to CSV
	err := exporter.ExportToCSV(testPosts, outputPath)
	if err != nil {
		t.Fatalf("ExportToCSV failed: %v", err)
	}

	// Read and parse CSV
	fileData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read CSV file: %v", err)
	}

	csvData := bytes.TrimPrefix(fileData, []byte{0xEF, 0xBB, 0xBF})
	reader := csv.NewReader(bytes.NewReader(csvData))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to parse CSV: %v", err)
	}

	if len(records) != 4 { // header + 3 data rows
		t.Fatalf("Expected 4 rows, got %d", len(records))
	}

	// Find MediaFiles column index
	header := records[0]
	mediaFilesIndex := -1
	for i, col := range header {
		if col == "MediaFiles" {
			mediaFilesIndex = i
			break
		}
	}
	if mediaFilesIndex == -1 {
		t.Fatal("MediaFiles column not found in header")
	}

	// Verify first post has two media hashes separated by semicolon
	row1MediaFiles := records[1][mediaFilesIndex]
	expectedRow1 := "bafkreiabc123;bafkreixyz789"
	if row1MediaFiles != expectedRow1 {
		t.Errorf("Row 1 MediaFiles: expected '%s', got '%s'", expectedRow1, row1MediaFiles)
	}

	// Verify second post has one media hash (external thumb)
	row2MediaFiles := records[2][mediaFilesIndex]
	expectedRow2 := "bafkreiexternal"
	if row2MediaFiles != expectedRow2 {
		t.Errorf("Row 2 MediaFiles: expected '%s', got '%s'", expectedRow2, row2MediaFiles)
	}

	// Verify third post has no media files (empty string)
	row3MediaFiles := records[3][mediaFilesIndex]
	if row3MediaFiles != "" {
		t.Errorf("Row 3 MediaFiles: expected empty string, got '%s'", row3MediaFiles)
	}
}
