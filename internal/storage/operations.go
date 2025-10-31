package storage

import (
	"database/sql"
	"fmt"

	"github.com/shindakun/bskyarchive/internal/models"
)

// CreateOperation creates a new archive operation
func CreateOperation(db *sql.DB, op *models.ArchiveOperation) error {
	if err := op.Validate(); err != nil {
		return fmt.Errorf("invalid operation: %w", err)
	}

	query := `
		INSERT INTO operations (
			id, did, type, status, progress, total, error, started_at, completed_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := db.Exec(query,
		op.ID, op.DID, op.Type, op.Status, op.ProgressCurrent, op.ProgressTotal,
		op.ErrorMessage, op.StartedAt, op.CompletedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create operation: %w", err)
	}

	return nil
}

// UpdateOperation updates an existing archive operation
func UpdateOperation(db *sql.DB, op *models.ArchiveOperation) error {
	if err := op.Validate(); err != nil {
		return fmt.Errorf("invalid operation: %w", err)
	}

	query := `
		UPDATE operations
		SET status = ?, progress = ?, total = ?, error = ?, completed_at = ?
		WHERE id = ?
	`

	result, err := db.Exec(query,
		op.Status, op.ProgressCurrent, op.ProgressTotal, op.ErrorMessage, op.CompletedAt, op.ID,
	)

	if err != nil {
		return fmt.Errorf("failed to update operation: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("operation not found: %s", op.ID)
	}

	return nil
}

// GetActiveOperation retrieves the currently active operation for a user
func GetActiveOperation(db *sql.DB, did string) (*models.ArchiveOperation, error) {
	query := `
		SELECT id, did, type, status, progress, total, error, started_at, completed_at
		FROM operations
		WHERE did = ? AND status IN ('pending', 'running')
		ORDER BY started_at DESC
		LIMIT 1
	`

	var op models.ArchiveOperation
	var completedAt sql.NullTime

	err := db.QueryRow(query, did).Scan(
		&op.ID, &op.DID, &op.Type, &op.Status, &op.ProgressCurrent, &op.ProgressTotal,
		&op.ErrorMessage, &op.StartedAt, &completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No active operation
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active operation: %w", err)
	}

	if completedAt.Valid {
		op.CompletedAt = &completedAt.Time
	}

	return &op, nil
}

// GetOperation retrieves an operation by ID
func GetOperation(db *sql.DB, id string) (*models.ArchiveOperation, error) {
	query := `
		SELECT id, did, type, status, progress, total, error, started_at, completed_at
		FROM operations
		WHERE id = ?
	`

	var op models.ArchiveOperation
	var completedAt sql.NullTime

	err := db.QueryRow(query, id).Scan(
		&op.ID, &op.DID, &op.Type, &op.Status, &op.ProgressCurrent, &op.ProgressTotal,
		&op.ErrorMessage, &op.StartedAt, &completedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("operation not found: %s", id)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get operation: %w", err)
	}

	if completedAt.Valid {
		op.CompletedAt = &completedAt.Time
	}

	return &op, nil
}

// ListRecentOperations retrieves the most recent operations for a user
func ListRecentOperations(db *sql.DB, did string, limit int) ([]models.ArchiveOperation, error) {
	if limit <= 0 {
		limit = 5 // Default to 5 recent operations
	}

	query := `
		SELECT id, did, type, status, progress, total, error, started_at, completed_at
		FROM operations
		WHERE did = ?
		ORDER BY started_at DESC
		LIMIT ?
	`

	rows, err := db.Query(query, did, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list operations: %w", err)
	}
	defer rows.Close()

	var operations []models.ArchiveOperation
	for rows.Next() {
		var op models.ArchiveOperation
		var completedAt sql.NullTime

		err := rows.Scan(
			&op.ID, &op.DID, &op.Type, &op.Status, &op.ProgressCurrent, &op.ProgressTotal,
			&op.ErrorMessage, &op.StartedAt, &completedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan operation: %w", err)
		}

		if completedAt.Valid {
			op.CompletedAt = &completedAt.Time
		}

		operations = append(operations, op)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating operations: %w", err)
	}

	return operations, nil
}
