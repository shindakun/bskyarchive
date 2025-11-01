# Feature Specification: Archive Export

**Feature Branch**: `002-archive-export`
**Created**: 2025-10-30
**Status**: Draft
**Input**: User description: "We need the ability to export the downloaded archives. Consider output files like csv and json with images being included in an output directory."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Export Complete Archive to JSON (Priority: P1)

A user wants to export their entire archived Bluesky data to a JSON file for backup or analysis. They navigate to the archive management page, select "Export to JSON", and the system generates a complete JSON file containing all posts with metadata. Media files are exported to an organized directory structure that maintains the relationship between posts and their images.

**Why this priority**: JSON export is the most complete and flexible export format. It preserves the full data structure including nested objects, arrays, and all metadata fields. This is essential for users who want a complete backup, plan to migrate to another system, or need programmatic access to their archive data.

**Independent Test**: Can be fully tested by initiating a JSON export, verifying the generated file is valid JSON, checking that all posts are included with complete metadata, and confirming all media files are present in the output directory with correct filenames that link to posts.

**Acceptance Scenarios**:

1. **Given** a user has archived posts in their database, **When** they initiate a JSON export, **Then** a valid JSON file is generated containing all posts with complete metadata (URI, CID, DID, text, timestamps, engagement metrics, embed data)
2. **Given** a user's archive contains posts with images, **When** they export to JSON, **Then** media files are copied to a `/media` subdirectory with filenames matching the hash references in the JSON
3. **Given** a user exports to JSON, **When** the export completes, **Then** the JSON file maintains the original data types (timestamps as ISO 8601, counts as integers, booleans as true/false)
4. **Given** a user has 1000+ posts, **When** they export to JSON, **Then** the entire archive exports successfully without memory issues or timeouts
5. **Given** a user exports their archive, **When** they open the JSON file, **Then** they can parse it with standard JSON tools and navigate the data structure

---

### User Story 2 - Export to CSV for Analysis (Priority: P2)

A user wants to analyze their Bluesky posting patterns using spreadsheet software like Excel or Google Sheets. They select "Export to CSV", and the system generates a CSV file with one row per post and columns for all key fields. The CSV is formatted for easy import into spreadsheet applications and data analysis tools.

**Why this priority**: CSV export enables non-technical users to work with their data in familiar spreadsheet applications. This is crucial for users who want to analyze posting frequency, search for patterns, create visualizations, or share data in a universally readable format. While less complete than JSON, it's more accessible to average users.

**Independent Test**: Can be tested by exporting to CSV, opening the file in Excel/Google Sheets, verifying all posts appear as rows with readable data in columns, and confirming media references are preserved in a dedicated column.

**Acceptance Scenarios**:

1. **Given** a user has archived posts, **When** they initiate a CSV export, **Then** a valid CSV file is generated with headers for all key fields (URI, CID, DID, Text, CreatedAt, LikeCount, RepostCount, ReplyCount, QuoteCount, HasMedia, MediaFiles)
2. **Given** a user exports to CSV, **When** they open the file in Excel or Google Sheets, **Then** all columns are properly formatted, text with commas and quotes is properly escaped, and timestamps are readable
3. **Given** a post contains line breaks or special characters, **When** it's exported to CSV, **Then** the text is properly quoted and escaped to preserve formatting
4. **Given** a post has multiple media files, **When** it's exported to CSV, **Then** the MediaFiles column contains a semicolon-separated list of filenames from the `/media` directory
5. **Given** a user exports to CSV, **When** media files are present, **Then** they are copied to a `/media` subdirectory just like JSON export

---

### User Story 3 - Selective Export by Date Range (Priority: P3)

A user wants to export only posts from a specific time period (e.g., "2024 posts only" or "last 6 months"). They access export options, specify a start and end date, and receive an export containing only posts created within that range. This allows users to create focused exports for specific analysis or sharing.

**Why this priority**: While useful, selective export is a convenience feature that enhances the core export functionality. Users can always export everything and filter manually, though this feature makes the workflow more efficient for targeted exports.

**Independent Test**: Can be tested by specifying a date range filter, initiating an export, and verifying that only posts with created_at timestamps within the specified range are included in the output.

**Acceptance Scenarios**:

1. **Given** a user has posts spanning multiple years, **When** they specify a date range filter and export, **Then** only posts with created_at timestamps within that range are included
2. **Given** a user specifies a date range with no matching posts, **When** they attempt export, **Then** the system shows a clear message indicating no posts match the criteria
3. **Given** a user exports with a date filter, **When** posts in that range have media, **Then** only the media files associated with filtered posts are included in the `/media` directory
4. **Given** a user selects a date range, **When** they export to CSV, **Then** the file contains only the filtered posts but maintains all column headers and formatting

---

### Edge Cases

- What happens when a user tries to export but has no archived posts?
- How does the system handle extremely large exports (10,000+ posts with thousands of media files)?
- What happens when the export directory already contains files from a previous export?
- How does the system handle media files that are referenced in the database but missing from disk?
- What happens when the user's disk is full and export fails mid-process?
- How does the system handle posts with extremely long text fields (approaching SQLite's limit)?
- What happens when a media file has a non-standard extension or corrupted data?
- How does the system handle Unicode characters, emoji, and non-Latin scripts in CSV export?
- What happens when a user initiates multiple exports simultaneously?

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST provide an export interface accessible from the archive management page
- **FR-002**: System MUST support exporting the complete archive to JSON format
- **FR-003**: System MUST support exporting the complete archive to CSV format
- **FR-004**: System MUST include all post metadata in JSON exports (URI, CID, DID, text, created_at, indexed_at, has_media, like_count, repost_count, reply_count, quote_count, is_reply, reply_parent, embed_type, embed_data, labels, archived_at)
- **FR-005**: System MUST include key post fields in CSV exports (URI, CID, DID, text, created_at, like_count, repost_count, reply_count, quote_count, is_reply, has_media, media_files)
- **FR-006**: System MUST copy all media files referenced by exported posts to a `/media` subdirectory in the export directory
- **FR-007**: System MUST maintain filename consistency where media filenames in the export match the hash-based filenames in the archive
- **FR-008**: System MUST properly escape CSV fields containing commas, quotes, and newlines according to RFC 4180
- **FR-009**: System MUST format timestamps in ISO 8601 format in both JSON and CSV exports
- **FR-010**: System MUST handle posts with multiple media files by creating a structured reference in CSV (semicolon-separated list)
- **FR-011**: System MUST create a timestamped export directory to prevent overwriting previous exports (e.g., `exports/2025-10-30_14-30-45/`)
- **FR-012**: System MUST provide export progress indicators showing posts processed and media files copied
- **FR-013**: System MUST support optional date range filters for selective export (start date, end date)
- **FR-014**: System MUST validate date range inputs and reject invalid date ranges (end before start)
- **FR-015**: System MUST handle missing media files gracefully by logging warnings but continuing the export
- **FR-016**: System MUST validate available disk space before starting large exports and warn users if insufficient
- **FR-017**: System MUST generate a manifest file (manifest.json) in the export directory listing what was exported (post count, media count, date range, format, timestamp)
- **FR-018**: System MUST prevent simultaneous export operations by the same user

### Key Entities

- **ExportJob**: Represents an export operation including format (JSON/CSV), date range filter (optional), output directory path, progress tracking (posts processed, media copied), status (queued/running/completed/failed), and any error messages
- **ExportFormat**: Defines the structure of each export type (JSON with nested objects, CSV with flat rows and columns)
- **MediaManifest**: List of media files included in an export, mapping post URIs to their associated media file hashes and paths
- **ExportDirectory**: Output location structure containing the data file (posts.json or posts.csv), /media subdirectory, and manifest.json metadata file

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can export 1,000 posts to JSON format in under 10 seconds on standard hardware
- **SC-002**: Users can export 1,000 posts to CSV format in under 10 seconds on standard hardware
- **SC-003**: Exported JSON files are 100% parseable by standard JSON parsers (Python json.load, jq, etc.)
- **SC-004**: Exported CSV files are 100% importable into Excel and Google Sheets without encoding or formatting errors
- **SC-005**: 100% of media files referenced in exported posts are present in the /media directory (or logged as missing)
- **SC-006**: Export operations provide progress updates at least every 2 seconds during execution
- **SC-007**: Users can export archives containing 10,000+ posts without memory issues or application crashes
- **SC-008**: CSV exports handle Unicode characters, emoji, and non-Latin scripts correctly without corruption
- **SC-009**: Date range filtering reduces export size proportionally (exporting 10% of date range results in ~10% of posts)
- **SC-010**: Export manifest files contain accurate metadata matching the actual export contents (post count, media count)

## Assumptions

- Exports are performed on-demand by the user and are not automated background tasks
- Export files are stored locally on the same machine as the archive database
- Users have sufficient disk space for exports (roughly 2x the current archive size for safety)
- The export directory is configurable but defaults to `./exports/` in the application root
- Each export creates a new timestamped subdirectory to preserve previous exports
- Users are expected to manually move or backup export directories to external storage if needed
- The system does not compress exports (users can manually zip the export directory)
- CSV exports flatten nested data structures (embed_data and labels are either omitted or stringified)
- JSON exports preserve the exact database schema structure for each post
- Media files are copied (not moved) so the original archive remains intact
