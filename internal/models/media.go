package models

import (
	"fmt"
	"time"
)

// Media represents media (images, videos) embedded in posts, with local storage information
type Media struct {
	Hash      string    `json:"hash" db:"hash"`             // SHA-256 hash (content-addressable)
	PostURI   string    `json:"post_uri" db:"post_uri"`     // FK to posts table
	MimeType  string    `json:"mime_type" db:"mime_type"`   // e.g., "image/jpeg"
	FilePath  string    `json:"file_path" db:"file_path"`   // Local file path
	SizeBytes int64     `json:"size_bytes" db:"size_bytes"` // File size in bytes
	Width     int       `json:"width" db:"width"`           // Image/video width
	Height    int       `json:"height" db:"height"`         // Image/video height
	AltText   string    `json:"alt_text" db:"alt_text"`     // Accessibility alt text
	CreatedAt time.Time `json:"created_at" db:"created_at"` // When archived
}

// Validate checks if the media fields are valid
func (m *Media) Validate() error {
	if m.Hash == "" {
		return fmt.Errorf("hash is required")
	}

	if len(m.Hash) != 64 {
		return fmt.Errorf("hash must be 64 characters (SHA-256)")
	}

	if m.PostURI == "" {
		return fmt.Errorf("post_uri is required")
	}

	if m.MimeType == "" {
		return fmt.Errorf("mime_type is required")
	}

	if m.FilePath == "" {
		return fmt.Errorf("file_path is required")
	}

	if m.SizeBytes < 0 {
		return fmt.Errorf("size_bytes must be non-negative")
	}

	if m.Width < 0 {
		return fmt.Errorf("width must be non-negative")
	}

	if m.Height < 0 {
		return fmt.Errorf("height must be non-negative")
	}

	return nil
}
