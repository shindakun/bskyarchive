package models

import (
	"fmt"
	"time"
)

// OperationType represents the type of archive operation
type OperationType string

const (
	OperationTypeInitial     OperationType = "initial"
	OperationTypeIncremental OperationType = "incremental"
	OperationTypeRefresh     OperationType = "refresh"
)

// OperationStatus represents the status of an archive operation
type OperationStatus string

const (
	OperationStatusPending   OperationStatus = "pending"
	OperationStatusRunning   OperationStatus = "running"
	OperationStatusCompleted OperationStatus = "completed"
	OperationStatusFailed    OperationStatus = "failed"
	OperationStatusCancelled OperationStatus = "cancelled"
)

// ArchiveOperation represents a background archive operation
type ArchiveOperation struct {
	ID              string          `json:"id" db:"id"`
	DID             string          `json:"did" db:"did"`
	Type            OperationType   `json:"type" db:"type"`
	Status          OperationStatus `json:"status" db:"status"`
	ProgressCurrent int64           `json:"progress_current" db:"progress"`
	ProgressTotal   int64           `json:"progress_total" db:"total"`
	ErrorMessage    string          `json:"error_message,omitempty" db:"error"`
	StartedAt       time.Time       `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time      `json:"completed_at,omitempty" db:"completed_at"`
}

// Validate checks if the operation fields are valid
func (o *ArchiveOperation) Validate() error {
	if o.ID == "" {
		return fmt.Errorf("id is required")
	}

	if o.DID == "" {
		return fmt.Errorf("did is required")
	}

	if o.Type != OperationTypeInitial && o.Type != OperationTypeIncremental && o.Type != OperationTypeRefresh {
		return fmt.Errorf("invalid operation type: %s", o.Type)
	}

	if o.Status != OperationStatusPending && o.Status != OperationStatusRunning &&
		o.Status != OperationStatusCompleted && o.Status != OperationStatusFailed &&
		o.Status != OperationStatusCancelled {
		return fmt.Errorf("invalid operation status: %s", o.Status)
	}

	if o.ProgressCurrent < 0 {
		return fmt.Errorf("progress_current must be non-negative")
	}

	if o.ProgressTotal < 0 {
		return fmt.Errorf("progress_total must be non-negative")
	}

	if o.ProgressTotal > 0 && o.ProgressCurrent > o.ProgressTotal {
		return fmt.Errorf("progress_current cannot exceed progress_total")
	}

	return nil
}

// ProgressPercentage calculates the completion percentage (0-100)
func (o *ArchiveOperation) ProgressPercentage() float64 {
	if o.ProgressTotal == 0 {
		return 0.0
	}
	return (float64(o.ProgressCurrent) / float64(o.ProgressTotal)) * 100.0
}

// IsComplete checks if the operation has finished (regardless of success/failure)
func (o *ArchiveOperation) IsComplete() bool {
	return o.Status == OperationStatusCompleted ||
		o.Status == OperationStatusFailed ||
		o.Status == OperationStatusCancelled
}

// IsActive checks if the operation is currently running
func (o *ArchiveOperation) IsActive() bool {
	return o.Status == OperationStatusRunning
}
