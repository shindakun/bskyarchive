# Quickstart: Large Export Batching Implementation

**Feature**: 003-large-export-batching
**Branch**: `003-large-export-batching`
**Target**: Enable exports of 100,000+ posts without memory exhaustion

## Prerequisites

- Go 1.21+ installed
- Existing bskyarchive codebase on branch `003-large-export-batching`
- SQLite database with test data (optional: use existing archive)

---

## Implementation Checklist

### Phase 1: Storage Layer Pagination (2-3 hours)

**File**: `internal/storage/posts.go`

- [ ] Modify `ListPostsWithDateRange()` to add deterministic ordering:
  - Change `ORDER BY created_at DESC` to `ORDER BY created_at DESC, uri ASC`
  - Ensure LIMIT/OFFSET logic handles `limit=0` correctly (backward compatibility)
- [ ] Add unit tests in `internal/storage/posts_test.go`:
  - Test pagination with offset=0, 1000, 2000, etc.
  - Test last partial batch (e.g., 2,500 posts → 3 batches)
  - Test empty result when offset > total count
  - Verify ORDER BY determinism (same query = same results)

**Verification**:
```bash
go test ./internal/storage -run TestListPostsWithDateRange_Pagination -v
```

---

### Phase 2: JSON Streaming Export (3-4 hours)

**File**: `internal/exporter/json.go`

- [ ] Create new function `exportToJSONStreamingWriter()`:
  ```go
  func exportToJSONStreamingWriter(w io.Writer,
                                    posts []models.Post,
                                    isFirst, isLast bool) error
  ```
  - Write opening `[` if `isFirst=true`
  - Write posts with json.Encoder (pretty-print, 2-space indent)
  - Add commas between objects (not after last post in batch)
  - Write closing `]` if `isLast=true`

- [ ] Refactor `ExportToJSON()` to call streaming writer in loop:
  ```go
  f, _ := os.Create(outputPath)
  defer f.Close()

  for offset := 0; offset < totalPosts; offset += 1000 {
      batch := fetchBatch(offset)
      isFirst := (offset == 0)
      isLast := (offset + len(batch) >= totalPosts)
      exportToJSONStreamingWriter(f, batch, isFirst, isLast)
  }
  ```

- [ ] Add unit tests in `internal/exporter/json_test.go`:
  - Test single batch (< 1,000 posts)
  - Test multiple batches (2,500 posts → 3 batches)
  - **Critical**: Test byte-identical output vs. original implementation
  - Test valid JSON array structure (parse with `json.Unmarshal`)

**Verification**:
```bash
go test ./internal/exporter -run TestExportToJSON -v

# Byte-identical test
diff original_export.json batched_export.json
# (Should show no differences)
```

---

### Phase 3: CSV Streaming Export (2-3 hours)

**File**: `internal/exporter/csv.go`

- [ ] Refactor `ExportToCSV()` for batched writes:
  ```go
  f, _ := os.Create(outputPath)
  defer f.Close()

  f.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM

  w := csv.NewWriter(f)
  w.Write(headerRow) // Write once at start

  for offset := 0; offset < totalPosts; offset += 1000 {
      batch := fetchBatch(offset)
      for _, post := range batch {
          w.Write(postToCSVRow(post))
      }
      w.Flush() // Force write after each batch
  }

  return w.Error()
  ```

- [ ] Add unit tests in `internal/exporter/csv_test.go`:
  - Test multiple batches (5,000 posts)
  - **Critical**: Test byte-identical output vs. original
  - Test RFC 4180 compliance (proper escaping)
  - Test UTF-8 BOM present

**Verification**:
```bash
go test ./internal/exporter -run TestExportToCSV -v

# Byte-identical test
diff original_export.csv batched_export.csv
```

---

### Phase 4: Main Exporter Loop (3-4 hours)

**File**: `internal/exporter/exporter.go`

- [ ] Refactor `Run()` function:
  - Replace single `storage.ListPostsWithDateRange(db, did, dateRange, 0, 0)` call
  - Add COUNT query to determine `totalPosts` (set `job.Progress.PostsTotal`)
  - Add batching loop:
    ```go
    const batchSize = 1000
    offset := 0

    for offset < job.Progress.PostsTotal {
        batch, err := storage.ListPostsWithDateRange(db, did, dateRange, batchSize, offset)
        if err != nil { return err }
        if len(batch) == 0 { break }

        // Write batch to file (call JSON or CSV streaming writer)

        offset += len(batch)
        job.Progress.PostsProcessed = offset

        // Send progress update
        if progressChan != nil {
            progressChan <- job.Progress
        }
    }
    ```

- [ ] Update `ExportToJSON()` and `ExportToCSV()` calls to use streaming variants
- [ ] Preserve existing error handling and cleanup logic (defer)

**Verification**:
```bash
# Manual test with real database
./bskyarchive export --format json --output /tmp/test_export
# Check logs for progress updates
```

---

### Phase 5: Integration Tests (2-3 hours)

**File**: `tests/integration/export_batching_test.go` (NEW)

- [ ] Create test database fixture with 10,000 posts:
  ```go
  func setupLargeTestDB(t *testing.T, postCount int) *sql.DB {
      db := setupTestDB(t)
      for i := 0; i < postCount; i++ {
          insertTestPost(db, generatePost(i))
      }
      return db
  }
  ```

- [ ] Test scenarios:
  - [ ] Export 10,000 posts (JSON): verify all posts present
  - [ ] Export 10,000 posts (CSV): verify row count matches
  - [ ] Progress updates: verify 10 updates sent (every 1,000 posts)
  - [ ] Memory usage: verify < 100MB during export (use runtime profiling)
  - [ ] Small archive (500 posts): verify performance same as baseline

**Verification**:
```bash
go test ./tests/integration -run TestExportBatching -v
```

---

### Phase 6: Performance Verification (1-2 hours)

- [ ] Create large test database (50,000 posts):
  ```bash
  go run scripts/generate_test_data.go --posts 50000 --output test_large.db
  ```

- [ ] Run benchmark tests:
  ```go
  func BenchmarkExport_50kPosts(b *testing.B) {
      db := setupLargeTestDB(b, 50000)
      for i := 0; i < b.N; i++ {
          job := &models.ExportJob{...}
          exporter.Run(db, job, nil)
      }
  }
  ```

- [ ] Profile memory usage:
  ```bash
  go test -bench=BenchmarkExport_50kPosts -memprofile=mem.prof ./internal/exporter
  go tool pprof -http=:8080 mem.prof
  # Verify peak memory < 500MB
  ```

- [ ] Verify success criteria (SC-001 to SC-007):
  - [ ] SC-002: Memory < 500MB ✓
  - [ ] SC-003: Export speed 1,500-2,000 posts/sec ✓
  - [ ] SC-004: Progress updates every 5 seconds ✓

---

## Development Workflow

### 1. Setup Development Environment

```bash
cd /Users/steve/go/src/github.com/shindakun/bskyarchive
git checkout 003-large-export-batching
go mod download
```

### 2. Run Tests Continuously

```bash
# Watch mode (install first: go install github.com/cespare/reflex@latest)
reflex -r '\.go$' -s -- go test ./internal/...
```

### 3. Manual Testing

```bash
# Build binary
go build -o bskyarchive ./cmd/bskyarchive

# Test with real archive
./bskyarchive export --format json --output /tmp/export_test

# Check output
ls -lh /tmp/export_test/posts.json
jq length /tmp/export_test/posts.json  # Count posts
```

### 4. Memory Profiling

```bash
# Add profiling flag to CLI (temporary)
go run ./cmd/bskyarchive export --format json --memprofile=mem.prof

# Analyze profile
go tool pprof -http=:8080 mem.prof
```

---

## Key Implementation Patterns

### Pattern 1: Deterministic Pagination

```go
// Always use ORDER BY with tie-breaker (uri is primary key)
query := `
    SELECT * FROM posts
    WHERE did = ? AND created_at BETWEEN ? AND ?
    ORDER BY created_at DESC, uri ASC
    LIMIT ? OFFSET ?
`
```

**Why**: Ensures same results across multiple queries (no skips/duplicates)

---

### Pattern 2: Streaming JSON Array

```go
// Opening
fmt.Fprintf(w, "[\n")

// Each batch
for i, post := range batch {
    if !isFirstPostOverall { fmt.Fprintf(w, ",\n") }
    enc := json.NewEncoder(w)
    enc.SetIndent("", "  ")
    enc.Encode(post)
}

// Closing
fmt.Fprintf(w, "\n]")
```

**Why**: Produces valid JSON array while streaming batches

---

### Pattern 3: Progress Reporting

```go
// After each batch
offset += len(batch)
job.Progress.PostsProcessed = offset

if progressChan != nil {
    select {
    case progressChan <- job.Progress:
    default: // Don't block if channel full
    }
}
```

**Why**: Non-blocking updates, UI stays responsive

---

## Common Pitfalls

### ❌ Pitfall 1: JSON Comma Handling

```go
// WRONG: Missing commas between batches
for offset := 0; offset < total; offset += 1000 {
    batch := fetchBatch(offset)
    json.NewEncoder(w).Encode(batch) // Each batch is separate array!
}
```

**Fix**: Track first post globally, add comma before each post (except first)

---

### ❌ Pitfall 2: Non-Deterministic Ordering

```go
// WRONG: Can produce duplicates/skips across batches
ORDER BY created_at DESC  // Multiple posts with same timestamp!
```

**Fix**: Add tie-breaker: `ORDER BY created_at DESC, uri ASC`

---

### ❌ Pitfall 3: Last Batch Logic

```go
// WRONG: Infinite loop if total posts is multiple of 1000
for offset := 0; offset < total; offset += 1000 {
    batch := fetchBatch(offset)
    // Doesn't check if batch is empty!
}
```

**Fix**: Add `if len(batch) == 0 { break }` or check `len(batch) < batchSize`

---

## Testing Strategy

### Unit Tests (Fast)

Focus: Individual functions in isolation
- Storage pagination logic
- JSON streaming writer
- CSV streaming writer
- Progress update logic

```bash
go test ./internal/... -short
```

---

### Integration Tests (Medium)

Focus: End-to-end export with real database
- Export 10,000 posts (JSON/CSV)
- Verify progress updates sent
- Verify output correctness

```bash
go test ./tests/integration/...
```

---

### Performance Tests (Slow)

Focus: Success criteria verification
- Memory usage < 500MB
- Export speed 1,500-2,000 posts/sec
- Small archive performance (regression)

```bash
go test -bench=. -benchmem ./internal/exporter
```

---

## Debugging Tips

### View SQL Queries

```go
// Add before db.Query() calls
log.Printf("SQL: %s | Params: %v", query, []interface{}{did, start, end, limit, offset})
```

### Track Memory Allocations

```go
import "runtime"

var m runtime.MemStats
runtime.ReadMemStats(&m)
log.Printf("Alloc: %d MB, TotalAlloc: %d MB", m.Alloc/1024/1024, m.TotalAlloc/1024/1024)
```

### Verify JSON Validity

```bash
jq . output.json > /dev/null && echo "Valid JSON" || echo "Invalid JSON"
```

---

## Success Criteria Verification

Before marking feature complete, verify:

- [ ] **SC-001**: Export 100,000 posts successfully (integration test)
- [ ] **SC-002**: Memory < 500MB (profiling)
- [ ] **SC-003**: Export speed 1,500-2,000 posts/sec (benchmarks)
- [ ] **SC-004**: Progress updates every 5 seconds (channel message count)
- [ ] **SC-005**: 99% success rate for 50k+ exports (stress test)
- [ ] **SC-006**: Small archive performance unchanged (regression test)
- [ ] **SC-007**: Byte-identical output (diff test)

---

## Rollout Plan

### Step 1: Merge to Main

```bash
git checkout 003-large-export-batching
git rebase main
go test ./...
git push origin 003-large-export-batching
# Create PR, request review
```

### Step 2: Release Notes

Document in `CHANGELOG.md`:
- **Fixed**: Memory exhaustion on large archive exports (50,000+ posts)
- **Improved**: Export progress updates now show incremental progress
- **Internal**: Refactored export engine for streaming writes

### Step 3: User Communication

- No user-facing changes (transparent optimization)
- Existing export commands work identically
- No migration required

---

## Next Steps After Implementation

1. **Monitor production usage**: Track memory metrics for large exports
2. **Consider future optimizations**:
   - Cursor-based pagination for >100k posts
   - Parallel batch processing (if I/O bound)
   - Configurable batch size (advanced users)
3. **Documentation updates**: Internal developer docs (this serves as reference)

---

## Reference Files

- **Spec**: [spec.md](./spec.md)
- **Research**: [research.md](./research.md)
- **Data Model**: [data-model.md](./data-model.md)
- **Contracts**: [contracts/internal-api.md](./contracts/internal-api.md)
- **Implementation Plan**: [plan.md](./plan.md)
