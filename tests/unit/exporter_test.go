package unit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/exporter"
	"github.com/shindakun/bskyarchive/internal/models"
)

// TestExportToJSON tests JSON export functionality
func TestExportToJSON(t *testing.T) {
	// Create temporary directory for test output
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test_export.json")

	// Create test posts
	testPosts := []models.Post{
		{
			URI:         "at://did:plc:test123/app.bsky.feed.post/post1",
			CID:         "bafytest123",
			DID:         "did:plc:test123",
			Text:        "First test post",
			CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
			IndexedAt:   time.Now().UTC(),
			LikeCount:   10,
			RepostCount: 5,
			ReplyCount:  2,
			QuoteCount:  1,
			HasMedia:    false,
		},
		{
			URI:         "at://did:plc:test123/app.bsky.feed.post/post2",
			CID:         "bafytest456",
			DID:         "did:plc:test123",
			Text:        "Second test post with emoji 🚀",
			CreatedAt:   time.Now().UTC().Add(-1 * time.Hour),
			IndexedAt:   time.Now().UTC(),
			LikeCount:   25,
			RepostCount: 10,
			ReplyCount:  5,
			QuoteCount:  3,
			HasMedia:    true,
			EmbedType:   "images",
		},
		{
			URI:         "at://did:plc:test123/app.bsky.feed.post/post3",
			CID:         "bafytest789",
			DID:         "did:plc:test123",
			Text:        "Third test post",
			CreatedAt:   time.Now().UTC(),
			IndexedAt:   time.Now().UTC(),
			LikeCount:   0,
			RepostCount: 0,
			ReplyCount:  0,
			QuoteCount:  0,
			HasMedia:    false,
		},
	}

	// Test ExportToJSON
	err := exporter.ExportToJSON(testPosts, outputPath)
	if err != nil {
		t.Fatalf("ExportToJSON failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Output file was not created: %s", outputPath)
	}

	// Read and parse the JSON file
	fileData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	// Unmarshal JSON
	var exportedPosts []models.Post
	if err := json.Unmarshal(fileData, &exportedPosts); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	// Verify post count
	if len(exportedPosts) != len(testPosts) {
		t.Errorf("Expected %d posts, got %d", len(testPosts), len(exportedPosts))
	}

	// Verify first post fields
	if len(exportedPosts) > 0 {
		first := exportedPosts[0]
		if first.URI != testPosts[0].URI {
			t.Errorf("URI mismatch: got %s, want %s", first.URI, testPosts[0].URI)
		}
		if first.CID != testPosts[0].CID {
			t.Errorf("CID mismatch: got %s, want %s", first.CID, testPosts[0].CID)
		}
		if first.Text != testPosts[0].Text {
			t.Errorf("Text mismatch: got %s, want %s", first.Text, testPosts[0].Text)
		}
		if first.LikeCount != testPosts[0].LikeCount {
			t.Errorf("LikeCount mismatch: got %d, want %d", first.LikeCount, testPosts[0].LikeCount)
		}
	}

	// Verify emoji preservation in second post
	if len(exportedPosts) > 1 {
		second := exportedPosts[1]
		if second.Text != testPosts[1].Text {
			t.Errorf("Emoji not preserved in text: got %s, want %s", second.Text, testPosts[1].Text)
		}
		if second.EmbedType != testPosts[1].EmbedType {
			t.Errorf("EmbedType mismatch: got %s, want %s", second.EmbedType, testPosts[1].EmbedType)
		}
		if second.HasMedia != testPosts[1].HasMedia {
			t.Errorf("HasMedia mismatch: got %v, want %v", second.HasMedia, testPosts[1].HasMedia)
		}
	}
}

// TestExportToJSONEmpty tests JSON export with empty post list
func TestExportToJSONEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "empty_export.json")

	// Test with empty post slice
	emptyPosts := []models.Post{}

	err := exporter.ExportToJSON(emptyPosts, outputPath)
	if err != nil {
		t.Fatalf("ExportToJSON with empty posts failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		t.Fatalf("Output file was not created: %s", outputPath)
	}

	// Read and verify it's valid JSON
	fileData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	var exportedPosts []models.Post
	if err := json.Unmarshal(fileData, &exportedPosts); err != nil {
		t.Fatalf("Failed to unmarshal empty JSON: %v", err)
	}

	if len(exportedPosts) != 0 {
		t.Errorf("Expected 0 posts, got %d", len(exportedPosts))
	}
}

// TestExportToJSONInvalidPath tests JSON export with invalid file path
func TestExportToJSONInvalidPath(t *testing.T) {
	// Use invalid path (directory that doesn't exist)
	invalidPath := "/nonexistent/directory/export.json"

	testPosts := []models.Post{
		{
			URI:       "at://did:plc:test123/app.bsky.feed.post/post1",
			CID:       "bafytest123",
			DID:       "did:plc:test123",
			Text:      "Test post",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
	}

	// Should return an error
	err := exporter.ExportToJSON(testPosts, invalidPath)
	if err == nil {
		t.Fatal("Expected error for invalid path, got nil")
	}
}

// TestExportToJSONPrettyFormatting tests that JSON output is properly formatted
func TestExportToJSONPrettyFormatting(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "formatted_export.json")

	testPosts := []models.Post{
		{
			URI:       "at://did:plc:test123/app.bsky.feed.post/post1",
			CID:       "bafytest123",
			DID:       "did:plc:test123",
			Text:      "Test post",
			CreatedAt: time.Now().UTC(),
			IndexedAt: time.Now().UTC(),
		},
	}

	err := exporter.ExportToJSON(testPosts, outputPath)
	if err != nil {
		t.Fatalf("ExportToJSON failed: %v", err)
	}

	// Read file content as string
	fileData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}

	content := string(fileData)

	// Check for indentation (pretty-printing)
	// The JSON encoder uses 2-space indentation
	if len(content) < 10 {
		t.Fatal("Output file is too short")
	}

	// Verify it contains newlines and indentation (pretty-printed)
	// Count newlines - should have multiple for pretty-printed JSON
	newlineCount := 0
	for _, char := range content {
		if char == '\n' {
			newlineCount++
		}
	}

	if newlineCount < 5 {
		t.Errorf("Expected pretty-printed JSON with multiple lines, got %d newlines", newlineCount)
	}
}
