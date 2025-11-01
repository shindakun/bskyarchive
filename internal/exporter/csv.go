package exporter

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/shindakun/bskyarchive/internal/models"
)

// ExportToCSV exports posts to a CSV file with proper encoding and formatting
// The CSV includes a UTF-8 BOM for Excel compatibility and follows RFC 4180
func ExportToCSV(posts []models.Post, outputPath string) error {
	// Create the CSV file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	// Write UTF-8 BOM for Excel compatibility
	// Excel requires BOM to correctly detect UTF-8 encoding
	if _, err := file.WriteString("\xEF\xBB\xBF"); err != nil {
		return fmt.Errorf("failed to write BOM: %w", err)
	}

	// Create CSV writer
	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header row with 15 columns
	header := []string{
		"URI",
		"CID",
		"DID",
		"Text",
		"CreatedAt",
		"LikeCount",
		"RepostCount",
		"ReplyCount",
		"QuoteCount",
		"IsReply",
		"ReplyParent",
		"HasMedia",
		"MediaFiles",
		"EmbedType",
		"IndexedAt",
	}

	if err := writer.Write(header); err != nil {
		return fmt.Errorf("failed to write CSV header: %w", err)
	}

	// Write data rows
	for _, post := range posts {
		row, err := postToCSVRow(post)
		if err != nil {
			return fmt.Errorf("failed to convert post to CSV row: %w", err)
		}

		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	// Ensure all data is written
	if err := writer.Error(); err != nil {
		return fmt.Errorf("CSV writer error: %w", err)
	}

	return nil
}

// postToCSVRow converts a Post model to a CSV row
// Returns a slice of strings representing the row values
func postToCSVRow(post models.Post) ([]string, error) {
	// Format timestamps in ISO 8601 format
	createdAt := post.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	indexedAt := post.IndexedAt.Format("2006-01-02T15:04:05Z07:00")

	// Convert boolean to string
	isReply := "false"
	if post.IsReply {
		isReply = "true"
	}

	hasMedia := "false"
	if post.HasMedia {
		hasMedia = "true"
	}

	// Format reply parent (empty if not a reply)
	replyParent := post.ReplyParent

	// Get media files as semicolon-separated list
	mediaFiles := getMediaFilesList(post)

	// Build row with all columns
	row := []string{
		post.URI,
		post.CID,
		post.DID,
		post.Text,
		createdAt,
		fmt.Sprintf("%d", post.LikeCount),
		fmt.Sprintf("%d", post.RepostCount),
		fmt.Sprintf("%d", post.ReplyCount),
		fmt.Sprintf("%d", post.QuoteCount),
		isReply,
		replyParent,
		hasMedia,
		mediaFiles,
		post.EmbedType,
		indexedAt,
	}

	return row, nil
}

// getMediaFilesList extracts media file hashes from post and returns semicolon-separated list
func getMediaFilesList(post models.Post) string {
	if !post.HasMedia || len(post.EmbedData) == 0 {
		return ""
	}

	// For now, return empty string - media file extraction would require
	// parsing the embed_data JSON which contains the media hashes
	// This can be enhanced in a future iteration if needed
	return ""
}
