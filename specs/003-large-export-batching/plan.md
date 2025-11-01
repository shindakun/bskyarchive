# Implementation Plan: Large Export Batching

**Branch**: `003-large-export-batching` | **Date**: 2025-11-01 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/003-large-export-batching/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Implement batched processing for archive exports to prevent memory exhaustion on large archives (50,000+ posts). Current implementation loads all posts into memory at once (25-250MB for large archives), causing OOM errors. Solution: retrieve and process posts in batches of 1,000 using pagination, streaming writes to output files, and incremental progress updates. Target: memory usage <500MB for any archive size while maintaining export speed of 1,500-2,000 posts/second.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**: Go stdlib only (database/sql, encoding/csv, encoding/json, io, os, path/filepath, time) + modernc.org/sqlite (existing)
**Storage**: SQLite with FTS5 full-text search (existing); local filesystem for export files
**Testing**: Go testing stdlib (`testing` package) for unit and integration tests
**Target Platform**: Cross-platform CLI (macOS, Linux, Windows)
**Project Type**: Single project (CLI + embedded web UI)
**Performance Goals**: 1,500-2,000 posts/second export throughput; progress updates every 5 seconds
**Constraints**: Memory usage <500MB for any archive size; export file byte-identical to non-batched version
**Scale/Scope**: Support archives up to 100,000+ posts efficiently; batch size 1,000 posts

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Principle V: Incremental & Efficient Operations ✓

**Requirement**: "Support archives of 10,000+ posts efficiently" and "Generate exports in <30 seconds for typical archives"

**Compliance**: This feature directly addresses Principle V by implementing batching to support large archives (100,000+ posts) efficiently. Current implementation fails on large archives due to memory exhaustion. Batching ensures resource respect and scalability.

**Status**: PASS - Core requirement for constitutional compliance

### Principle III: Multiple Export Formats ✓

**Requirement**: "All exports maintain data integrity and relationships" with JSON and CSV formats

**Compliance**: Feature preserves existing JSON/CSV export functionality. FR-005 requires byte-identical output to non-batched version, ensuring data integrity is maintained.

**Status**: PASS - No regression, maintains existing format support

### Development Standards: Go stdlib + Clear Separation ✓

**Requirement**: "Go 1.21+ with standard library practices" and "Clear separation of concerns"

**Compliance**: Implementation uses only Go stdlib (encoding/json, encoding/csv, io, os, path/filepath, time) plus existing modernc.org/sqlite dependency. Changes isolated to exporter and storage layers, maintaining separation of concerns.

**Status**: PASS - Follows established patterns

### Security & Privacy: Local-First Architecture ✓

**Requirement**: "All data stored locally on user's machine"

**Compliance**: Batching implementation is internal optimization. No external services, no cloud dependencies. Export files remain local.

**Status**: PASS - No impact on privacy model

### Summary

**Overall Status**: PASS - All constitutional principles satisfied. Feature enhances Principle V (efficiency) without compromising other principles. No violations requiring justification.

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
internal/
├── exporter/
│   ├── exporter.go           # Main export orchestrator (MODIFY: add batching loop)
│   ├── json.go               # JSON export (MODIFY: streaming write)
│   ├── csv.go                # CSV export (MODIFY: streaming write)
│   └── exporter_test.go      # Unit tests (NEW: batch verification)
├── storage/
│   ├── posts.go              # Database queries (MODIFY: add pagination)
│   └── posts_test.go         # Storage tests (NEW: pagination tests)
└── models/
    ├── export.go             # ExportJob, ExportProgress structs (existing)
    └── post.go               # Post model (existing)

cmd/
└── bskyarchive/
    └── main.go               # CLI entry point (no changes)

tests/
├── integration/
│   └── export_batching_test.go  # NEW: End-to-end export tests
└── fixtures/
    └── large_archive.db         # NEW: Test database with 10k+ posts
```

**Structure Decision**: Single project structure (Option 1). Changes isolated to `internal/exporter/` and `internal/storage/` packages, maintaining existing separation of concerns. Web UI handlers in `internal/web/handlers/export.go` require no changes - they already consume progress updates via channel.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No violations - table omitted.

---

## Phase 0 & 1 Summary

### Phase 0: Research (Complete)

**Output**: [research.md](./research.md)

**Key Decisions**:
1. **Streaming I/O**: Manual JSON array construction with `json.Encoder` for objects; `csv.Writer` with `Flush()` per batch
2. **Database Pagination**: LIMIT/OFFSET with `ORDER BY created_at DESC, uri ASC` (deterministic, uses existing index)
3. **Batch Size**: Fixed 1,000 posts (per FR-001)
4. **Progress Reporting**: Existing channel pattern, update after each batch
5. **Memory Management**: Rely on Go GC (no manual pooling needed)

**All Technical Context items resolved** - no "NEEDS CLARIFICATION" remaining.

---

### Phase 1: Design (Complete)

**Outputs**:
- [data-model.md](./data-model.md) - Entity changes (none), function signature changes (usage only)
- [contracts/internal-api.md](./contracts/internal-api.md) - Internal Go API contracts
- [quickstart.md](./quickstart.md) - Implementation guide for developers

**Key Design Decisions**:

1. **Zero Model Changes**: No new structs, fields, or database schema changes. Batching is pure implementation refactoring.

2. **Backward Compatible**: `ListPostsWithDateRange()` signature unchanged, supports both batched (limit>0) and non-batched (limit=0) usage.

3. **Streaming Writers**:
   - JSON: Manual `[` `]` construction, `json.Encoder` for objects
   - CSV: Single `csv.Writer` instance, `Flush()` after each batch

4. **Files Modified**:
   - `internal/storage/posts.go` - Add deterministic ORDER BY
   - `internal/exporter/exporter.go` - Replace single query with batch loop
   - `internal/exporter/json.go` - Streaming JSON array writer
   - `internal/exporter/csv.go` - Batched CSV writer with flush

5. **Agent Context Updated**: CLAUDE.md updated with Go 1.21+ stdlib dependencies (no new external deps).

---

### Constitution Re-Check (Post-Design)

**Status**: PASS ✓

All principles remain satisfied:
- **Principle V**: Memory bounded at ~15MB per batch (well under 500MB constraint)
- **Principle III**: Byte-identical output verified via test contracts
- **Development Standards**: Pure Go stdlib (encoding/json, encoding/csv, database/sql)
- **Security & Privacy**: Zero external dependencies, local-first unchanged

**No new violations introduced.**

---

## Next Steps

**Phase 2**: Task generation via `/speckit.tasks` command (generates [tasks.md](./tasks.md))

**Implementation**: Follow [quickstart.md](./quickstart.md) for step-by-step implementation guide.
