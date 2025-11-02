package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
)

// CreateExportRecord inserts a new export record into the database
func CreateExportRecord(db *sql.DB, record *models.ExportRecord) error {
	if err := record.Validate(); err != nil {
		return fmt.Errorf("invalid export record: %w", err)
	}

	var startUnix, endUnix *int64
	if record.DateRangeStart != nil {
		unix := record.DateRangeStart.Unix()
		startUnix = &unix
	}
	if record.DateRangeEnd != nil {
		unix := record.DateRangeEnd.Unix()
		endUnix = &unix
	}

	_, err := db.Exec(`
		INSERT INTO exports (id, did, format, created_at, directory_path,
		                     post_count, media_count, size_bytes,
		                     date_range_start, date_range_end, manifest_path)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, record.ID, record.DID, record.Format, record.CreatedAt.Unix(),
		record.DirectoryPath, record.PostCount, record.MediaCount,
		record.SizeBytes, startUnix, endUnix, record.ManifestPath)

	if err != nil {
		return fmt.Errorf("failed to insert export record: %w", err)
	}

	return nil
}

// GetExportByID retrieves a single export record by ID
func GetExportByID(db *sql.DB, id string) (*models.ExportRecord, error) {
	var e models.ExportRecord
	var createdAtUnix int64
	var startUnix, endUnix *int64

	err := db.QueryRow(`
		SELECT id, did, format, created_at, directory_path,
		       post_count, media_count, size_bytes,
		       date_range_start, date_range_end, manifest_path
		FROM exports WHERE id = ?
	`, id).Scan(&e.ID, &e.DID, &e.Format, &createdAtUnix,
		&e.DirectoryPath, &e.PostCount, &e.MediaCount,
		&e.SizeBytes, &startUnix, &endUnix, &e.ManifestPath)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("export not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query export: %w", err)
	}

	// Convert Unix timestamps to time.Time
	e.CreatedAt = time.Unix(createdAtUnix, 0)
	if startUnix != nil {
		t := time.Unix(*startUnix, 0)
		e.DateRangeStart = &t
	}
	if endUnix != nil {
		t := time.Unix(*endUnix, 0)
		e.DateRangeEnd = &t
	}

	return &e, nil
}

// ListExportsByDID returns all exports for a specific user (DID)
// Results are ordered by created_at DESC (newest first)
func ListExportsByDID(db *sql.DB, did string, limit, offset int) ([]models.ExportRecord, error) {
	rows, err := db.Query(`
		SELECT id, did, format, created_at, directory_path,
		       post_count, media_count, size_bytes,
		       date_range_start, date_range_end, manifest_path
		FROM exports
		WHERE did = ?
		ORDER BY created_at DESC
		LIMIT ? OFFSET ?
	`, did, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query exports: %w", err)
	}
	defer rows.Close()

	var exports []models.ExportRecord
	for rows.Next() {
		var e models.ExportRecord
		var createdAtUnix int64
		var startUnix, endUnix *int64

		err := rows.Scan(&e.ID, &e.DID, &e.Format, &createdAtUnix,
			&e.DirectoryPath, &e.PostCount, &e.MediaCount,
			&e.SizeBytes, &startUnix, &endUnix, &e.ManifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan export record: %w", err)
		}

		// Convert Unix timestamps to time.Time
		e.CreatedAt = time.Unix(createdAtUnix, 0)
		if startUnix != nil {
			t := time.Unix(*startUnix, 0)
			e.DateRangeStart = &t
		}
		if endUnix != nil {
			t := time.Unix(*endUnix, 0)
			e.DateRangeEnd = &t
		}

		exports = append(exports, e)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating export records: %w", err)
	}

	return exports, nil
}

// DeleteExport removes an export record from the database
// Note: This only removes the database record, not the files on disk
// The caller is responsible for deleting the export directory
func DeleteExport(db *sql.DB, id string) error {
	result, err := db.Exec("DELETE FROM exports WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete export: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rows == 0 {
		return fmt.Errorf("export not found: %s", id)
	}

	return nil
}
