# Implementation Plan: Export Download & Management

**Branch**: `005-export-download` | **Date**: 2025-11-01 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `/specs/005-export-download/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Enable users to download completed exports as ZIP archives and optionally delete them from the server to manage disk space. Exports are streamed to handle large archives efficiently without memory exhaustion. Users can view all their completed exports with metadata, download them individually, and delete them with confirmation. An optional "Delete after download" feature streamlines the workflow.

## Technical Context

**Language/Version**: Go 1.21+ (existing project standard)
**Primary Dependencies**:
  - Go stdlib: archive/zip, io, os, path/filepath, net/http
  - github.com/go-chi/chi/v5 (existing - HTTP router)
  - modernc.org/sqlite (existing - database)

**Storage**:
  - Local filesystem for export files (existing in ./exports directory)
  - SQLite for export metadata tracking (need to add exports table)
  - Export directory structure: ./exports/{did}/{timestamp}/

**Testing**:
  - Go testing (go test)
  - Unit tests for ZIP streaming, export listing, deletion logic
  - Integration tests for download/delete workflows
  - Memory profiling for large file streaming

**Target Platform**: Linux/macOS/Windows server (Go cross-platform)

**Project Type**: Single web application (existing architecture)

**Performance Goals**:
  - Stream ZIP creation without loading full export into memory
  - Handle 5GB+ exports with <500MB memory footprint
  - Download initiation <1 second for typical archives
  - Export list query <1 second for 50+ exports

**Constraints**:
  - Memory-efficient streaming (no buffering entire ZIP in RAM)
  - Concurrent download limits (max 10 per user)
  - Must preserve existing export format and structure
  - Must maintain security isolation between users (DID-based ownership)

**Scale/Scope**:
  - Support users with 10-50 archived exports
  - Handle individual exports up to 10GB
  - Serve 10-50 concurrent downloads across all users

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Data Privacy & Local-First Architecture ✅

- **All user data MUST remain under user control**: ✅ Exports remain on local server; users can download and delete
- **No third-party services**: ✅ ZIP streaming uses Go stdlib, no external services
- **User controls all data export, deletion, and sharing**: ✅ This feature explicitly enables user control over exports
- **No telemetry**: ✅ Only audit logging for security (local only)

**Status**: PASS - This feature enhances user control over their data.

### II. Comprehensive & Accurate Archival ✅

- **Preserve complete history**: ✅ Feature does not modify archival process
- **Maintain data integrity**: ✅ ZIP archives preserve all files and directory structure
- **Export maintains relationships**: ✅ Existing manifest.json structure preserved

**Status**: PASS - Download feature preserves existing export integrity.

### III. Multiple Export Formats ✅

- **Support JSON, CSV, HTML formats**: ✅ Feature downloads whatever format user exported
- **Maintain data integrity**: ✅ ZIP contains all original export files unchanged

**Status**: PASS - Format support unchanged, just adds download capability.

### IV. Fast & Efficient Search ✅

- **N/A**: This feature does not involve search functionality

**Status**: N/A

### V. Incremental & Efficient Operations ✅

- **Respect resources and scale efficiently**: ✅ Streaming ZIP prevents memory exhaustion
- **Handle large archives efficiently**: ✅ Designed for 10GB+ exports with <500MB memory
- **Support 10,000+ posts**: ✅ Memory-efficient streaming handles any export size

**Status**: PASS - Streaming architecture respects resource constraints.

### Security & Privacy ✅

- **Secure session management**: ✅ Downloads require authenticated session
- **Verify ownership before operations**: ✅ DID-based authorization on download/delete
- **CSRF protection**: ✅ Delete operations protected by CSRF tokens
- **Rate limiting**: ✅ Max 10 concurrent downloads per user
- **No credential storage**: ✅ Feature does not touch credentials

**Status**: PASS - Comprehensive security measures included.

### Development Standards ✅

- **Go 1.21+ with standard library**: ✅ Uses archive/zip, net/http stdlib
- **Clear separation of concerns**: ✅ Handlers, storage layer, business logic separated
- **Comprehensive error handling**: ✅ Graceful handling of missing files, disk errors
- **HTML, Picocss, HTMX**: ✅ UI follows existing patterns

**Status**: PASS - Adheres to all development standards.

### Overall Constitution Compliance: ✅ PASS

All principles satisfied. No violations requiring justification. Ready to proceed with Phase 0.

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
├── web/
│   ├── handlers/
│   │   ├── export.go         # Update: Add download/delete handlers
│   │   └── export_test.go    # Add: Tests for new handlers
│   └── templates/
│       └── pages/
│           └── export.html    # Update: Add export list, download/delete UI
│
├── models/
│   └── export.go              # Update: Add ExportMetadata, ExportRecord types
│
├── storage/
│   ├── exports.go             # Add: Export CRUD operations
│   ├── exports_test.go        # Add: Tests for export storage
│   └── migrations/            # Add: Migration for exports table
│
└── exporter/
    ├── download.go            # Add: ZIP streaming logic
    ├── download_test.go       # Add: Tests for ZIP streaming
    └── exporter.go            # Update: Track exports in DB after creation

tests/
├── integration/
│   ├── export_download_test.go  # Add: E2E download tests
│   └── export_deletion_test.go  # Add: E2E deletion tests
└── unit/
    ├── zip_streaming_test.go    # Add: Unit tests for ZIP creation
    └── export_storage_test.go   # Add: Unit tests for export queries
```

**Structure Decision**: This is a single-project web application using the existing Go codebase structure. New functionality is integrated into existing packages (handlers, models, storage, exporter) rather than creating new top-level modules. This maintains consistency with the current architecture and follows the separation of concerns already established.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitution violations detected. This table is not needed.

---

## Phase 0: Research (Complete)

All unknowns from Technical Context have been resolved through research. Key decisions:

1. **ZIP Streaming**: Use Go stdlib `archive/zip` with `io.Pipe()` for memory-efficient streaming
2. **Export Tracking**: SQLite table with DID-indexed metadata
3. **Authorization**: DID-based ownership checks with session authentication
4. **Memory Efficiency**: Buffered file streaming via `io.Copy()` - handles 10GB+ exports with <500MB memory
5. **Deletion**: Simple `os.RemoveAll()` + database cleanup with audit logging
6. **UI Pattern**: Pico.css table + HTMX for consistency with existing app
7. **Rate Limiting**: In-memory concurrent download tracking (max 10 per user)
8. **Delete After Download**: Query parameter with server-side cleanup (safe approach)

See [research.md](research.md) for detailed analysis and alternatives considered.

---

## Phase 1: Design & Contracts (Complete)

### Data Model

Created comprehensive data model defining:
- **ExportRecord entity** with validation rules
- **DownloadSession** for rate limiting (in-memory)
- **Relationships**: User→Exports (one-to-many), Export→Filesystem (one-to-one)
- **Database schema** with indexes for performance
- **Migration script** for `exports` table

See [data-model.md](data-model.md) for full entity definitions.

### API Contracts

Defined 4 HTTP endpoints:
- `GET /export` - Export management page (HTML)
- `GET /export/list` - List user's exports (JSON)
- `GET /export/download/{export_id}` - Download as ZIP (streaming)
- `DELETE /export/delete/{export_id}` - Delete export (with CSRF)

All endpoints include:
- Authentication requirements
- Authorization checks (DID ownership)
- Error response formats
- Security considerations
- Example requests/responses

See [contracts/http-api.md](contracts/http-api.md) for complete API specification.

### Quickstart Guide

Step-by-step implementation guide covering:
1. Database migration (create exports table)
2. Model updates (ExportRecord type)
3. Storage layer (CRUD operations)
4. ZIP streaming implementation
5. Export tracking (update exporter.go)
6. HTTP handlers (download, delete, list)
7. Route registration
8. UI template updates
9. Testing procedures
10. Deployment checklist

See [quickstart.md](quickstart.md) for detailed implementation steps.

### Agent Context Update

Updated CLAUDE.md with new technology choices:
- Added: Go 1.21+ (existing project standard)
- Project type: Single web application

---

## Phase 2: Constitution Re-Check

Re-evaluated all design artifacts against constitution principles:

### ✅ Data Privacy & Local-First Architecture
- Exports remain on local server under user control
- No third-party services for ZIP creation or storage
- Only audit logging (local only)

### ✅ Comprehensive & Accurate Archival
- Feature does not modify archival process
- ZIP archives preserve complete directory structure
- Manifest files maintained for integrity verification

### ✅ Multiple Export Formats
- Feature downloads any format user created (JSON/CSV)
- No format restrictions or conversions

### ✅ Fast & Efficient Search
- N/A - Feature does not involve search

### ✅ Incremental & Efficient Operations
- Streaming ZIP prevents memory exhaustion (handles 10GB+ with <500MB memory)
- Rate limiting prevents resource abuse
- Efficient database queries with proper indexes

### ✅ Security & Privacy
- Session-based authentication required
- DID-based ownership verification on all operations
- CSRF protection on state-changing operations (DELETE)
- Rate limiting (10 concurrent downloads per user)
- Audit logging for security monitoring
- Path validation prevents traversal attacks

### ✅ Development Standards
- Go 1.21+ with stdlib only (archive/zip, io, net/http)
- Clear separation: handlers, storage, exporter
- Comprehensive error handling
- HTML templates with Pico.css + HTMX (consistent with existing)

**Result**: All principles satisfied. No violations. Design approved.

---

## Implementation Readiness

✅ **Technical Context**: Complete and validated
✅ **Constitution Check**: All principles satisfied
✅ **Research**: All unknowns resolved
✅ **Data Model**: Entities, relationships, validation defined
✅ **API Contracts**: 4 endpoints fully specified
✅ **Quickstart**: Step-by-step implementation guide ready
✅ **Agent Context**: Updated for Go 1.21+

**Status**: Ready for Phase 3 (Implementation via `/speckit.tasks`)

---

## Summary

This implementation plan delivers export download and management functionality that:
- Enables users to download completed exports as ZIP archives
- Provides optional "delete after download" for streamlined workflow
- Implements secure, memory-efficient streaming for large files (10GB+)
- Maintains comprehensive security (authentication, authorization, rate limiting)
- Preserves existing export structure and formats
- Follows all constitutional principles
- Integrates seamlessly with existing architecture

No additional research or design decisions needed. Implementation can proceed directly from quickstart guide and API contracts.
