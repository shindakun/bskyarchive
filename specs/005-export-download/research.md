# Research: Export Download & Management

**Feature**: 005-export-download
**Date**: 2025-11-01
**Status**: Complete

This document consolidates research findings that inform implementation decisions for the export download and management feature.

---

## 1. Go ZIP Streaming for Large Files

### Decision: Use archive/zip with io.Pipe for streaming

**Rationale**:
- Go's `archive/zip` package supports streaming writes via `io.Writer` interface
- Using `io.Pipe()` creates a producer-consumer pattern where ZIP creation happens in a goroutine while HTTP writes to response simultaneously
- This approach never loads the entire ZIP into memory - files are read, compressed, and written incrementally
- No third-party dependencies needed - stdlib is sufficient and battle-tested

**Implementation Pattern**:
```go
// Pseudocode
pr, pw := io.Pipe()
go func() {
    defer pw.Close()
    zipWriter := zip.NewWriter(pw)
    // Walk directory and add files incrementally
    // Each file is read in chunks and written to ZIP
    zipWriter.Close()
}()
http.ServeContent(w, r, "export.zip", time.Now(), pr)
```

**Alternatives Considered**:
- **Pre-create ZIP files**: Rejected because it doubles storage requirements and adds complexity
- **Third-party ZIP libraries (e.g., archiver)**: Rejected because stdlib is sufficient and reduces dependencies
- **Temporary file approach**: Rejected because it's slower and still requires disk space

**References**:
- Go archive/zip docs: https://pkg.go.dev/archive/zip
- io.Pipe pattern: https://pkg.go.dev/io#Pipe

---

## 2. SQLite Schema for Export Tracking

### Decision: Create `exports` table with metadata indexed by DID and timestamp

**Rationale**:
- Need to track completed exports for listing, filtering, and security
- Current implementation stores exports in `./exports/{did}/{timestamp}/` but has no database tracking
- Adding a table enables efficient queries for user's exports without filesystem traversal
- Indexes on `did` and `created_at` optimize common queries (list user's exports, find by ID)

**Schema Design**:
```sql
CREATE TABLE exports (
    id TEXT PRIMARY KEY,              -- Format: {did}/{timestamp}
    did TEXT NOT NULL,                -- Owner's DID for security
    format TEXT NOT NULL,             -- 'json' or 'csv'
    created_at INTEGER NOT NULL,      -- Unix timestamp
    directory_path TEXT NOT NULL,     -- Full path to export directory
    post_count INTEGER NOT NULL,      -- Number of posts in export
    media_count INTEGER DEFAULT 0,    -- Number of media files
    size_bytes INTEGER DEFAULT 0,     -- Total size in bytes (calculated)
    date_range_start INTEGER,         -- Filter start (unix timestamp, nullable)
    date_range_end INTEGER,           -- Filter end (unix timestamp, nullable)
    manifest_path TEXT                -- Path to manifest.json
);

CREATE INDEX idx_exports_did ON exports(did);
CREATE INDEX idx_exports_created_at ON exports(created_at DESC);
CREATE INDEX idx_exports_did_created ON exports(did, created_at DESC);
```

**Alternatives Considered**:
- **No database tracking (filesystem only)**: Rejected because listing requires expensive directory traversal and parsing manifests
- **Separate table per user**: Rejected because single table with DID index is simpler and standard practice
- **Store in existing posts table**: Rejected because exports are logically separate entities with different lifecycle

**Migration Strategy**:
- Add migration script to create table
- Optionally scan existing `./exports/` directory and backfill table (handle missing/incomplete exports gracefully)

---

## 3. Security Patterns for File Download Authorization

### Decision: DID-based ownership check before serving files

**Rationale**:
- Exports contain user's private posts and media - unauthorized access is a critical security violation
- Must verify requesting user's DID matches export's owner DID before serving
- Use existing session management (already implemented in auth package)
- Prevent path traversal attacks by validating export ID format

**Implementation Pattern**:
```go
// 1. Extract session from context
session := auth.GetSessionFromContext(r.Context())

// 2. Parse export ID from URL (format: {did}/{timestamp})
exportID := chi.URLParam(r, "export_id")

// 3. Query database for export metadata
export, err := storage.GetExportByID(db, exportID)

// 4. Verify ownership
if export.DID != session.DID {
    logger.Printf("Security: Unauthorized download attempt - user %s attempted to access export %s owned by %s",
        session.DID, exportID, export.DID)
    http.Error(w, "Forbidden", http.StatusForbidden)
    return
}

// 5. Validate directory path (prevent traversal)
if !strings.HasPrefix(export.DirectoryPath, "./exports/") {
    logger.Printf("Security: Suspicious export path: %s", export.DirectoryPath)
    http.Error(w, "Invalid export", http.StatusBadRequest)
    return
}

// 6. Serve ZIP stream
```

**Security Checklist**:
- ✅ Require authentication (reject if no session)
- ✅ Verify DID ownership (exports table has did column)
- ✅ Validate export ID format (prevent path traversal like `../../etc/passwd`)
- ✅ Log access attempts (audit trail for security reviews)
- ✅ Rate limiting (max 10 concurrent downloads per user)
- ✅ CSRF protection on state-changing operations (delete)

**Alternatives Considered**:
- **Signed URLs with expiration**: Rejected as over-engineering for local-first architecture
- **File-level ACLs**: Rejected because database-driven authz is simpler and more flexible
- **JWT tokens for download**: Rejected because session cookies already provide secure authentication

---

## 4. Memory-Efficient ZIP Creation Techniques

### Decision: Stream files with buffered reading, no pre-compression

**Rationale**:
- Reading entire files into memory fails for large exports (10GB+)
- Use `io.Copy` with buffered reader to stream file contents into ZIP writer in chunks
- Let `archive/zip` handle compression incrementally (DEFLATE algorithm)
- Walk directory tree depth-first, processing one file at a time

**Implementation Pattern**:
```go
func StreamZIPDirectory(w io.Writer, rootPath string) error {
    zipWriter := zip.NewWriter(w)
    defer zipWriter.Close()

    return filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
        if err != nil || info.IsDir() {
            return err
        }

        // Calculate relative path for ZIP entry
        relPath, _ := filepath.Rel(rootPath, path)

        // Create ZIP entry header
        header, err := zip.FileInfoHeader(info)
        header.Name = relPath
        header.Method = zip.Deflate

        // Write header to ZIP
        writer, err := zipWriter.CreateHeader(header)

        // Stream file contents in chunks (io.Copy uses 32KB buffer)
        file, err := os.Open(path)
        defer file.Close()

        _, err = io.Copy(writer, file) // Streams in chunks, never loads full file
        return err
    })
}
```

**Memory Profile**:
- Stack: ~1MB (goroutine overhead, local variables)
- Heap: ~64KB (io.Copy buffer) + ~32KB (compression buffer) = ~100KB per file being processed
- Total: <500MB even for 10GB+ exports because files are processed sequentially

**Alternatives Considered**:
- **Pre-compress files**: Rejected because it requires double storage and doesn't improve memory usage
- **Parallel file reading**: Rejected because it increases memory footprint (N files × buffer size)
- **Store compression**: Rejected because it increases ZIP size significantly (no compression)

**Performance Characteristics**:
- Throughput: ~50-100 MB/s (limited by disk I/O and compression)
- Latency: <1 second to start streaming (header overhead is minimal)
- Scalability: O(n) time with O(1) memory where n = total file size

---

## 5. Export Deletion Best Practices

### Decision: Soft delete with os.RemoveAll and database record cleanup

**Rationale**:
- Use `os.RemoveAll(exportDir)` to recursively delete export directory
- Remove database record in same transaction or immediately after
- Show confirmation dialog in UI (prevent accidental deletion)
- Log deletion operations for audit trail

**Error Handling**:
- If disk deletion fails but DB delete succeeds: Show error, mark record as "orphaned" in logs
- If DB deletion fails but disk delete succeeds: Rollback disk delete or mark as stale
- If export is being downloaded during deletion: Proceed with deletion (download may fail - acceptable)

**Implementation Pattern**:
```go
func DeleteExport(db *sql.DB, exportID string) error {
    // 1. Get export metadata
    export, err := storage.GetExportByID(db, exportID)

    // 2. Delete from filesystem
    if err := os.RemoveAll(export.DirectoryPath); err != nil {
        logger.Printf("Warning: Failed to delete export directory %s: %v", export.DirectoryPath, err)
        // Continue anyway - remove DB record even if files are stuck
    }

    // 3. Delete from database
    _, err = db.Exec("DELETE FROM exports WHERE id = ?", exportID)
    return err
}
```

**Alternatives Considered**:
- **Soft delete flag**: Rejected because export files should be immediately removed for disk space
- **Scheduled cleanup**: Rejected because immediate deletion is simpler and matches user expectations
- **Trash/recycle bin**: Rejected as over-engineering for MVP

---

## 6. UI/UX Patterns for Export Management

### Decision: Table-based listing with inline actions (HTMX for interactivity)

**Rationale**:
- Current app uses Pico.css + HTMX - maintain consistency
- Table format presents multiple exports clearly with sortable columns
- Inline download/delete buttons reduce clicks
- HTMX enables dynamic updates without full page reload

**UI Components**:
```html
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
    <!-- For each export -->
    <tr>
      <td>2025-11-01 14:32</td>
      <td>JSON</td>
      <td>1,234</td>
      <td>567</td>
      <td>123 MB</td>
      <td>
        <a href="/export/download/{id}" role="button">Download</a>
        <button hx-delete="/export/delete/{id}"
                hx-confirm="Delete this export? This cannot be undone.">
          Delete
        </button>
      </td>
    </tr>
  </tbody>
</table>
```

**Alternatives Considered**:
- **Card-based layout**: Rejected because tables are more space-efficient for lists
- **React/Vue frontend**: Rejected to maintain simplicity and avoid build complexity
- **Infinite scroll**: Rejected because pagination is simpler and users rarely have 100+ exports

---

## 7. Rate Limiting for Downloads

### Decision: In-memory concurrent download tracking per user

**Rationale**:
- Prevent resource exhaustion from malicious or accidental parallel downloads
- Track active downloads in memory (map[DID]int) - simple and fast
- Limit: Max 10 concurrent downloads per user
- Clean up tracking when download completes or errors

**Implementation Pattern**:
```go
var (
    activeDownloads = make(map[string]int) // DID -> count
    downloadsMu     sync.RWMutex
)

func TrackDownload(did string) (release func(), err error) {
    downloadsMu.Lock()
    defer downloadsMu.Unlock()

    if activeDownloads[did] >= 10 {
        return nil, fmt.Errorf("too many concurrent downloads")
    }

    activeDownloads[did]++

    release = func() {
        downloadsMu.Lock()
        activeDownloads[did]--
        if activeDownloads[did] <= 0 {
            delete(activeDownloads, did)
        }
        downloadsMu.Unlock()
    }

    return release, nil
}
```

**Alternatives Considered**:
- **No rate limiting**: Rejected because it allows resource exhaustion
- **Global limit (not per-user)**: Rejected because it enables DoS (one user can block others)
- **Token bucket**: Rejected as over-engineering for this use case
- **Database-backed tracking**: Rejected because it's slower and memory is sufficient

---

## 8. Delete After Download Implementation

### Decision: Client-side flag with server validation

**Rationale**:
- User sets checkbox "Delete after download" in UI
- Client includes `?delete_after=true` query param on download URL
- Server completes download, THEN deletes export if param is true
- If download fails (network error, user cancels), export is NOT deleted (server never knows download failed)

**Implementation Consideration**:
This creates a race condition: if download fails client-side, user must manually delete. This is acceptable because:
1. Safety first: Better to leave orphaned export than delete before confirming download
2. User can always manually delete afterward
3. Network failures are rare enough to not optimize for

**Alternative Approach**:
Two-phase commit (download → client confirms → server deletes) adds complexity and requires client-side JavaScript logic. Rejected for MVP simplicity.

---

## Summary of Key Decisions

| Area | Decision | Rationale |
|------|----------|-----------|
| ZIP Streaming | io.Pipe + archive/zip | Memory-efficient, stdlib, proven pattern |
| Export Tracking | SQLite table with DID index | Fast queries, security, metadata storage |
| Authorization | DID-based ownership check | Secure, simple, consistent with existing auth |
| Memory Efficiency | Buffered streaming (io.Copy) | Handles 10GB+ exports with <500MB memory |
| Deletion | os.RemoveAll + DB cleanup | Simple, immediate, logs for audit |
| UI Pattern | Pico.css table + HTMX | Consistent with existing app, minimal JavaScript |
| Rate Limiting | In-memory counter (max 10/user) | Prevents resource exhaustion, simple |
| Delete After Download | Query param with server-side cleanup | Simple, safe (keeps export if download fails) |

All unknowns from Technical Context have been resolved. Ready for Phase 1 (Design).
