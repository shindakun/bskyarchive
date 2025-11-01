package exporter

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/shindakun/bskyarchive/internal/models"
	"github.com/shindakun/bskyarchive/internal/storage"
)

// ExportToJSON exports posts to a JSON file with pretty-printing
// Posts are exported as an array of Post objects
//
// DEPRECATED: This function loads all posts into memory.
// For large archives, use ExportToJSONBatched instead.
func ExportToJSON(posts []models.Post, outputPath string) error {
	// Create the output file
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer file.Close()

	// Create JSON encoder with pretty-printing (2-space indent)
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false) // Don't escape HTML entities (< > &)

	// Write the posts as a JSON array
	if err := encoder.Encode(posts); err != nil {
		return fmt.Errorf("failed to encode posts to JSON: %w", err)
	}

	return nil
}

// ExportToJSONBatched exports posts to JSON using batched streaming writes
// This prevents memory exhaustion on large archives by processing posts in batches
func ExportToJSONBatched(db *sql.DB, did string, dateRange *models.DateRange, outputPath string, batchSize int) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create JSON file: %w", err)
	}
	defer file.Close()

	// Write opening bracket
	if _, err := file.WriteString("[\n"); err != nil {
		return fmt.Errorf("failed to write opening bracket: %w", err)
	}

	offset := 0
	isFirst := true

	for {
		// Fetch next batch
		batch, err := storage.ListPostsWithDateRange(db, did, dateRange, batchSize, offset)
		if err != nil {
			return fmt.Errorf("failed to fetch batch at offset %d: %w", offset, err)
		}

		// No more posts
		if len(batch) == 0 {
			break
		}

		// Write each post in the batch
		for _, post := range batch {
			if err := exportToJSONStreamingWriter(file, post, isFirst, false); err != nil {
				return err
			}
			isFirst = false
		}

		offset += len(batch)

		// If we got fewer posts than batch size, we're done
		if len(batch) < batchSize {
			break
		}
	}

	// Write closing bracket
	if _, err := file.WriteString("\n]\n"); err != nil {
		return fmt.Errorf("failed to write closing bracket: %w", err)
	}

	return nil
}

// exportToJSONStreamingWriter writes a single post to the JSON array
// isFirst: true if this is the first post (no comma prefix)
// isLast: true if this is the last post (no comma suffix) - currently unused but kept for symmetry
func exportToJSONStreamingWriter(w io.Writer, post models.Post, isFirst, isLast bool) error {
	// Add comma separator before all posts except the first
	if !isFirst {
		if _, err := w.Write([]byte(",\n")); err != nil {
			return fmt.Errorf("failed to write comma separator: %w", err)
		}
	}

	// Encode the post with indentation
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	// Encode to temporary buffer to handle indentation properly
	var buf strings.Builder
	tempEncoder := json.NewEncoder(&buf)
	tempEncoder.SetIndent("", "  ")
	tempEncoder.SetEscapeHTML(false)

	if err := tempEncoder.Encode(post); err != nil {
		return fmt.Errorf("failed to encode post: %w", err)
	}

	// Write the encoded post with proper indentation
	// Remove trailing newline from encoder output
	encoded := strings.TrimRight(buf.String(), "\n")

	// Add 2-space indentation to each line
	lines := strings.Split(encoded, "\n")
	for i, line := range lines {
		if i > 0 {
			if _, err := w.Write([]byte("\n")); err != nil {
				return err
			}
		}
		if _, err := w.Write([]byte("  " + line)); err != nil {
			return fmt.Errorf("failed to write indented line: %w", err)
		}
	}

	return nil
}
