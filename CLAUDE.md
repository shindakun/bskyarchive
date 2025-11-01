# bskyarchive Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-10-30

## Active Technologies
- SQLite with FTS5 full-text search (001-web-interface)
- Go 1.21+ (existing project standard) + Go stdlib only (encoding/csv, encoding/json, io, os, path/filepath, time) (002-archive-export)
- SQLite (existing - no changes needed), local filesystem for exports (002-archive-export)
- Go 1.21+ + Go stdlib only (database/sql, encoding/csv, encoding/json, io, os, path/filepath, time) + modernc.org/sqlite (existing) (003-large-export-batching)
- SQLite with FTS5 full-text search (existing); local filesystem for export files (003-large-export-batching)

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
- 003-large-export-batching: Added Go 1.21+ + Go stdlib only (database/sql, encoding/csv, encoding/json, io, os, path/filepath, time) + modernc.org/sqlite (existing)
- 002-archive-export: Added Go 1.21+ (existing project standard) + Go stdlib only (encoding/csv, encoding/json, io, os, path/filepath, time)
- 001-web-interface: Added Go 1.21+

<!-- MANUAL ADDITIONS START -->
Kill Go process when finished with work.
Do not kill ngrok processes!
<!-- MANUAL ADDITIONS END -->
