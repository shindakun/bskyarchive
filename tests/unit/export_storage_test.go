package unit

import (
	"testing"
	"time"

	"github.com/shindakun/bskyarchive/internal/models"
)

// TestExportSizeFormatting verifies the HumanSize method correctly formats byte sizes
func TestExportSizeFormatting(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "bytes",
			bytes:    512,
			expected: "512 B",
		},
		{
			name:     "kilobytes",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "kilobytes with decimal",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "megabytes",
			bytes:    1048576, // 1 MB
			expected: "1.0 MB",
		},
		{
			name:     "megabytes with decimal",
			bytes:    2621440, // 2.5 MB
			expected: "2.5 MB",
		},
		{
			name:     "gigabytes",
			bytes:    1073741824, // 1 GB
			expected: "1.0 GB",
		},
		{
			name:     "gigabytes with decimal",
			bytes:    5368709120, // 5 GB
			expected: "5.0 GB",
		},
		{
			name:     "large archive",
			bytes:    10737418240, // 10 GB
			expected: "10.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := &models.ExportRecord{
				SizeBytes: tt.bytes,
			}

			result := record.HumanSize()
			if result != tt.expected {
				t.Errorf("HumanSize() for %d bytes = %s; want %s", tt.bytes, result, tt.expected)
			}
		})
	}

	t.Log("✓ Export size formatting test passed")
}

// TestDateRangeFormatting verifies the DateRangeString method
func TestDateRangeFormatting(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *models.ExportRecord
		expected string
	}{
		{
			name: "no date range",
			setup: func() *models.ExportRecord {
				return &models.ExportRecord{}
			},
			expected: "All posts",
		},
		{
			name: "with start date only",
			setup: func() *models.ExportRecord {
				// This case should not happen in practice, but test it anyway
				start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				return &models.ExportRecord{
					DateRangeStart: &start,
				}
			},
			expected: "From 2025-01-01",
		},
		{
			name: "with end date only",
			setup: func() *models.ExportRecord {
				// This case should not happen in practice, but test it anyway
				end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
				return &models.ExportRecord{
					DateRangeEnd: &end,
				}
			},
			expected: "Until 2025-12-31",
		},
		{
			name: "with both start and end dates",
			setup: func() *models.ExportRecord {
				start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
				end := time.Date(2025, 12, 31, 0, 0, 0, 0, time.UTC)
				return &models.ExportRecord{
					DateRangeStart: &start,
					DateRangeEnd:   &end,
				}
			},
			expected: "2025-01-01 to 2025-12-31",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := tt.setup()
			result := record.DateRangeString()
			if result != tt.expected {
				t.Errorf("DateRangeString() = %s; want %s", result, tt.expected)
			}
		})
	}

	t.Log("✓ Date range formatting test passed")
}

// TestExportRecordValidation verifies export record validation
func TestExportRecordValidation(t *testing.T) {
	tests := []struct {
		name      string
		record    *models.ExportRecord
		wantError bool
		errorMsg  string
	}{
		{
			name: "valid record",
			record: &models.ExportRecord{
				ID:            "did:plc:test/2025-11-01_12-00-00",
				DID:           "did:plc:test",
				Format:        "json",
				DirectoryPath: "./exports/did:plc:test/2025-11-01_12-00-00",
				PostCount:     100,
				MediaCount:    50,
				SizeBytes:     1048576,
			},
			wantError: false,
		},
		{
			name: "missing ID",
			record: &models.ExportRecord{
				DID:           "did:plc:test",
				Format:        "json",
				DirectoryPath: "./exports/test",
			},
			wantError: true,
			errorMsg:  "ID is required",
		},
		{
			name: "missing DID",
			record: &models.ExportRecord{
				ID:            "test/timestamp",
				Format:        "json",
				DirectoryPath: "./exports/test",
			},
			wantError: true,
			errorMsg:  "DID is required",
		},
		{
			name: "invalid format",
			record: &models.ExportRecord{
				ID:            "test/timestamp",
				DID:           "did:plc:test",
				Format:        "xml", // invalid
				DirectoryPath: "./exports/test",
			},
			wantError: true,
			errorMsg:  "format must be 'json' or 'csv'",
		},
		{
			name: "negative post count",
			record: &models.ExportRecord{
				ID:            "test/timestamp",
				DID:           "did:plc:test",
				Format:        "json",
				DirectoryPath: "./exports/test",
				PostCount:     -1,
			},
			wantError: true,
			errorMsg:  "post count must be >= 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errorMsg)
				} else if err.Error() != tt.errorMsg && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errorMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}

	t.Log("✓ Export record validation test passed")
}

// Helper function for error message checking
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		findSubstringInString(s, substr)))
}

func findSubstringInString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
