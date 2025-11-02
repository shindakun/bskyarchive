# Data Model: Export Download & Management

**Feature**: 005-export-download
**Date**: 2025-11-01

This document defines the entities, relationships, and data structures for the export download and management feature.

---

## Entity: ExportRecord

Represents a completed export with metadata for tracking, listing, and authorization.

### Fields

| Field | Type | Constraints | Description |
|-------|------|-------------|-------------|
| `ID` | string | PRIMARY KEY, format: `{did}/{timestamp}` | Unique identifier matching directory structure |
| `DID` | string | NOT NULL, indexed | Owner's DID for security and querying |
| `Format` | string | NOT NULL, enum: `"json"` \| `"csv"` | Export format type |
| `CreatedAt` | time.Time | NOT NULL, indexed DESC | When export was created |
| `DirectoryPath` | string | NOT NULL | Full filesystem path to export directory |
| `PostCount` | int | NOT NULL, >= 0 | Number of posts in this export |
| `MediaCount` | int | DEFAULT 0, >= 0 | Number of media files included |
| `SizeBytes` | int64 | DEFAULT 0, >= 0 | Total size of export in bytes (calculated) |
| `DateRangeStart` | *time.Time | NULLABLE | Filter start date (nil = no filter) |
| `DateRangeEnd` | *time.Time | NULLABLE | Filter end date (nil = no filter) |
| `ManifestPath` | string | NULLABLE | Path to manifest.json file |

### Indexes

```sql
CREATE INDEX idx_exports_did ON exports(did);
CREATE INDEX idx_exports_created_at ON exports(created_at DESC);
CREATE INDEX idx_exports_did_created ON exports(did, created_at DESC);
```

### Validation Rules

- `ID` must match pattern: `did:plc:[a-z0-9]+/\d{4}-\d{2}-\d{2}_\d{2}-\d{2}-\d{2}`
- `Format` must be one of: `"json"`, `"csv"`
- `DirectoryPath` must start with `./exports/` (security validation)
- `PostCount` must be >= 0
- `MediaCount` must be >= 0
- `SizeBytes` must be >= 0
- If `DateRangeStart` and `DateRangeEnd` are both set, `DateRangeEnd` must be after `DateRangeStart`

### State Transitions

Exports have a simple lifecycle (no state machine):

1. **Creation**: Export created via exporter → Record inserted into database
2. **Active**: Export exists on disk and in database (can be downloaded/deleted)
3. **Deleted**: User triggers deletion → Record removed from database + directory removed from disk

No intermediate states (no "pending deletion", "archived", etc.).

---

## Entity: DownloadSession (In-Memory Only)

Tracks active download operations for rate limiting. Not persisted to database.

### Fields

| Field | Type | Description |
|-------|------|-------------|
| `DID` | string | User performing download |
| `ExportID` | string | Which export is being downloaded |
| `StartedAt` | time.Time | When download began |

### In-Memory Structure

```go
var (
    activeDownloads = make(map[string]int) // DID -> count
    downloadsMu     sync.RWMutex
)
```

### Rules

- Maximum 10 concurrent downloads per DID
- Cleaned up automatically when download completes (success or error)
- Not persisted across server restarts (acceptable - transient data)

---

## Relationships

### User (Session) → ExportRecord

- **Relationship Type**: One-to-Many
- **Cardinality**: One user (DID) can have 0 to N exports
- **Foreign Key**: `ExportRecord.DID` references `Session.DID` (logical, not enforced by DB)
- **Cascade**: When user is deleted, their exports should be deleted (manual cleanup, not FK constraint)

**Query Pattern**:
```sql
-- List all exports for a user
SELECT * FROM exports WHERE did = ? ORDER BY created_at DESC;
```

### ExportRecord → Filesystem

- **Relationship Type**: One-to-One
- **Mapping**: `ExportRecord.DirectoryPath` points to directory on disk
- **Consistency**: Record must always point to valid directory (handle missing directories gracefully)
- **Cleanup**: When `ExportRecord` is deleted, corresponding directory must be removed

**Integrity Check**:
```go
// Verify export directory exists
if _, err := os.Stat(export.DirectoryPath); os.IsNotExist(err) {
    return ErrExportNotFound
}
```

---

## Go Type Definitions

### ExportRecord

```go
// ExportRecord represents a completed export in the database
type ExportRecord struct {
    ID             string     `json:"id"`
    DID            string     `json:"did"`
    Format         string     `json:"format"` // "json" or "csv"
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
    if e.ID == "" {
        return fmt.Errorf("ID is required")
    }
    if e.DID == "" {
        return fmt.Errorf("DID is required")
    }
    if e.Format != "json" && e.Format != "csv" {
        return fmt.Errorf("format must be 'json' or 'csv'")
    }
    if e.PostCount < 0 {
        return fmt.Errorf("post count must be >= 0")
    }
    if e.MediaCount < 0 {
        return fmt.Errorf("media count must be >= 0")
    }
    if e.SizeBytes < 0 {
        return fmt.Errorf("size must be >= 0")
    }
    if e.DateRangeStart != nil && e.DateRangeEnd != nil {
        if e.DateRangeEnd.Before(*e.DateRangeStart) {
            return fmt.Errorf("date range end must be after start")
        }
    }
    if !strings.HasPrefix(e.DirectoryPath, "./exports/") {
        return fmt.Errorf("invalid directory path (security)")
    }
    return nil
}

// HumanSize returns size in human-readable format (KB, MB, GB)
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

// DateRangeString returns formatted date range or "All posts"
func (e *ExportRecord) DateRangeString() string {
    if e.DateRangeStart == nil && e.DateRangeEnd == nil {
        return "All posts"
    }
    if e.DateRangeStart != nil && e.DateRangeEnd != nil {
        return fmt.Sprintf("%s to %s",
            e.DateRangeStart.Format("2006-01-02"),
            e.DateRangeEnd.Format("2006-01-02"))
    }
    if e.DateRangeStart != nil {
        return fmt.Sprintf("From %s", e.DateRangeStart.Format("2006-01-02"))
    }
    return fmt.Sprintf("Until %s", e.DateRangeEnd.Format("2006-01-02"))
}
```

### ExportListResponse

```go
// ExportListResponse is returned by the list exports API
type ExportListResponse struct {
    Exports []ExportRecord `json:"exports"`
    Total   int            `json:"total"`
}
```

### DownloadRequest

```go
// DownloadRequest represents a download operation request
type DownloadRequest struct {
    ExportID     string `json:"export_id"`
    DeleteAfter  bool   `json:"delete_after"` // Optional: delete after successful download
}
```

---

## Database Schema (SQLite)

### Table: `exports`

```sql
CREATE TABLE IF NOT EXISTS exports (
    id TEXT PRIMARY KEY,                   -- Format: {did}/{timestamp}
    did TEXT NOT NULL,                     -- Owner's DID
    format TEXT NOT NULL,                  -- 'json' or 'csv'
    created_at INTEGER NOT NULL,           -- Unix timestamp
    directory_path TEXT NOT NULL,          -- Full path to export directory
    post_count INTEGER NOT NULL,           -- Number of posts
    media_count INTEGER DEFAULT 0,         -- Number of media files
    size_bytes INTEGER DEFAULT 0,          -- Total size in bytes
    date_range_start INTEGER,              -- Filter start (unix timestamp, nullable)
    date_range_end INTEGER,                -- Filter end (unix timestamp, nullable)
    manifest_path TEXT,                    -- Path to manifest.json

    -- Constraints
    CHECK (post_count >= 0),
    CHECK (media_count >= 0),
    CHECK (size_bytes >= 0)
);

-- Indexes for query performance
CREATE INDEX IF NOT EXISTS idx_exports_did
    ON exports(did);

CREATE INDEX IF NOT EXISTS idx_exports_created_at
    ON exports(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_exports_did_created
    ON exports(did, created_at DESC);
```

### Migration Script

```go
// Migration: Add exports table
// Version: 005
// Description: Track completed exports for download/management

func MigrateExportsTable(db *sql.DB) error {
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

---

## Data Flow Diagrams

### Create Export (Updated)

```
User clicks "Start Export"
    ↓
Handler (StartExport)
    ↓
Exporter.Run()
    ↓
[Create files on disk]
    ↓
Calculate size (new)
    ↓
Insert ExportRecord into DB (new)
    ↓
Return success
```

### List Exports (New)

```
User visits /export page
    ↓
Handler (ListExports)
    ↓
Query: SELECT * FROM exports WHERE did = ? ORDER BY created_at DESC
    ↓
For each record: Check if directory exists (optional integrity check)
    ↓
Render HTML table with download/delete buttons
```

### Download Export (New)

```
User clicks "Download"
    ↓
Handler (DownloadExport)
    ↓
Verify session & DID ownership
    ↓
Check rate limit (activeDownloads[did] < 10)
    ↓
Create io.Pipe()
    ↓
Goroutine: Stream ZIP creation
    ↓
http.ServeContent(pipe reader)
    ↓
If delete_after=true: Delete export
    ↓
Cleanup rate limit tracker
```

### Delete Export (New)

```
User clicks "Delete" → Confirmation dialog
    ↓
Handler (DeleteExport) with CSRF check
    ↓
Verify session & DID ownership
    ↓
os.RemoveAll(export.DirectoryPath)
    ↓
DELETE FROM exports WHERE id = ?
    ↓
Return 200 OK (or error)
    ↓
HTMX removes table row from UI
```

---

## Error Handling

### Common Error Cases

| Error Scenario | HTTP Status | Response | Recovery |
|----------------|-------------|----------|----------|
| Export not found in DB | 404 Not Found | "Export not found" | User sees error, can retry |
| Export directory missing | 404 Not Found | "Export files not found" | Auto-cleanup DB record (optional) |
| Unauthorized access | 403 Forbidden | "Forbidden" | Log security event, reject request |
| Rate limit exceeded | 429 Too Many Requests | "Too many downloads" | User waits, retries later |
| Disk error during ZIP | 500 Internal Server Error | "Failed to create archive" | Log error, cleanup partial download |
| Deletion failure | 500 Internal Server Error | "Failed to delete export" | Log error, mark as orphaned |

---

## Summary

This data model introduces:
1. **ExportRecord entity** to track completed exports in SQLite
2. **DownloadSession tracking** (in-memory) for rate limiting
3. **One-to-many relationship** between User (DID) and ExportRecords
4. **One-to-one relationship** between ExportRecord and filesystem directory
5. **Migration script** to add `exports` table with indexes
6. **Validation rules** for data integrity and security
7. **Go type definitions** with helper methods for formatting

All fields align with existing export structure. No breaking changes to current export format.
