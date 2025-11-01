# Research: Large Export Batching

**Feature**: 003-large-export-batching
**Date**: 2025-11-01
**Status**: Complete

## Overview

Research findings for implementing memory-efficient batched exports in Go 1.21+ with SQLite backend. Focus areas: streaming I/O patterns, database pagination strategies, and progress reporting mechanisms.

---

## 1. Streaming JSON Array Writes

### Decision

Use manual JSON array construction with `json.Encoder` for individual objects:
1. Write opening `[` bracket to file
2. For each batch: encode objects with `json.Encoder`, add comma separators
3. Write closing `]` bracket after last batch

### Rationale

- **Memory efficiency**: `json.Marshal()` on full slice loads entire dataset into memory (violates 500MB constraint)
- **Streaming capability**: `json.Encoder.Encode()` writes directly to `io.Writer` without intermediate buffer
- **Array construction**: Manual bracket/comma handling allows incremental array building across batches
- **Byte-identical output**: With proper formatting (indentation, newlines), produces identical JSON to non-batched version

### Implementation Pattern

```go
func ExportToJSONStreaming(db *sql.DB, query func(offset int) ([]Post, error),
                           outputPath string, batchSize int) error {
    f, _ := os.Create(outputPath)
    defer f.Close()

    f.WriteString("[\n")
    offset := 0
    first := true

    for {
        batch, _ := query(offset)
        if len(batch) == 0 { break }

        for _, post := range batch {
            if !first { f.WriteString(",\n") }
            first = false

            enc := json.NewEncoder(f)
            enc.SetIndent("", "  ")
            enc.Encode(post)
        }
        offset += batchSize
    }

    f.WriteString("\n]")
    return nil
}
```

### Alternatives Considered

| Alternative | Rejection Reason |
|-------------|-----------------|
| `json.Marshal()` entire slice | Loads all posts into memory - fails on large archives |
| Third-party streaming library (jsoniter, etc.) | Violates stdlib-only constraint in CLAUDE.md |
| `json.Encoder` without manual array construction | Produces JSON lines (JSONL) format, not valid JSON array |
| Buffer batches before encoding | Still accumulates memory across batches |

---

## 2. Streaming CSV Writes

### Decision

Use `encoding/csv` Writer directly with `Flush()` after each batch:

```go
func ExportToCSVStreaming(db *sql.DB, query func(offset int) ([]Post, error),
                          outputPath string, batchSize int) error {
    f, _ := os.Create(outputPath)
    defer f.Close()

    // UTF-8 BOM for Excel compatibility (existing behavior)
    f.Write([]byte{0xEF, 0xBB, 0xBF})

    w := csv.NewWriter(f)
    w.Write([]string{"uri", "text", "created_at", ...}) // header

    offset := 0
    for {
        batch, _ := query(offset)
        if len(batch) == 0 { break }

        for _, post := range batch {
            row := postToCSVRow(post)
            w.Write(row)
        }
        w.Flush() // Write buffered data after each batch
        offset += batchSize
    }

    return w.Error()
}
```

### Rationale

- **Built-in streaming**: `csv.Writer` internally buffers and can flush incrementally
- **RFC 4180 compliance**: Maintains existing CSV format (already implemented)
- **Memory control**: `Flush()` forces write to disk, allowing batch slice to be GC'd
- **Excel compatibility**: Preserves UTF-8 BOM for existing user workflows

### Alternatives Considered

| Alternative | Rejection Reason |
|-------------|-----------------|
| Manual CSV construction | Reinvents wheel; csv.Writer handles escaping, quoting correctly |
| Accumulate full CSV string | Same memory problem as non-batched version |
| Third-party CSV library | Violates stdlib-only constraint |

---

## 3. SQLite Pagination Strategy

### Decision

Use LIMIT/OFFSET pagination with `ORDER BY` for deterministic ordering:

```go
func ListPostsWithPagination(db *sql.DB, did string, dateRange DateRange,
                              limit, offset int) ([]Post, error) {
    query := `
        SELECT uri, cid, did, text, created_at, indexed_at,
               has_media, like_count, repost_count, reply_count,
               quote_count, is_reply, reply_parent, embed_type,
               embed_data, labels, archived_at
        FROM posts
        WHERE did = ?
          AND created_at BETWEEN ? AND ?
        ORDER BY created_at DESC, uri ASC
        LIMIT ? OFFSET ?
    `
    // Execute query with parameters...
}
```

### Rationale

- **Simplicity**: Straightforward implementation with stdlib database/sql
- **Deterministic**: `ORDER BY created_at DESC, uri ASC` ensures stable pagination (uri is primary key)
- **Existing index**: `idx_posts_created_at DESC` index already exists (line 184 in storage/db.go)
- **Performance**: For 100k rows, OFFSET is acceptable (<1s per batch with index)
- **Stateless**: No cursor state to maintain between batches

### Performance Considerations

- **Index usage**: Query uses existing `idx_posts_created_at` index efficiently
- **OFFSET cost**: O(offset) for SQLite, but with 1000-row batches, max offset is 100 for 100k posts
- **Memory**: Each query returns only 1000 rows (~500KB-5MB depending on post size)

### Alternatives Considered

| Alternative | Rejection Reason |
|-------------|-----------------|
| Cursor-based pagination (WHERE created_at < ?) | More complex; requires tracking last_seen values; marginal performance gain |
| Keyset pagination | Requires composite cursor (created_at, uri); adds complexity for multi-column ordering |
| FTS5 pagination | Export doesn't use full-text search; regular table pagination sufficient |
| No ORDER BY | Non-deterministic results; pagination could skip/duplicate rows |

### Caveat: Large Offsets

For archives >100k posts, OFFSET performance degrades. Mitigation:
- Monitor batch query time during implementation
- If query time >500ms, consider cursor-based approach in future iteration
- Current spec targets 100k posts, so LIMIT/OFFSET acceptable

---

## 4. Batch Size Selection

### Decision

Fixed batch size of 1,000 posts (as specified in FR-001).

### Rationale

- **Specification requirement**: FR-001 mandates 1,000 posts per batch
- **Memory calculation**:
  - Average post: ~1-2KB (text + metadata)
  - Worst case (rich embeds): ~10KB per post
  - Batch memory: 1,000 posts × 10KB = 10MB (well under 500MB constraint)
- **Progress granularity**: 1,000-post batches provide reasonable progress updates (every 0.5-1 seconds at 1,500-2,000 posts/sec)
- **Database efficiency**: 1,000-row queries are efficient for SQLite (single page read in many cases)

### Alternatives Considered

| Alternative | Rejection Reason |
|-------------|-----------------|
| Dynamic batch size | Adds complexity; spec requires fixed size |
| Larger batches (5,000+) | Reduces progress update frequency; higher memory per batch |
| Smaller batches (100-500) | More database queries; overhead of batch loop iterations |

---

## 5. Progress Reporting Pattern

### Decision

Update `ExportProgress` struct via channel after each batch completes:

```go
func Run(db *sql.DB, job *ExportJob, progressChan chan<- ExportProgress) error {
    totalPosts := countPosts(db, job.Options.DID, job.Options.DateRange)
    job.Progress.PostsTotal = totalPosts

    offset := 0
    batchSize := 1000

    for offset < totalPosts {
        batch := fetchBatch(db, offset, batchSize)
        writeBatchToFile(batch) // JSON or CSV

        offset += len(batch)
        job.Progress.PostsProcessed = offset

        // Send progress update
        if progressChan != nil {
            progressChan <- job.Progress
        }

        if len(batch) < batchSize { break } // Last batch
    }
}
```

### Rationale

- **Existing pattern**: Code already uses `progressChan chan<- ExportProgress` for media copying (line 180-191 in handlers/export.go)
- **Non-blocking**: Channel updates don't block export goroutine
- **UI integration**: Web UI already consumes progress via HTMX polling (export.go:238-265)
- **Frequency**: Updates every batch (1,000 posts) ≈ every 0.5-1 seconds, satisfying "every 5 seconds" requirement (SC-004)

### Alternatives Considered

| Alternative | Rejection Reason |
|-------------|-----------------|
| Polling database for progress | Requires storing progress in DB; adds I/O overhead |
| Time-based updates (ticker) | Misses final batch update; channel is simpler |
| Callback function | Channel is more idiomatic Go; already implemented |

---

## 6. Memory Management & GC

### Decision

Rely on Go's garbage collector with explicit nil assignment for batch slices:

```go
for offset < totalPosts {
    batch := fetchBatch(db, offset, batchSize) // Allocates new slice
    processBatch(batch)

    batch = nil // Hint to GC that slice is no longer needed
    offset += batchSize
}
```

### Rationale

- **Automatic GC**: Go 1.21+ GC is efficient; no manual memory management needed
- **Slice lifecycle**: Each batch slice goes out of scope after iteration; eligible for GC
- **No pooling needed**: 1,000-post batches are small enough that sync.Pool overhead unnecessary
- **Monitoring**: Can add `runtime.GC()` calls in tests to verify memory behavior

### Alternatives Considered

| Alternative | Rejection Reason |
|-------------|-----------------|
| Manual `runtime.GC()` calls | Unnecessary; adds latency; Go GC is sufficient |
| `sync.Pool` for batch slices | Over-engineering; batch creation cost is negligible |
| Preallocate single large slice | Defeats purpose of batching; loads all into memory |

---

## 7. Error Handling & Recovery

### Decision

Maintain existing cleanup logic with mid-batch failure detection:

```go
func Run(db *sql.DB, job *ExportJob, progressChan chan<- ExportProgress) error {
    outputFile := createOutputFile()
    defer func() {
        if job.Progress.Status == ExportStatusFailed {
            os.Remove(outputFile) // Clean up partial file
        }
    }()

    for offset < totalPosts {
        batch, err := fetchBatch(db, offset, batchSize)
        if err != nil {
            job.Progress.Status = ExportStatusFailed
            job.Progress.Error = err.Error()
            return err
        }

        if err := writeBatch(batch); err != nil {
            job.Progress.Status = ExportStatusFailed
            return err
        }

        // Update progress...
    }

    job.Progress.Status = ExportStatusCompleted
    return nil
}
```

### Rationale

- **Existing pattern**: Code already does cleanup on failure (exporter.go:56-68)
- **Partial file removal**: Failed exports cleaned up automatically
- **Error propagation**: Database and I/O errors returned to caller
- **Status tracking**: `ExportStatus` enum tracks queued/running/completed/failed states

---

## Implementation Summary

### Key Architectural Decisions

1. **Streaming I/O**: Manual JSON array construction + csv.Writer with Flush()
2. **Database pagination**: LIMIT/OFFSET with ORDER BY (deterministic, uses existing index)
3. **Batch size**: Fixed 1,000 posts (per spec)
4. **Progress reporting**: Existing channel pattern (update after each batch)
5. **Memory management**: Rely on Go GC (no manual pooling)
6. **Error handling**: Existing cleanup pattern (defer + status tracking)

### Files to Modify

| File | Changes | Complexity |
|------|---------|-----------|
| `internal/storage/posts.go` | Add LIMIT/OFFSET params to `ListPostsWithDateRange()` | Low |
| `internal/exporter/json.go` | Refactor `ExportToJSON()` for streaming array write | Medium |
| `internal/exporter/csv.go` | Refactor `ExportToCSV()` for batch flushing | Low |
| `internal/exporter/exporter.go` | Replace single query with batch loop | Medium |

### No New Dependencies

All implementation uses existing stdlib imports:
- `database/sql` (queries)
- `encoding/json` (JSON encoding)
- `encoding/csv` (CSV writing)
- `io`, `os` (file I/O)

---

## Open Questions: NONE

All Technical Context items resolved. No "NEEDS CLARIFICATION" remaining.

---

## References

- Existing codebase investigation (internal/exporter/, internal/storage/)
- Go 1.21 stdlib documentation (encoding/json, encoding/csv, database/sql)
- SQLite LIMIT/OFFSET documentation
- Feature spec: [spec.md](./spec.md)
- Constitution: [constitution.md](../../.specify/memory/constitution.md)
