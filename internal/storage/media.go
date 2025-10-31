package storage

import (
	"database/sql"
	"fmt"

	"github.com/shindakun/bskyarchive/internal/models"
)

// SaveMedia saves media metadata to the database with content-addressable path
func SaveMedia(db *sql.DB, media *models.Media) error {
	if err := media.Validate(); err != nil {
		return fmt.Errorf("invalid media: %w", err)
	}

	query := `
		INSERT INTO media (
			hash, post_uri, mime_type, file_path, size_bytes,
			width, height, alt_text, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(hash) DO UPDATE SET
			post_uri = excluded.post_uri,
			mime_type = excluded.mime_type,
			file_path = excluded.file_path,
			size_bytes = excluded.size_bytes,
			width = excluded.width,
			height = excluded.height,
			alt_text = excluded.alt_text
	`

	_, err := db.Exec(query,
		media.Hash, media.PostURI, media.MimeType, media.FilePath, media.SizeBytes,
		media.Width, media.Height, media.AltText, media.CreatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save media: %w", err)
	}

	return nil
}

// ListMediaForPost retrieves all media associated with a post
func ListMediaForPost(db *sql.DB, postURI string) ([]models.Media, error) {
	query := `
		SELECT hash, post_uri, mime_type, file_path, size_bytes,
			   width, height, alt_text, created_at
		FROM media
		WHERE post_uri = ?
		ORDER BY created_at ASC
	`

	rows, err := db.Query(query, postURI)
	if err != nil {
		return nil, fmt.Errorf("failed to list media: %w", err)
	}
	defer rows.Close()

	var mediaList []models.Media
	for rows.Next() {
		var media models.Media
		err := rows.Scan(
			&media.Hash, &media.PostURI, &media.MimeType, &media.FilePath,
			&media.SizeBytes, &media.Width, &media.Height, &media.AltText, &media.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan media: %w", err)
		}
		mediaList = append(mediaList, media)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating media: %w", err)
	}

	return mediaList, nil
}

// GetMediaByHash retrieves media by its content hash
func GetMediaByHash(db *sql.DB, hash string) (*models.Media, error) {
	query := `
		SELECT hash, post_uri, mime_type, file_path, size_bytes,
			   width, height, alt_text, created_at
		FROM media
		WHERE hash = ?
	`

	var media models.Media
	err := db.QueryRow(query, hash).Scan(
		&media.Hash, &media.PostURI, &media.MimeType, &media.FilePath,
		&media.SizeBytes, &media.Width, &media.Height, &media.AltText, &media.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("media not found: %s", hash)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get media: %w", err)
	}

	return &media, nil
}
