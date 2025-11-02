# Quickstart: Export Download & Management

**Feature**: 005-export-download
**Date**: 2025-11-01

This guide helps developers implement the export download and management feature from scratch.

---

## Prerequisites

- Go 1.21+ installed
- Existing bskyarchive project cloned
- SQLite database configured
- Familiar with Go's `archive/zip`, `net/http`, `database/sql`

---

## Step 1: Database Migration

Create the `exports` table to track completed exports.

### Create Migration File

**File**: `internal/storage/migrations/005_exports.go`

```go
package migrations

import "database/sql"

func Migrate005ExportsTable(db *sql.DB) error {
    _, err := db.Exec(`
        CREATE TABLE IF NOT EXISTS exports (
            id TEXT PRIMARY KEY,
            did TEXT NOT NULL,
            format TEXT NOT NULL,
            created_at INTEGER NOT NULL,
            directory_path TEXT NOT NULL,
            post_count INTEGER NOT NULL,
            media_count INTEGER DEFAULT 0,
            size_bytes INTEGER DEFAULT 0,
            date_range_start INTEGER,
            date_range_end INTEGER,
            manifest_path TEXT,
            CHECK (post_count >= 0),
            CHECK (media_count >= 0),
            CHECK (size_bytes >= 0)
        );

        CREATE INDEX IF NOT EXISTS idx_exports_did ON exports(did);
        CREATE INDEX IF NOT EXISTS idx_exports_created_at ON exports(created_at DESC);
        CREATE INDEX IF NOT EXISTS idx_exports_did_created ON exports(did, created_at DESC);
    `)
    return err
}
```

### Run Migration

Add to migration runner:

```go
// internal/storage/migrate.go
func RunMigrations(db *sql.DB) error {
    // ... existing migrations ...
    if err := migrations.Migrate005ExportsTable(db); err != nil {
        return fmt.Errorf("migration 005 failed: %w", err)
    }
    return nil
}
```

---

## Step 2: Update Models

Add new types to `internal/models/export.go`.

### Add ExportRecord Type

```go
// ExportRecord represents a completed export in the database
type ExportRecord struct {
    ID             string     `json:"id"`
    DID            string     `json:"did"`
    Format         string     `json:"format"`
    CreatedAt      time.Time  `json:"created_at"`
    DirectoryPath  string     `json:"directory_path"`
    PostCount      int        `json:"post_count"`
    MediaCount     int        `json:"media_count"`
    SizeBytes      int64      `json:"size_bytes"`
    DateRangeStart *time.Time `json:"date_range_start,omitempty"`
    DateRangeEnd   *time.Time `json:"date_range_end,omitempty"`
    ManifestPath   string     `json:"manifest_path,omitempty"`
}

// Validate checks if the export record is valid
func (e *ExportRecord) Validate() error {
    if e.ID == "" || e.DID == "" {
        return fmt.Errorf("ID and DID are required")
    }
    if e.Format != "json" && e.Format != "csv" {
        return fmt.Errorf("format must be 'json' or 'csv'")
    }
    if !strings.HasPrefix(e.DirectoryPath, "./exports/") {
        return fmt.Errorf("invalid directory path")
    }
    return nil
}

// HumanSize returns size in human-readable format
func (e *ExportRecord) HumanSize() string {
    const unit = 1024
    if e.SizeBytes < unit {
        return fmt.Sprintf("%d B", e.SizeBytes)
    }
    div, exp := int64(unit), 0
    for n := e.SizeBytes / unit; n >= unit; n /= unit {
        div *= unit
        exp++
    }
    return fmt.Sprintf("%.1f %cB", float64(e.SizeBytes)/float64(div), "KMGTPE"[exp])
}
```

---

## Step 3: Create Storage Layer

Implement CRUD operations for exports.

### Create File

**File**: `internal/storage/exports.go`

```go
package storage

import (
    "database/sql"
    "fmt"
    "time"

    "github.com/shindakun/bskyarchive/internal/models"
)

// CreateExportRecord inserts a new export record
func CreateExportRecord(db *sql.DB, record *models.ExportRecord) error {
    if err := record.Validate(); err != nil {
        return err
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

    return err
}

// ListExportsByDID returns all exports for a user
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
        return nil, err
    }
    defer rows.Close()

    var exports []models.ExportRecord
    for rows.Next() {
        var e models.ExportRecord
        var createdAtUnix, startUnix, endUnix *int64
        err := rows.Scan(&e.ID, &e.DID, &e.Format, &createdAtUnix,
            &e.DirectoryPath, &e.PostCount, &e.MediaCount,
            &e.SizeBytes, &startUnix, &endUnix, &e.ManifestPath)
        if err != nil {
            return nil, err
        }

        if createdAtUnix != nil {
            e.CreatedAt = time.Unix(*createdAtUnix, 0)
        }
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

    return exports, rows.Err()
}

// GetExportByID retrieves a single export by ID
func GetExportByID(db *sql.DB, id string) (*models.ExportRecord, error) {
    var e models.ExportRecord
    var createdAtUnix, startUnix, endUnix *int64

    err := db.QueryRow(`
        SELECT id, did, format, created_at, directory_path,
               post_count, media_count, size_bytes,
               date_range_start, date_range_end, manifest_path
        FROM exports WHERE id = ?
    `, id).Scan(&e.ID, &e.DID, &e.Format, &createdAtUnix,
        &e.DirectoryPath, &e.PostCount, &e.MediaCount,
        &e.SizeBytes, &startUnix, &endUnix, &e.ManifestPath)

    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("export not found")
    }
    if err != nil {
        return nil, err
    }

    if createdAtUnix != nil {
        e.CreatedAt = time.Unix(*createdAtUnix, 0)
    }
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

// DeleteExport removes an export record
func DeleteExport(db *sql.DB, id string) error {
    result, err := db.Exec("DELETE FROM exports WHERE id = ?", id)
    if err != nil {
        return err
    }

    rows, err := result.RowsAffected()
    if err != nil {
        return err
    }
    if rows == 0 {
        return fmt.Errorf("export not found")
    }

    return nil
}
```

---

## Step 4: Implement ZIP Streaming

Create streaming ZIP functionality.

### Create File

**File**: `internal/exporter/download.go`

```go
package exporter

import (
    "archive/zip"
    "fmt"
    "io"
    "os"
    "path/filepath"
)

// StreamDirectoryAsZIP streams a directory as a ZIP archive
func StreamDirectoryAsZIP(w io.Writer, rootPath string) error {
    zipWriter := zip.NewWriter(w)
    defer zipWriter.Close()

    return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            return nil
        }

        // Calculate relative path for ZIP entry
        relPath, err := filepath.Rel(rootPath, path)
        if err != nil {
            return err
        }

        // Create ZIP entry header
        header, err := zip.FileInfoHeader(info)
        if err != nil {
            return err
        }
        header.Name = relPath
        header.Method = zip.Deflate

        // Write header
        writer, err := zipWriter.CreateHeader(header)
        if err != nil {
            return err
        }

        // Open and stream file
        file, err := os.Open(path)
        if err != nil {
            return err
        }
        defer file.Close()

        _, err = io.Copy(writer, file)
        return err
    })
}
```

---

## Step 5: Update Export Creation

Modify the exporter to track exports in the database.

### Update File

**File**: `internal/exporter/exporter.go`

Add this at the end of the `Run()` function (after export completes):

```go
// Calculate total export size
var totalSize int64
filepath.Walk(exportDir, func(_ string, info os.FileInfo, err error) error {
    if err == nil && !info.IsDir() {
        totalSize += info.Size()
    }
    return nil
})

// Create export record
exportRecord := &models.ExportRecord{
    ID:             fmt.Sprintf("%s/%s", job.Options.DID, filepath.Base(exportDir)),
    DID:            job.Options.DID,
    Format:         string(job.Options.Format),
    CreatedAt:      job.CreatedAt,
    DirectoryPath:  exportDir,
    PostCount:      totalPosts,
    MediaCount:     job.Progress.MediaCopied,
    SizeBytes:      totalSize,
    ManifestPath:   manifestPath,
}

if job.Options.DateRange != nil {
    exportRecord.DateRangeStart = &job.Options.DateRange.StartDate
    exportRecord.DateRangeEnd = &job.Options.DateRange.EndDate
}

// Save to database
if err := storage.CreateExportRecord(db, exportRecord); err != nil {
    log.Printf("Warning: Failed to save export record: %v", err)
    // Don't fail the export - this is metadata only
}
```

---

## Step 6: Add HTTP Handlers

Implement download and delete endpoints.

### Update File

**File**: `internal/web/handlers/export.go`

```go
// Rate limiting for downloads
var (
    activeDownloads = make(map[string]int)
    downloadsMu     sync.RWMutex
)

// ListExports shows all user's exports
func (h *Handlers) ListExports(w http.ResponseWriter, r *http.Request) {
    session := auth.GetSessionFromContext(r.Context())
    if session == nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    exports, err := storage.ListExportsByDID(h.db, session.DID, 50, 0)
    if err != nil {
        h.logger.Printf("Error listing exports: %v", err)
        http.Error(w, "Internal server error", http.StatusInternalServerError)
        return
    }

    // If JSON requested
    if r.Header.Get("Accept") == "application/json" {
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "exports": exports,
            "total":   len(exports),
        })
        return
    }

    // Otherwise render in template
    data := TemplateData{
        Session: session,
        Exports: exports,
    }
    h.renderTemplate(w, r, "export", data)
}

// DownloadExport streams an export as a ZIP file
func (h *Handlers) DownloadExport(w http.ResponseWriter, r *http.Request) {
    session := auth.GetSessionFromContext(r.Context())
    if session == nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    exportID := chi.URLParam(r, "export_id")
    deleteAfter := r.URL.Query().Get("delete_after") == "true"

    // Get export record
    export, err := storage.GetExportByID(h.db, exportID)
    if err != nil {
        http.Error(w, "Export not found", http.StatusNotFound)
        return
    }

    // Verify ownership
    if export.DID != session.DID {
        h.logger.Printf("Security: Unauthorized download attempt - user %s attempted to access export %s owned by %s",
            session.DID, exportID, export.DID)
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    // Check rate limit
    downloadsMu.Lock()
    if activeDownloads[session.DID] >= 10 {
        downloadsMu.Unlock()
        http.Error(w, "Too many concurrent downloads", http.StatusTooManyRequests)
        return
    }
    activeDownloads[session.DID]++
    downloadsMu.Unlock()

    // Cleanup on exit
    defer func() {
        downloadsMu.Lock()
        activeDownloads[session.DID]--
        if activeDownloads[session.DID] <= 0 {
            delete(activeDownloads, session.DID)
        }
        downloadsMu.Unlock()
    }()

    // Check if directory exists
    if _, err := os.Stat(export.DirectoryPath); os.IsNotExist(err) {
        http.Error(w, "Export files not found", http.StatusNotFound)
        return
    }

    // Set headers for ZIP download
    filename := fmt.Sprintf("export-%s.zip", filepath.Base(export.DirectoryPath))
    w.Header().Set("Content-Type", "application/zip")
    w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))

    // Stream ZIP
    pr, pw := io.Pipe()
    go func() {
        defer pw.Close()
        if err := exporter.StreamDirectoryAsZIP(pw, export.DirectoryPath); err != nil {
            h.logger.Printf("Error streaming ZIP: %v", err)
        }
    }()

    if _, err := io.Copy(w, pr); err != nil {
        h.logger.Printf("Error sending ZIP: %v", err)
        return
    }

    // Delete after successful download
    if deleteAfter {
        if err := h.deleteExportInternal(export); err != nil {
            h.logger.Printf("Warning: Failed to delete export after download: %v", err)
        }
    }

    h.logger.Printf("Export downloaded: user=%s, export=%s, size=%d", session.DID, exportID, export.SizeBytes)
}

// DeleteExport removes an export
func (h *Handlers) DeleteExport(w http.ResponseWriter, r *http.Request) {
    session := auth.GetSessionFromContext(r.Context())
    if session == nil {
        http.Error(w, "Unauthorized", http.StatusUnauthorized)
        return
    }

    exportID := chi.URLParam(r, "export_id")

    // Get export record
    export, err := storage.GetExportByID(h.db, exportID)
    if err != nil {
        http.Error(w, "Export not found", http.StatusNotFound)
        return
    }

    // Verify ownership
    if export.DID != session.DID {
        h.logger.Printf("Security: Unauthorized delete attempt - user %s attempted to delete export %s owned by %s",
            session.DID, exportID, export.DID)
        http.Error(w, "Forbidden", http.StatusForbidden)
        return
    }

    if err := h.deleteExportInternal(export); err != nil {
        h.logger.Printf("Error deleting export: %v", err)
        http.Error(w, "Failed to delete export", http.StatusInternalServerError)
        return
    }

    h.logger.Printf("Export deleted: user=%s, export=%s", session.DID, exportID)

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{
        "message":   "Export deleted successfully",
        "export_id": exportID,
    })
}

func (h *Handlers) deleteExportInternal(export *models.ExportRecord) error {
    // Delete from filesystem
    if err := os.RemoveAll(export.DirectoryPath); err != nil {
        h.logger.Printf("Warning: Failed to delete export directory %s: %v", export.DirectoryPath, err)
    }

    // Delete from database
    return storage.DeleteExport(h.db, export.ID)
}
```

---

## Step 7: Add Routes

Register new endpoints.

### Update Router

**File**: `internal/web/router.go` or wherever routes are defined

```go
// Export routes
r.Get("/export", handlers.ExportPage)          // Existing
r.Get("/export/list", handlers.ListExports)    // New
r.Post("/export/start", handlers.StartExport)  // Existing
r.Get("/export/progress/{job_id}", handlers.ExportProgress) // Existing
r.Get("/export/download/{export_id}", handlers.DownloadExport) // New
r.Delete("/export/delete/{export_id}", handlers.DeleteExport)  // New
```

---

## Step 8: Update UI Template

Modify the export page to show export list.

### Update Template

**File**: `internal/web/templates/pages/export.html`

Add after the export form:

```html
<!-- Export List -->
{{if .Exports}}
<article>
    <header><strong>Your Exports</strong></header>
    <table>
        <thead>
            <tr>
                <th>Created</th>
                <th>Format</th>
                <th>Posts</th>
                <th>Media</th>
                <th>Size</th>
                <th>Actions</th>
            </tr>
        </thead>
        <tbody>
            {{range .Exports}}
            <tr>
                <td>{{.CreatedAt.Format "2006-01-02 15:04"}}</td>
                <td>{{.Format | upper}}</td>
                <td>{{.PostCount}}</td>
                <td>{{.MediaCount}}</td>
                <td>{{.HumanSize}}</td>
                <td>
                    <a href="/export/download/{{.ID | urlquery}}" role="button" class="secondary">Download</a>
                    <button hx-delete="/export/delete/{{.ID | urlquery}}"
                            hx-confirm="Delete this export? This action cannot be undone."
                            hx-target="closest tr"
                            hx-swap="outerHTML">
                        Delete
                    </button>
                </td>
            </tr>
            {{end}}
        </tbody>
    </table>
</article>
{{else}}
<p>No exports yet. Create your first export above.</p>
{{end}}
```

---

## Step 9: Test

### Unit Tests

```bash
go test ./internal/storage -v
go test ./internal/exporter -v
go test ./internal/web/handlers -v
```

### Integration Test

1. Start the server: `go run main.go`
2. Navigate to `/export`
3. Create an export
4. Verify it appears in the export list
5. Click "Download" and verify ZIP file
6. Click "Delete" and verify removal

---

## Step 10: Deploy

1. Run database migration
2. Deploy updated code
3. Monitor logs for errors
4. Test download/delete functionality

---

## Troubleshooting

### Export not showing in list

- Check if migration ran: `SELECT * FROM exports;`
- Check if export creation saves record (see `exporter.go` update)

### Download fails

- Check file permissions on `./exports/` directory
- Verify export directory exists: `ls -la ./exports/{did}/{timestamp}/`
- Check server logs for ZIP streaming errors

### Rate limit issues

- Check `activeDownloads` counter (add debug log)
- Verify cleanup happens on download completion

---

## Next Steps

- Add pagination for export list
- Add bulk delete functionality
- Add export size warnings in UI
- Add scheduled cleanup for old exports
- Add export statistics dashboard

---

## Summary

You've implemented:
- ✅ Database migration for export tracking
- ✅ Storage layer for CRUD operations
- ✅ ZIP streaming for efficient downloads
- ✅ HTTP handlers for download/delete
- ✅ Updated UI with export list
- ✅ Rate limiting for resource protection
- ✅ Security checks for authorization

Total implementation time: ~4-6 hours for experienced Go developer.
