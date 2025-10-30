# Implementation Plan: Web Interface

**Branch**: `001-web-interface` | **Date**: 2025-10-30 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-web-interface/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Create a complete locally hosted Bluesky Personal Archive Tool that combines web interface, data collection, and storage in a single application. Users authenticate via OAuth (bskyoauth), and the tool fetches their posts, profiles, and media directly from Bluesky using the AT Protocol (indigo SDK). The web interface provides a modern, dark-themed landing page, archive management dashboard, and about page. All data is stored locally using SQLite with full-text search capabilities.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**:
- github.com/shindakun/bskyoauth (OAuth authentication)
- github.com/bluesky-social/indigo (AT Protocol SDK for Go)
- net/http (standard library HTTP server)
- database/sql + modernc.org/sqlite (SQLite for archive storage and indexing)
- HTMX (client-side dynamic interactions)
- Pico CSS (classless CSS framework for dark theme)

**Storage**: SQLite with FTS5 full-text search
**Testing**: go test (standard library testing)
**Target Platform**: Local development environment (macOS, Linux, Windows)
**Project Type**: Web application (single Go binary with embedded web interface)
**Performance Goals**:
- Page load < 2 seconds
- Search queries < 100ms
- Archive 100 posts/second with rate limiting

**Constraints**:
- Local-first (no cloud dependencies)
- Bluesky API rate limit (300 requests per 5 minutes)
- Single-user instance per installation
- Session expiration: 7 days

**Scale/Scope**:
- Support archives of 10,000+ posts
- 5 web pages (landing, dashboard, archive, browse, about)
- Background worker for async archival operations

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### Core Principles Alignment

**I. Data Privacy & Local-First Architecture** ✅
- All data stored locally via SQLite
- No cloud dependencies
- OAuth tokens encrypted via secure session management
- User controls all data through local file system

**II. Comprehensive & Accurate Archival** ✅
- Fetches 100% of user posts via AT Protocol
- Preserves media, timestamps, engagement metrics
- Maintains thread context
- Supports incremental sync

**III. Multiple Export Formats** ⚠️ FUTURE
- Phase 1 focuses on web interface and data collection
- Export functionality deferred to future features

**IV. Fast & Efficient Search** ✅
- SQLite FTS5 for <100ms search
- Filter by date, media, engagement
- Web interface for browsing

**V. Incremental & Efficient Operations** ✅
- Incremental backups (only new content)
- Background worker for async operations
- Rate limit handling (300 req/5min)
- Progress tracking in database

### Security & Privacy ✅
- OAuth 2.0 via bskyoauth
- Session management with 7-day expiration
- CSRF protection planned
- HTTP-only encrypted cookies
- Rate limiting

### Development Standards ✅
- Go 1.21+ with stdlib practices
- Clear separation: auth, archiver, storage, web
- Database migrations
- HTML, Pico CSS, HTMX, vanilla JS
- Testing: unit, integration, contract tests

**GATE STATUS**: ✅ PASS (with export formats deferred to future phase)

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
cmd/
└── bskyarchive/          # Main application entry point
    └── main.go

internal/                 # Private application code
├── web/                  # HTTP handlers, middleware, templates
│   ├── handlers/        # HTTP route handlers
│   ├── middleware/      # Auth, CSRF, logging middleware
│   ├── templates/       # HTML templates (Go templates)
│   └── static/          # CSS, JS, images
│       ├── css/         # Pico CSS + custom styles
│       ├── js/          # HTMX + vanilla JavaScript
│       └── images/
├── auth/                 # OAuth and session management
│   ├── oauth.go         # bskyoauth integration
│   └── session.go       # Session storage and validation
├── archiver/             # Bluesky data collection
│   ├── client.go        # AT Protocol client wrapper
│   ├── collector.go     # Post/profile/media collection
│   ├── worker.go        # Background sync worker
│   └── ratelimit.go     # Rate limit handling
├── storage/              # Data persistence
│   ├── db.go            # SQLite database setup
│   ├── posts.go         # Post storage operations
│   ├── profiles.go      # Profile storage operations
│   ├── media.go         # Media download & storage
│   ├── search.go        # Full-text search (FTS5)
│   └── migrations/      # Database schema migrations
│       ├── 001_initial.sql
│       ├── 002_fts.sql
│       └── ...
└── models/               # Data models
    ├── post.go
    ├── profile.go
    ├── session.go
    └── operation.go

archive/                  # Local archive storage (runtime data)
├── media/               # Downloaded images/videos
│   └── YYYY/MM/        # Organized by year/month
└── db/
    └── archive.db       # SQLite database

tests/
├── integration/         # Integration tests with AT Protocol
├── contract/            # HTTP API contract tests
└── unit/                # Unit tests for business logic

go.mod
go.sum
README.md
```

**Structure Decision**: Single Go project with embedded web interface. The application is organized into internal packages for separation of concerns: web layer (handlers, templates, static assets), auth layer (OAuth + sessions), archiver layer (AT Protocol data collection), storage layer (SQLite operations), and shared models. The `cmd/bskyarchive` package serves as the entry point that wires everything together. Runtime data (media files and SQLite database) is stored in the `archive/` directory which is created at first run.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
