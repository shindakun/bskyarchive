# Data Model: Large Export Batching

**Feature**: 003-large-export-batching
**Date**: 2025-11-01

## Overview

This feature introduces minimal data model changes - primarily focused on adding pagination parameters to existing query functions. Core entities (Post, ExportJob, ExportProgress) remain unchanged.

---

## Entity Changes

### 1. Post (No Changes)

**Source**: `internal/models/post.go`

Existing Post struct remains unchanged:

```go
type Post struct {
    URI         string
    CID         string
    DID         string
    Text        string
    CreatedAt   time.Time
    IndexedAt   time.Time
    HasMedia    bool
    LikeCount   int
    RepostCount int
    ReplyCount  int
    QuoteCount  int
    IsReply     bool
    ReplyParent string
    EmbedType   string
    EmbedData   json.RawMessage
    Labels      json.RawMessage
    ArchivedAt  time.Time
}
```

**Rationale**: Batching is transparent to Post model. No new fields required.

---

### 2. ExportJob (No Changes)

**Source**: `internal/models/export.go`

Existing ExportJob struct remains unchanged:

```go
type ExportJob struct {
    ID          string
    UserID      string
    Options     ExportOptions
    Status      ExportStatus
    Progress    ExportProgress
    OutputDir   string
    CreatedAt   time.Time
    CompletedAt *time.Time
}

type ExportOptions struct {
    Format     ExportFormat  // "json" or "csv"
    DID        string
    DateRange  DateRange
    IncludeMedia bool
}
```

**Rationale**: Batching is implementation detail. Job configuration unchanged.

---

### 3. ExportProgress (No Changes)

**Source**: `internal/models/export.go`

Existing ExportProgress struct remains unchanged:

```go
type ExportProgress struct {
    PostsProcessed int
    PostsTotal     int
    MediaCopied    int
    MediaTotal     int
    Status         ExportStatus
    Error          string
}
```

**Rationale**: Existing fields support incremental updates:
- `PostsProcessed` incremented after each batch (every 1,000 posts)
- `PostsTotal` set once at export start (count query)
- Progress reporting already channel-based

**Update Frequency**: Changed from "once at end" to "after each batch" (implementation change, not model change)

---

## Function Signature Changes

### Storage Layer: `internal/storage/posts.go`

#### Current Signature

```go
func ListPostsWithDateRange(db *sql.DB, did string, dateRange models.DateRange,
                            limit, offset int) ([]models.Post, error)
```

**Current behavior**: Called with `limit=0, offset=0` (returns all posts)

#### Modified Behavior

**No signature change** - existing parameters already support pagination!

**New usage pattern**:

```go
// Batched retrieval (NEW usage in exporter.go)
const batchSize = 1000
offset := 0

for {
    batch, err := storage.ListPostsWithDateRange(db, did, dateRange, batchSize, offset)
    if err != nil { return err }
    if len(batch) == 0 { break }

    // Process batch...

    offset += batchSize
    if len(batch) < batchSize { break } // Last batch
}
```

**Implementation change** (internal to `ListPostsWithDateRange`):

```diff
 func ListPostsWithDateRange(db *sql.DB, did string, dateRange models.DateRange,
                             limit, offset int) ([]models.Post, error) {
     query := `SELECT ... FROM posts WHERE did = ? AND created_at BETWEEN ? AND ?`

-    if limit > 0 {
-        query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT %d", limit)
-    }
+    // Always add ORDER BY for deterministic pagination
+    query += " ORDER BY created_at DESC, uri ASC"
+
+    if limit > 0 {
+        query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)
+    }

     rows, err := db.Query(query, did, dateRange.Start, dateRange.End)
     // ...
}
```

**Key change**: Add `uri ASC` to ORDER BY for stable pagination (uri is primary key)

---

## Validation Rules

### FR-001: Batch Size Validation

**Requirement**: "System MUST retrieve posts from database in fixed batches of 1,000 posts per batch"

**Implementation**: Hard-coded constant in `exporter.go`:

```go
const (
    ExportBatchSize = 1000 // Fixed batch size per FR-001
)
```

**Validation**: No runtime validation needed (constant, not configurable)

---

### FR-008: Last Batch Handling

**Requirement**: "System MUST handle batch boundary conditions (last batch with fewer than 1,000 posts)"

**Logic**:

```go
batch, err := storage.ListPostsWithDateRange(db, did, dateRange, batchSize, offset)
if len(batch) == 0 {
    break // No more posts
}

processBatch(batch) // Works for any batch size (1-1000)

if len(batch) < batchSize {
    break // Last partial batch processed
}
```

**Edge cases**:
- **Exact multiple of 1,000**: Last query returns empty slice, loop breaks
- **Not multiple of 1,000**: Last batch has <1,000 posts, processed normally
- **Zero posts total**: First query returns empty slice, loop breaks immediately

---

## State Transitions

### ExportJob Status Flow (No Changes)

```
ExportStatusQueued
    ↓
ExportStatusRunning ← (Start batching loop)
    ↓
    ├─→ [For each batch]
    │       ├─ Fetch batch (LIMIT 1000 OFFSET n)
    │       ├─ Write to file (streaming)
    │       ├─ Update PostsProcessed (+1000)
    │       └─ Send progress via channel
    ↓
ExportStatusCompleted (All batches processed)
    OR
ExportStatusFailed (Database error, I/O error, etc.)
```

**New behavior**: Progress updates sent during `ExportStatusRunning` (not just at end)

---

## Database Schema (No Changes)

SQLite schema remains unchanged. Batching uses existing:
- `posts` table (no new columns)
- `idx_posts_did` index (for DID filtering)
- `idx_posts_created_at` index (for ORDER BY)

**Query performance**: Existing indexes support LIMIT/OFFSET pagination efficiently (see research.md section 3).

---

## Memory Constraints

### Per-Batch Memory Usage

| Component | Size | Calculation |
|-----------|------|-------------|
| Post slice | 1-10 MB | 1,000 posts × 1-10 KB/post |
| JSON encoder buffer | <1 MB | Streaming, minimal buffer |
| CSV writer buffer | <1 MB | 4KB default buffer |
| Database result set | ~2 MB | SQLite row cache |
| **Total per batch** | **~5-15 MB** | Well under 500MB constraint |

**Verification**: FR-004 requires memory <500MB. With 15MB/batch, can handle 30+ concurrent exports before approaching limit.

---

## Relationships (No Changes)

Existing relationships preserved:

```
ExportJob (1) ──has──> (1) ExportProgress
ExportJob (1) ──processes──> (N) Post [via batches]
Post (1) ──has──> (N) MediaFile [unchanged, copied after posts]
```

**Batching impact**: Posts processed in chunks, but relationships unchanged.

---

## API Contract Impact

**Internal APIs only** - no public API changes:
- `storage.ListPostsWithDateRange()` - existing signature, new usage pattern
- `exporter.Run()` - signature unchanged, implementation refactored
- `exporter.ExportToJSON()` - signature unchanged, streaming implementation
- `exporter.ExportToCSV()` - signature unchanged, streaming implementation

**Web API**: No changes to HTTP handlers. Progress updates flow through existing channel mechanism.

---

## Testing Implications

### Data Model Tests

1. **Pagination Correctness**:
   - Test `ListPostsWithDateRange()` with various offsets
   - Verify ORDER BY determinism (same query = same results)
   - Test last batch handling (partial results)

2. **Memory Behavior**:
   - Verify batch slices are GC'd between iterations
   - Test export of 50k+ posts stays under 500MB

3. **Progress Tracking**:
   - Verify `PostsProcessed` increments by batch size
   - Verify `PostsTotal` matches database count

4. **Output Integrity**:
   - Verify batched export output byte-identical to non-batched (FR-005)
   - Test JSON array structure (proper brackets, commas)
   - Test CSV row ordering matches database ORDER BY

---

## Migration Notes

**No database migrations required** - feature uses existing schema.

**Backward compatibility**:
- Old export code can coexist during development (different function names)
- Switch over atomic (change exporter.Run() implementation)
- No data format changes - users see identical export files

---

## Summary

**Data model changes**: Minimal
- Zero new entities
- Zero new fields
- Zero schema changes

**Implementation changes**: Significant
- Query usage pattern (add limit/offset)
- Export functions (streaming writes)
- Progress reporting (incremental updates)

This is an **internal refactoring** for performance, not a data model evolution.
