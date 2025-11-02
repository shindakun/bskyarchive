# Bug Fix: Export Records Not Being Saved to Database

## Issue Description

**Symptom**: Exports were being created successfully on the filesystem, but the export list in the UI remained blank. The database `exports` table was empty even after successful exports.

**User Report**: "Your export list is blank even after successful export"

## Root Cause Analysis

### The Problem

The issue was in the path handling between `CreateExportDirectory()` and `ExportRecord.Validate()`:

1. **Path Creation** ([internal/exporter/exporter.go:21-36](internal/exporter/exporter.go:21-36)):
   - Input: `baseDir = "./exports"`, `did = "did:plc:xxx"`
   - `filepath.Join("./exports", "did:plc:xxx")` → normalizes to `"exports/did:plc:xxx"` (removes `./` prefix)
   - Result: `exportDir = "exports/did:plc:xxx/2025-11-01_20-27-49"` (no `./` prefix)

2. **Validation Check** ([internal/models/export.go:182-184](internal/models/export.go:182-184)):
   ```go
   if !strings.HasPrefix(e.DirectoryPath, "./exports/") {
       return fmt.Errorf("invalid directory path (security)")
   }
   ```
   - Requires paths to start with `"./exports/"`
   - Received path: `"exports/did:plc:xxx/2025-11-01_20-27-49"`
   - Validation failed silently (only logged as warning)

3. **Silent Failure** ([internal/exporter/exporter.go:298-302](internal/exporter/exporter.go:298-302)):
   ```go
   if err := storage.CreateExportRecord(db, exportRecord); err != nil {
       log.Printf("Warning: Failed to save export record to database: %v", err)
       // Don't fail the export - this is metadata only
   }
   ```
   - Export continued successfully
   - Database record was never created
   - User saw "No exports yet" in the UI

### Why filepath.Join Strips `./`

Go's `filepath.Join` normalizes paths by:
- Removing redundant separators
- Eliminating `.` and `..` segments
- Converting to the OS-specific path separator

When joining `"./exports"` with `"did:plc:xxx"`, it normalizes to `"exports/did:plc:xxx"`, losing the `./` prefix.

## Solution

Modified `CreateExportDirectory()` to preserve the `./` prefix for relative paths:

```go
func CreateExportDirectory(baseDir string, did string) (string, error) {
    // Preserve ./ prefix for relative paths
    // filepath.Join normalizes paths and can strip ./ prefix
    preserveDotSlash := strings.HasPrefix(baseDir, "./")

    // Create per-user subdirectory structure
    userDir := filepath.Join(baseDir, did)
    if err := os.MkdirAll(userDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create user export directory: %w", err)
    }

    // Generate timestamp in filesystem-safe format (YYYY-MM-DD_HH-MM-SS)
    timestamp := time.Now().Format("2006-01-02_15-04-05")
    exportDir := filepath.Join(userDir, timestamp)

    // Create the timestamped directory
    if err := os.MkdirAll(exportDir, 0755); err != nil {
        return "", fmt.Errorf("failed to create export directory: %w", err)
    }

    // Restore ./ prefix if it was present in baseDir and got stripped by filepath.Join
    if preserveDotSlash && !strings.HasPrefix(exportDir, "./") && !filepath.IsAbs(exportDir) {
        exportDir = "./" + exportDir
    }

    return exportDir, nil
}
```

### How It Works

1. **Detect** if `baseDir` starts with `"./"`
2. **Use** `filepath.Join` normally for path construction (gets OS-specific separators)
3. **Restore** the `"./"`prefix after joining if:
   - The original `baseDir` had `"./"`
   - The result doesn't already have `"./"` (wasn't restored by Join)
   - The path isn't absolute (don't add `"./"` to `/tmp/exports`)

## Files Modified

1. [internal/exporter/exporter.go](internal/exporter/exporter.go)
   - Added `"strings"` import
   - Modified `CreateExportDirectory()` to preserve `./` prefix
   - Added logic to restore prefix after `filepath.Join`

2. [internal/exporter/path_fix_test.go](internal/exporter/path_fix_test.go) (new file)
   - Added comprehensive tests for path prefix preservation
   - Tests relative paths with `./`, without `./`, and absolute paths

## Testing

### Unit Tests Added

```bash
go test ./internal/exporter -run TestCreateExportDirectory_PreservesDotSlash -v
```

**Test Cases**:
1. ✓ Preserves `./` prefix for relative paths (`./exports` → `./exports/did:plc:test/...`)
2. ✓ Handles paths without `./` prefix (`exports` → `exports/did:plc:test/...`)
3. ✓ Handles absolute paths correctly (`/tmp/exports` → `/tmp/exports/did:plc:test/...`)

### Existing Tests

All existing tests continue to pass:
- ✓ `internal/exporter` tests (CSV, JSON, download, Run)
- ✓ `internal/storage` tests (CreateExportRecord, validation, CRUD operations)
- ✓ `internal/web/handlers` tests (DeleteExport, cleanup, concurrency)

## Impact

### Before Fix
- ❌ Exports created on filesystem but not tracked in database
- ❌ UI showed "No exports yet" despite successful exports
- ❌ Download and delete functionality unavailable
- ❌ Silent failure with only warning log

### After Fix
- ✅ Exports properly tracked in database
- ✅ UI displays export list with metadata
- ✅ Download and delete buttons available
- ✅ Future exports will be automatically tracked

## Verification Steps

To verify the fix works:

1. **Start the application**:
   ```bash
   go run ./cmd/bskyarchive
   ```

2. **Create a new export** via the web UI at `/export`

3. **Verify database record**:
   ```bash
   sqlite3 data/archive.db "SELECT id, directory_path FROM exports;"
   ```
   - Should show: `did:plc:xxx/2025-11-01_...|./exports/did:plc:xxx/2025-11-01_...`

4. **Verify UI** at `/export`:
   - Should display the export in the table
   - Download and Delete buttons should be visible

## Migration Note

### Existing Exports

Old exports created before this fix:
- Exist on filesystem: `./exports/did:plc:xxx/YYYY-MM-DD_HH-MM-SS/`
- **NOT in database** (validation failed)
- **NOT visible in UI**

**These will NOT be automatically migrated.** They can be:
1. Manually deleted from filesystem to clean up
2. Manually inserted into database (requires calculating size and metadata)
3. Left as-is (won't appear in UI but files remain accessible)

### Fresh Start Recommended

For simplest resolution:
```bash
# Backup if needed
mv exports exports.backup

# Clean slate
mkdir -p exports

# Verify database is empty
sqlite3 data/archive.db "DELETE FROM exports;"

# Create new export via UI - should now track properly
```

## Related Tasks

- ✅ T001-T004: Database migration and models (completed)
- ✅ T005-T010: Storage layer implementation (completed)
- ✅ T009: Update exporter.Run() to create ExportRecord (was failing silently - now fixed)
- ✅ T011-T021: Download functionality (completed)
- ✅ T022-T040: Browse and delete functionality (completed)

## Lessons Learned

1. **Validation should be louder**: Silent failures in critical paths make debugging difficult
2. **Path handling is subtle**: `filepath.Join` behavior with `./` isn't intuitive
3. **Security validation matters**: The `./exports/` prefix check prevents path traversal
4. **Test path scenarios**: Always test relative paths with/without `./` and absolute paths
5. **Check logs carefully**: The warning was logged but easy to miss

## References

- Issue: Export list blank after successful export
- Root cause: Path normalization removing `./` prefix
- Validation: Security check requiring `./exports/` prefix
- Solution: Preserve `./` prefix after `filepath.Join`
