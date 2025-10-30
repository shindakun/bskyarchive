# Implementation Plan: Web Interface

**Branch**: `001-web-interface` | **Date**: 2025-10-30 | **Spec**: [spec.md](spec.md)
**Input**: Feature specification from `specs/001-web-interface/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/commands/plan.md` for the execution workflow.

## Summary

Create a locally hosted web interface for the Bluesky Personal Archive Tool that provides OAuth authentication, archive management, and content browsing. The interface will feature a modern, responsive dark theme using Go for the backend, HTML templates, vanilla JavaScript, and HTMX for dynamic interactions. Users will authenticate via bskyoauth, manage archival operations, and browse their archived content through an intuitive web UI.

## Technical Context

**Language/Version**: Go 1.21+
**Primary Dependencies**:
- github.com/shindakun/bskyoauth (OAuth authentication)
- net/http (standard library HTTP server)
- html/template (server-side HTML rendering)
- HTMX (client-side dynamic interactions)
- Vanilla JavaScript (progressive enhancement)
- Pico CSS or similar minimal CSS framework for dark theme

**Storage**:
- Session data: Encrypted cookies or server-side session store
- Archive data: File-based storage (JSON) and SQLite for indexing (from existing archival system)

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
- Must integrate with existing archival backend services
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
└── archive/               # Archive service interface
    └── client.go          # Client for existing archive backend

cmd/
└── web/
    └── main.go            # Web server entry point

tests/
├── integration/
│   └── oauth_test.go     # OAuth flow integration tests
└── unit/
    └── handlers_test.go  # Handler unit tests
```

**Structure Decision**: This is a web application adding a web interface layer to the existing Bluesky archive tool. The structure follows Go conventions with `internal/web` containing all web-specific code (handlers, middleware, templates, static assets), `cmd/web` for the server entry point, and integrations with existing `internal/auth` and `internal/archive` services.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

No constitution violations. This section is not applicable.
