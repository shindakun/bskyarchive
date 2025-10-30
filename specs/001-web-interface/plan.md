# Implementation Plan: Web Interface

**Branch**: `001-web-interface` | **Date**: 2025-10-30 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/001-web-interface/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Create a complete locally hosted Bluesky Personal Archive Tool that combines web interface, data collection, and storage in a single application. Users authenticate via OAuth (bskyoauth), and the tool fetches their posts, profiles, and media directly from Bluesky using the AT Protocol (indigo SDK). All data is stored locally in SQLite with file-based media storage. The web interface features a modern, responsive dark theme using Go for the backend, HTML templates, vanilla JavaScript, and HTMX for dynamic interactions. Users can initiate archive syncs, monitor progress in real-time, and browse their archived content through an intuitive web UI.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**:
- github.com/shindakun/bskyoauth (OAuth authentication)
- github.com/bluesky-social/indigo (AT Protocol SDK for Go)
- net/http (standard library HTTP server)
- html/template (server-side HTML rendering)
- database/sql + modernc.org/sqlite (SQLite for archive storage and indexing)
- HTMX (client-side dynamic interactions)
- Vanilla JavaScript (progressive enhancement)
- Pico CSS or similar minimal CSS framework for dark theme

**Storage**:
- Session data: Encrypted cookies or server-side session store
- Archive data: SQLite database for posts, profiles, media metadata
- Media files: File-based storage (organized by year/month)
- Full-text search: SQLite FTS5 for fast content search

**Testing**: Go testing (testing package), table-driven tests for handlers, integration tests for OAuth flow

**Target Platform**: Cross-platform (macOS, Linux, Windows) - locally hosted web server on localhost

**Project Type**: Web application (server-rendered HTML with progressive enhancement)

**Performance Goals**:
- Page load time <2 seconds
- OAuth flow completion <30 seconds
- Real-time progress updates <1 second latency
- Support single-user concurrent requests efficiently

**Constraints**:
- Localhost-only (no public internet exposure required)
- Single-user instance per installation
- Session expiration: 7 days of inactivity
- Must implement AT Protocol communication for data collection
- Must handle Bluesky API rate limits (300 requests per 5 minutes)
- WCAG AA accessibility compliance for dark theme

**Scale/Scope**:
- Single user per instance
- Support browsing 10,000+ archived posts with pagination
- 5 primary pages: Landing, Dashboard, Archive Management, Archive Browse, About
- Minimal JavaScript footprint (<50KB total)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Data Privacy & Local-First Architecture ✅
- **Compliant**: Web interface runs on localhost only, no external data transmission
- **Compliant**: OAuth tokens stored securely (encrypted cookies/session store)
- **Compliant**: All archive data remains local (file + SQLite storage)
- **Compliant**: No telemetry or external analytics

### II. Comprehensive & Accurate Archival ✅
- **Compliant**: Web UI displays existing archive data without modification
- **Compliant**: Initiates archival operations via existing backend services
- **Compliant**: Real-time progress tracking for archival operations
- **Not Applicable**: UI layer does not handle data collection directly

### III. Multiple Export Formats ✅
- **Compliant**: Web interface provides access to browse archived content
- **Future**: Export format generation handled by existing backend (not in this feature scope)

### IV. Fast & Efficient Search ✅
- **Compliant**: Archive browse interface supports pagination for 10,000+ posts
- **Compliant**: Page load targets <2 seconds
- **Future**: Advanced search functionality (filtering, full-text) will integrate with existing search backend

### V. Incremental & Efficient Operations ✅
- **Compliant**: Supports triggering both full and incremental sync operations
- **Compliant**: Progress updates with <1 second latency
- **Compliant**: Efficient server-side rendering with minimal client-side JavaScript

### Security & Privacy ✅
- **Compliant**: OAuth 2.0 via bskyoauth library
- **Compliant**: Secure session management (7-day expiration, HTTP-only cookies)
- **Compliant**: CSRF protection on all state-changing operations required
- **Compliant**: No plaintext credential storage

### Development Standards ✅
- **Compliant**: Go 1.21+ with net/http standard library
- **Compliant**: Clear separation: handlers, templates, middleware, services
- **Compliant**: HTML templates + HTMX + vanilla JavaScript (as specified in constitution)
- **Compliant**: Testing via Go testing package

**Result**: ✅ All gates pass. No constitution violations. Proceed to Phase 0.

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
│   ├── handlers/           # HTTP request handlers
│   │   ├── auth.go        # OAuth login/logout/callback
│   │   ├── dashboard.go   # Dashboard page
│   │   ├── archive.go     # Archive management
│   │   ├── browse.go      # Archive browsing
│   │   └── about.go       # About page
│   ├── middleware/         # HTTP middleware
│   │   ├── auth.go        # Authentication middleware
│   │   ├── session.go     # Session management
│   │   └── csrf.go        # CSRF protection
│   ├── templates/          # HTML templates
│   │   ├── layouts/       # Base layouts
│   │   │   └── base.html
│   │   ├── pages/         # Page templates
│   │   │   ├── landing.html
│   │   │   ├── dashboard.html
│   │   │   ├── archive.html
│   │   │   ├── browse.html
│   │   │   └── about.html
│   │   └── partials/      # Reusable components
│   │       ├── nav.html
│   │       └── footer.html
│   ├── static/             # Static assets
│   │   ├── css/
│   │   │   └── styles.css # Custom dark theme
│   │   ├── js/
│   │   │   ├── htmx.min.js
│   │   │   └── app.js     # Vanilla JS enhancements
│   │   └── images/
│   └── server.go           # HTTP server setup
├── auth/                   # Authentication service
│   ├── oauth.go           # bskyoauth integration
│   └── session.go         # Session store
├── archiver/               # Bluesky data collection
│   ├── client.go          # AT Protocol client wrapper
│   ├── collector.go       # Post/profile/media collection
│   ├── worker.go          # Background sync worker
│   └── ratelimit.go       # Rate limit handling
├── storage/                # Data persistence
│   ├── db.go              # SQLite database setup
│   ├── posts.go           # Post storage operations
│   ├── profiles.go        # Profile storage operations
│   ├── media.go           # Media download & storage
│   ├── search.go          # Full-text search (FTS5)
│   └── migrations/        # Database schema migrations
│       └── 001_initial.sql
└── models/                 # Data models
    ├── post.go            # Post structures
    ├── profile.go         # Profile structures
    ├── session.go         # Session structures
    └── operation.go       # Archive operation tracking

cmd/
└── bskyarchive/
    └── main.go            # Application entry point

tests/
├── integration/
│   ├── oauth_test.go      # OAuth flow tests
│   └── archiver_test.go   # AT Protocol integration tests
└── unit/
    ├── handlers_test.go   # Handler unit tests
    ├── collector_test.go  # Collection logic tests
    └── storage_test.go    # Storage layer tests
```

**Structure Decision**: This is a complete web application that combines web interface, AT Protocol data collection, and local storage. The structure follows Go conventions with:
- `internal/web`: Web interface (handlers, middleware, templates, static assets)
- `internal/archiver`: Bluesky data collection via AT Protocol
- `internal/storage`: SQLite persistence layer with migrations
- `internal/models`: Shared data structures
- `internal/auth`: OAuth and session management
- `cmd/bskyarchive`: Single binary entry point

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitution violations. This section is not applicable.
