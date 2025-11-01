package exporter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/shindakun/bskyarchive/internal/models"
)

// ExportToJSON exports posts to a JSON file with pretty-printing
// Posts are exported as an array of Post objects
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
