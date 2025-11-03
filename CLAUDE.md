# bskyarchive Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-10-30

## Active Technologies
- SQLite with FTS5 full-text search (001-web-interface)
- Go 1.21+ (existing project standard) + Go stdlib only (encoding/csv, encoding/json, io, os, path/filepath, time) (002-archive-export)
- SQLite (existing - no changes needed), local filesystem for exports (002-archive-export)
- Go 1.21+ + Go stdlib only (database/sql, encoding/csv, encoding/json, io, os, path/filepath, time) + modernc.org/sqlite (existing) (003-large-export-batching)
- SQLite with FTS5 full-text search (existing); local filesystem for export files (003-large-export-batching)
- Go 1.25.3 (existing project standard) (004-security-hardening)
- Go 1.21+ (existing project standard) + Go standard library (html/template, net/http), Pico CSS (existing), bskyoauth (existing OAuth library) (006-login-template-styling)
- N/A (no data storage changes, UI-only refactoring) (006-login-template-styling)

- Go 1.21+ (001-web-interface)

## Project Structure

```text
src/
tests/
```

## Commands

# Add commands for Go 1.21+

## Code Style

Go 1.21+: Follow standard conventions

## Recent Changes
- 006-login-template-styling: Added Go 1.21+ (existing project standard) + Go standard library (html/template, net/http), Pico CSS (existing), bskyoauth (existing OAuth library)
- 005-export-download: Added Go 1.21+ (existing project standard)
- 004-security-hardening: Added Go 1.25.3 (existing project standard)

<!-- MANUAL ADDITIONS START -->
Kill Go process when finished with work.
Do not kill ngrok processes!
<!-- MANUAL ADDITIONS END -->
