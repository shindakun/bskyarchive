# Implementation Plan: Archive Export

**Branch**: `002-archive-export` | **Date**: 2025-10-30 | **Spec**: [spec.md](spec.md)

**Note**: This plan leverages existing Go stdlib functionality (encoding/csv, encoding/json, io, os) with no external dependencies required.

## Summary

Enable users to export their complete Bluesky archive to JSON or CSV formats with associated media files. JSON exports preserve full post metadata and structure for programmatic access or backup. CSV exports provide spreadsheet-compatible format for analysis in Excel/Google Sheets. Optional date range filtering allows targeted exports. All exports include media files in organized /media subdirectories with consistent hash-based filenames matching the archive's content-addressable storage.

**Technical approach**: Leverage Go stdlib (encoding/csv, encoding/json) with existing database/storage patterns. Reuse models.Post structure, storage.ListPosts queries, and file I/O utilities already present in the codebase.

## Technical Context

**Language/Version**: Go 1.21+ (existing project standard)
**Primary Dependencies**: Go stdlib only (encoding/csv, encoding/json, io, os, path/filepath, time)
**Storage**: SQLite (existing - no changes needed), local filesystem for exports
**Testing**: Go testing stdlib (tests/unit/, tests/integration/)
**Target Platform**: macOS/Linux/Windows (localhost web application)
**Project Type**: Single project (web application backend + handlers)
**Performance Goals**: Export 1,000 posts in <10 seconds, support 10,000+ posts without memory issues
**Constraints**: Must handle Unicode/emoji correctly, RFC 4180 CSV compliance, valid JSON output
**Scale/Scope**: Handle archives from 100 to 50,000+ posts with associated media

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### ✅ I. Data Privacy & Local-First Architecture

- **Compliance**: All exports stay local on user's machine
- **Evidence**: Exports written to configurable local `./exports/` directory, no external services
- **No violations**

### ✅ II. Comprehensive & Accurate Archival

- **Compliance**: Exports preserve 100% of archived data with full metadata
- **Evidence**: JSON exports include all Post fields (URI, CID, timestamps, engagement metrics, embed_data, labels). CSV exports include key fields. Media files copied with hash-based filenames.
- **No violations**

### ✅ III. Multiple Export Formats

- **Compliance**: Implements JSON and CSV export formats
- **Evidence**: Directly fulfills constitution requirement for multiple formats. JSON for programmatic access, CSV for spreadsheets.
- **No violations**

### ✅ IV. Fast & Efficient Search

- **Compliance**: Not applicable - export feature doesn't modify search
- **No impact on existing FTS5 search capabilities**

### ✅ V. Incremental & Efficient Operations

- **Compliance**: Export operations respect resource constraints
- **Evidence**:
  - Streaming writes prevent memory bloat for large archives
  - Date range filtering enables selective exports
  - Target: 1,000 posts in <10 seconds
  - Disk space validation before export prevents failures
- **No violations**

### ✅ Security & Privacy

- **Compliance**: No authentication changes, maintains existing security model
- **Evidence**: Exports only accessible to authenticated user via existing RequireAuth middleware
- **No violations**

### ✅ Development Standards

- **Compliance**: Uses Go stdlib, follows existing code patterns
- **Evidence**:
  - No external dependencies (encoding/csv, encoding/json are stdlib)
  - Follows internal/models, internal/storage patterns
  - Clear separation: internal/exporter/ package
  - Standard Go testing approach
- **No violations**

**GATE STATUS**: ✅ PASS - No constitution violations. Feature aligns with principles.

## Project Structure

### Documentation (this feature)

```text
specs/002-archive-export/
├── plan.md              # This file
├── research.md          # Phase 0 - stdlib usage patterns
├── data-model.md        # Phase 1 - ExportJob, ExportOptions models
├── quickstart.md        # Phase 1 - User guide for exports
├── contracts/           # Phase 1 - HTTP endpoints and responses
│   └── export-api.yaml  # OpenAPI spec for export endpoints
└── tasks.md             # Phase 2 - Implementation tasks (/speckit.tasks)
```

### Source Code (repository root)

```text
internal/
├── exporter/                # NEW - Export functionality
│   ├── exporter.go         # Core export logic (JSON/CSV generation)
│   ├── json.go             # JSON export implementation
│   ├── csv.go              # CSV export implementation
│   ├── media.go            # Media file copying
│   └── manifest.go         # Manifest generation
├── models/                  # EXISTING
│   ├── post.go             # Reuse existing Post model
│   └── export.go           # NEW - ExportJob, ExportOptions, ExportManifest
├── storage/                 # EXISTING
│   ├── posts.go            # Reuse ListPosts with date filtering
│   ├── media.go            # Reuse GetMediaForPost
│   └── export.go           # NEW - Export operation tracking (optional)
└── web/                     # EXISTING
    ├── handlers/
    │   ├── handlers.go     # EXISTING
    │   └── export.go       # NEW - HTTP handlers for export UI and API
    └── templates/
        └── pages/
            └── export.html # NEW - Export UI page

exports/                     # NEW - Export output directory (gitignored)
└── {timestamp}/            # Timestamped export directories
    ├── posts.json or posts.csv
    ├── manifest.json
    └── media/              # Copied media files

tests/
├── unit/
│   ├── exporter_test.go    # NEW - Test JSON/CSV generation
│   └── export_media_test.go # NEW - Test media copying
└── integration/
    └── export_integration_test.go # NEW - End-to-end export tests
```

**Structure Decision**: Extends existing single-project structure. New `internal/exporter/` package encapsulates export logic. Reuses existing `internal/storage/` queries and `internal/models/` types. Integrates with existing web handlers and templates following established patterns.

## Complexity Tracking

> **No complexity violations - no table needed**

This feature introduces no new architectural patterns beyond existing practices. Uses stdlib only, follows existing package structure, integrates naturally with current database/storage layer.
