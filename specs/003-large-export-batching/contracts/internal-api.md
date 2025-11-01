# Internal API Contracts: Large Export Batching

**Feature**: 003-large-export-batching
**Date**: 2025-11-01
**Type**: Internal Go API (no external HTTP/REST contracts)

## Overview

This feature modifies internal package APIs for batched export processing. No public-facing API changes. All contracts are Go function signatures within `internal/` packages.

---

## Storage Layer Contract

### Package: `internal/storage`

#### Function: `ListPostsWithDateRange`

**Signature** (unchanged):

```go
func ListPostsWithDateRange(
    db *sql.DB,
    did string,
    dateRange models.DateRange,
    limit int,
    offset int,
) ([]models.Post, error)
```

**Parameters**:

| Parameter | Type | Description | Constraints |
|-----------|------|-------------|-------------|
| `db` | `*sql.DB` | SQLite database connection | Non-nil |
| `did` | `string` | Decentralized identifier (user ID) | Non-empty |
| `dateRange` | `models.DateRange` | Start/end timestamps for filtering | Valid time range |
| `limit` | `int` | Max posts to return (0 = unlimited) | ≥ 0 |
| `offset` | `int` | Number of posts to skip | ≥ 0 |

**Returns**:

| Type | Description |
|------|-------------|
| `[]models.Post` | Slice of posts matching criteria |
| `error` | Database error or nil on success |

**Behavior Changes** (implementation-only):

**Before**:
- `limit=0, offset=0` → return all posts (single query)
- No guaranteed ordering (undefined pagination behavior)

**After**:
- `limit=0, offset=0` → return all posts (backward compatible)
- `limit=1000, offset=0` → return first 1,000 posts
- `limit=1000, offset=1000` → return posts 1,001-2,000
- **Always** uses `ORDER BY created_at DESC, uri ASC` for deterministic pagination
- Empty slice `[]` when offset exceeds total post count

**Error Conditions**:

| Condition | Error |
|-----------|-------|
| Database connection closed | `sql.ErrConnDone` |
| Invalid SQL syntax | `sqlite3.Error` |
| DID not found | No error (empty slice) |
| Invalid date range | No error (empty slice) |

**Performance Guarantee**:

- Query time < 100ms for batch size 1,000 (with existing indexes)
- Memory: O(limit) - proportional to batch size, not total post count

---

## Exporter Layer Contract

### Package: `internal/exporter`

#### Function: `Run`

**Signature** (unchanged):

```go
func Run(
    db *sql.DB,
    job *models.ExportJob,
    progressChan chan<- models.ExportProgress,
) error
```

**Parameters**:

| Parameter | Type | Description | Constraints |
|-----------|------|-------------|-------------|
| `db` | `*sql.DB` | Database connection | Non-nil |
| `job` | `*models.ExportJob` | Export job configuration | Non-nil, valid Options |
| `progressChan` | `chan<- models.ExportProgress` | Progress updates (optional) | Can be nil |

**Returns**:

| Type | Description |
|------|-------------|
| `error` | Export error or nil on success |

**Behavior Changes**:

**Before**:
- Single `ListPostsWithDateRange()` call (all posts at once)
- Single progress update after all posts processed
- Memory usage: O(N) where N = total posts

**After**:
- Multiple `ListPostsWithDateRange()` calls (1,000 posts per batch)
- Progress update after each batch completes
- Memory usage: O(1,000) - constant regardless of total posts

**Progress Updates** (sent via `progressChan`):

```go
type ExportProgress struct {
    PostsProcessed int          // Updated after each batch (+1000)
    PostsTotal     int          // Set once at start (count query)
    MediaCopied    int          // Updated during media phase (unchanged)
    MediaTotal     int          // Set once at media start (unchanged)
    Status         ExportStatus // queued/running/completed/failed
    Error          string       // Error message if failed
}
```

**Update frequency**: Every 1,000 posts (~0.5-1 seconds at 1,500-2,000 posts/sec)

**Error Handling**:

| Error Type | Behavior |
|------------|----------|
| Database error during batch | Stop immediately, clean up output file, return error |
| I/O error during write | Stop immediately, clean up output file, return error |
| Context cancellation | N/A (not yet implemented) |

**File Cleanup Guarantee**:

- On success: Output files remain in `job.OutputDir`
- On failure: Partial output files deleted automatically (defer cleanup)

---

#### Function: `ExportToJSON`

**Signature** (unchanged):

```go
func ExportToJSON(
    posts []models.Post,
    outputPath string,
) error
```

**NEW Internal Function**: `ExportToJSONStreaming` (batched variant)

```go
func ExportToJSONStreaming(
    db *sql.DB,
    fetchBatch func(offset int) ([]models.Post, error),
    outputPath string,
    totalPosts int,
) error
```

**Parameters**:

| Parameter | Type | Description |
|-----------|------|-------------|
| `db` | `*sql.DB` | Database connection for batch queries |
| `fetchBatch` | `func(offset int) ([]Post, error)` | Callback to fetch next batch |
| `outputPath` | `string` | Absolute path to output JSON file |
| `totalPosts` | `int` | Total posts to export (for loop termination) |

**Output Format** (byte-identical to original):

```json
[
  {
    "uri": "at://did:plc:abc123/app.bsky.feed.post/xyz789",
    "cid": "bafyreiabc123...",
    "text": "Post content",
    "created_at": "2025-11-01T12:00:00Z",
    ...
  },
  {
    ...
  }
]
```

**Guarantees**:

- Valid JSON array (proper brackets, commas between objects)
- Pretty-printed with 2-space indentation (matches original)
- UTF-8 encoding
- Posts in `ORDER BY created_at DESC, uri ASC` order

---

#### Function: `ExportToCSV`

**Signature** (unchanged):

```go
func ExportToCSV(
    posts []models.Post,
    outputPath string,
) error
```

**NEW Internal Function**: `ExportToCSVStreaming` (batched variant)

```go
func ExportToCSVStreaming(
    db *sql.DB,
    fetchBatch func(offset int) ([]models.Post, error),
    outputPath string,
    totalPosts int,
) error
```

**Parameters**: Same as `ExportToJSONStreaming`

**Output Format** (byte-identical to original):

```
<UTF-8 BOM>
uri,cid,did,text,created_at,indexed_at,has_media,like_count,repost_count,reply_count,quote_count,is_reply,reply_parent,embed_type,embed_data
at://...,bafyrei...,did:plc:...,"Post content",2025-11-01T12:00:00Z,...
...
```

**Guarantees**:

- RFC 4180 compliant CSV
- UTF-8 BOM (0xEF, 0xBB, 0xBF) for Excel compatibility
- Proper escaping of quotes, commas, newlines in text fields
- Posts in `ORDER BY created_at DESC, uri ASC` order

---

## Models Layer Contract

### Package: `internal/models`

**No changes to struct definitions.**

Existing contracts remain valid:

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

type ExportProgress struct {
    PostsProcessed int          // Incremented after each batch
    PostsTotal     int          // Set once at export start
    MediaCopied    int          // Incremented per media file
    MediaTotal     int          // Set at media phase start
    Status         ExportStatus // queued → running → completed/failed
    Error          string       // Non-empty on failure
}
```

**Behavioral contracts**:

- `PostsProcessed` increments by batch size (1,000) until reaching `PostsTotal`
- `PostsTotal` set via initial COUNT query before batching starts
- `Status` transitions: `queued → running → completed` (success) or `queued → running → failed` (error)

---

## Testing Contracts

### Unit Test Interface

**Storage layer tests** (`internal/storage/posts_test.go`):

```go
func TestListPostsWithDateRange_Pagination(t *testing.T) {
    // Given: Database with 2,500 posts
    // When: Query with limit=1000, offset=0
    // Then: Returns first 1,000 posts ordered by created_at DESC

    // When: Query with limit=1000, offset=1000
    // Then: Returns posts 1,001-2,000

    // When: Query with limit=1000, offset=2000
    // Then: Returns last 500 posts

    // When: Query with limit=1000, offset=3000
    // Then: Returns empty slice []
}
```

**Exporter tests** (`internal/exporter/exporter_test.go`):

```go
func TestRun_BatchedExport(t *testing.T) {
    // Given: Database with 5,000 posts
    // When: Run export with JSON format
    // Then: Progress updates sent 5 times (after each 1,000-post batch)
    // And: Output JSON file is valid and contains all 5,000 posts
    // And: Memory usage < 50MB during export
}

func TestExportToJSON_ByteIdentical(t *testing.T) {
    // Given: Database with 2,000 posts
    // When: Export with batched implementation
    // Then: Output byte-identical to non-batched export
}
```

---

## Non-Functional Contracts

### Performance Guarantees (from FR/SC requirements)

| Metric | Requirement | Verification |
|--------|-------------|--------------|
| Export throughput | 1,500-2,000 posts/sec | Integration test |
| Memory usage | < 500MB for any archive size | Runtime profiling |
| Progress update frequency | ≥ every 5 seconds | Channel message count |
| Small archive performance | Same as v0.3.0 (< 2 sec for 500 posts) | Regression test |

### Reliability Guarantees

| Guarantee | Contract |
|-----------|----------|
| Output correctness | Byte-identical to non-batched export (FR-005) |
| Error recovery | Partial files cleaned up on failure (FR-006) |
| Last batch handling | Correctly processes final batch < 1,000 posts (FR-008) |
| Empty archive | Exports valid empty JSON array `[]` or CSV with header only |

---

## Backward Compatibility

**Breaking changes**: NONE

**Compatible changes**:
- `ListPostsWithDateRange()` now guarantees deterministic ordering (improvement)
- Export functions produce identical output (transparent refactoring)
- Progress updates more frequent (improvement)

**Deprecations**: NONE

---

## Security Considerations

**No security impact** - internal refactoring only:

- No new authentication requirements
- No new authorization checks
- No new data exposure
- File permissions unchanged (existing `os.Create()` behavior)
- SQL injection protection unchanged (existing parameterized queries)

---

## Summary

**Contract changes**: Zero external API changes

**Internal behavior changes**:
- Database queries: Add pagination with deterministic ordering
- Export functions: Streaming writes instead of single allocation
- Progress reporting: Incremental updates every batch

**Compatibility**: 100% backward compatible
