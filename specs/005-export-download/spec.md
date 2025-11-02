# Feature Specification: Export Download & Management

**Feature Branch**: `005-export-download`
**Created**: 2025-11-01
**Status**: Draft
**Input**: User description: "Plan out a feature up for users to download the exports as files. Give them the option to delete the archive on the server afterward"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Download Completed Export Archive (Priority: P1)

A user has completed an export and wants to download it to their local machine so they can back it up externally or transfer it to another system.

**Why this priority**: This is the core functionality - without the ability to download exports, they remain trapped on the server. This delivers immediate value by making exports portable.

**Independent Test**: Can be fully tested by completing an export, clicking a download button, and receiving a ZIP archive containing all exported files.

**Acceptance Scenarios**:

1. **Given** a user has completed an export, **When** they view the export page, **Then** they see a list of their completed exports with download buttons
2. **Given** a user clicks the download button, **When** the download request is processed, **Then** they receive a ZIP file containing all export files (posts.json/csv, manifest.json, and media/ directory if included)
3. **Given** a user downloads a large export (>100MB), **When** the download is in progress, **Then** they see accurate download progress and the file streams without loading entirely into server memory
4. **Given** a user downloads an export, **When** the ZIP file is opened, **Then** all original files maintain their directory structure and integrity

---

### User Story 2 - Delete Export After Download (Priority: P2)

A user has successfully downloaded their export and wants to free up disk space on the server by deleting the export files.

**Why this priority**: Storage management is important for server resources, but the user must have the export first. This is a natural follow-up to downloading.

**Independent Test**: After downloading an export, click a delete button with confirmation, verify the export is removed from the list and disk.

**Acceptance Scenarios**:

1. **Given** a user views their completed exports, **When** they click delete on an export, **Then** they see a confirmation dialog warning them the deletion is permanent
2. **Given** a user confirms deletion, **When** the deletion completes, **Then** the export is removed from the list and all files are deleted from the server
3. **Given** a user deletes an export, **When** they refresh the page, **Then** the deleted export does not reappear
4. **Given** a user attempts to delete an export, **When** an error occurs, **Then** they see an error message and the export remains available

---

### User Story 3 - Delete Export Immediately After Download (Priority: P3)

A user wants to streamline their workflow by automatically deleting the export from the server immediately after successful download.

**Why this priority**: This is a convenience feature that saves a click but is not essential. Users can achieve the same result with two separate actions.

**Independent Test**: Enable "Delete after download" checkbox, download an export, verify it's automatically removed after download completes.

**Acceptance Scenarios**:

1. **Given** a user is viewing an export, **When** they check "Delete after download" and click download, **Then** the export is automatically deleted upon successful download completion
2. **Given** a user enables "Delete after download", **When** the download fails or is interrupted, **Then** the export is NOT deleted and remains available
3. **Given** a user downloads with auto-delete enabled, **When** the process completes, **Then** they see a confirmation message indicating both successful download and deletion

---

### User Story 4 - Browse and Manage Multiple Exports (Priority: P2)

A user has created multiple exports over time and wants to see a list of all exports, their details, and manage them efficiently.

**Why this priority**: Users will naturally create multiple exports with different date ranges or formats. This provides essential visibility and management.

**Independent Test**: Create multiple exports with different parameters, view the export management page showing all exports with metadata.

**Acceptance Scenarios**:

1. **Given** a user has multiple completed exports, **When** they view the export page, **Then** they see a table/list showing export timestamp, format, post count, media count, and file size
2. **Given** a user views the export list, **When** they see each export, **Then** each entry displays its date range filter (if any) and format type
3. **Given** a user has many exports, **When** the list loads, **Then** exports are sorted by creation time (newest first) and paginated if necessary
4. **Given** a user has exports for multiple DIDs (future multi-user scenario), **When** they view exports, **Then** they only see exports belonging to their account

---

### Edge Cases

- What happens when a user tries to download an export that no longer exists on disk?
  - Show appropriate error message: "Export not found. It may have been deleted."
- What happens when a user tries to download an export while one is being created?
  - Only completed exports have download buttons; in-progress exports show status instead
- What happens when disk space is critically low during ZIP creation?
  - Fail gracefully with error message and don't create partial ZIP files
- What happens when a user closes their browser during download?
  - Standard browser download resume behavior applies; server doesn't track download state
- What happens when a user tries to delete an export that's currently being downloaded by them or another session?
  - Deletion proceeds (no locking); if download was in progress, it may fail or receive partial data
- What happens when export directory permissions prevent deletion?
  - Show error message and log the issue; export remains in list
- How does the system handle exports with very large media folders (e.g., 10GB+)?
  - Stream ZIP creation to avoid memory exhaustion; show size warnings in UI

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST list all completed exports for the authenticated user showing timestamp, format, date range, post count, media count, and estimated size
- **FR-002**: System MUST provide a download endpoint that streams a ZIP archive containing the export directory contents
- **FR-003**: System MUST preserve directory structure inside ZIP files (media/ subdirectory, manifest.json, posts.json/csv)
- **FR-004**: System MUST stream ZIP file creation to handle large exports without exhausting memory
- **FR-005**: System MUST provide a delete endpoint that removes an export directory and all its contents from disk
- **FR-006**: System MUST show a confirmation dialog before deleting exports to prevent accidental data loss
- **FR-007**: System MUST verify export ownership before allowing download or deletion (security requirement)
- **FR-008**: System MUST handle missing export directories gracefully (e.g., manually deleted files)
- **FR-009**: System MUST provide an optional "Delete after download" option that removes the export upon successful download
- **FR-010**: System MUST calculate and display export file sizes in human-readable format (KB, MB, GB)
- **FR-011**: System MUST prevent concurrent deletion of the same export (idempotent delete operations)
- **FR-012**: System MUST validate ZIP archive integrity before serving (checksum or similar)
- **FR-013**: System MUST log all download and deletion operations for audit purposes
- **FR-014**: System MUST set appropriate HTTP headers for ZIP downloads (Content-Disposition, Content-Type, Content-Length)
- **FR-015**: System MUST rate-limit download endpoints to prevent abuse (e.g., max 10 concurrent downloads per user)

### Key Entities

- **Export Record**: Represents a completed export with metadata (ID, timestamp, format, owner DID, directory path, post count, media count, size)
- **Export Directory**: Physical directory structure on disk containing export files
- **Download Session**: Tracks an active download operation for rate limiting and monitoring
- **Deletion Confirmation**: User interaction requiring explicit confirmation before irreversible action

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can download a completed export as a ZIP file in under 10 seconds for typical archives (<100MB)
- **SC-002**: System can stream ZIP files up to 5GB without exceeding 500MB memory usage
- **SC-003**: 100% of export downloads maintain file integrity (verified via checksums in manifest)
- **SC-004**: Users can delete exports successfully 99.9% of the time (excluding permission errors)
- **SC-005**: Zero unauthorized access to other users' exports (enforced by DID ownership checks)
- **SC-006**: Export list loads in under 1 second for users with up to 50 completed exports
- **SC-007**: System recovers gracefully from 100% of disk space errors during download/deletion
- **SC-008**: 95% of users successfully download their first export without errors or confusion
