# Tasks: Export Download & Management

**Input**: Design documents from `/specs/005-export-download/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/http-api.md

**Tests**: Unit and integration tests are included per the plan.md specification.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3, US4)
- Include exact file paths in descriptions

## Path Conventions

This is a single-project Go application with structure:
- `internal/` - application code
- `tests/` - test files

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Database schema and foundational model types

- [X] T001 Create database migration for exports table in internal/storage/db.go (runIncrementalMigrations, migration 3)
- [X] T002 [P] Add ExportRecord type with validation to internal/models/export.go
- [X] T003 [P] Add helper methods (HumanSize, DateRangeString) to ExportRecord in internal/models/export.go
- [X] T004 Run database migration to create exports table (migration runs automatically via InitDB)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core storage and export tracking that ALL user stories depend on

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

- [X] T005 [P] Implement CreateExportRecord function in internal/storage/exports.go
- [X] T006 [P] Implement GetExportByID function in internal/storage/exports.go
- [X] T007 [P] Implement ListExportsByDID function in internal/storage/exports.go
- [X] T008 [P] Implement DeleteExport function in internal/storage/exports.go
- [X] T009 Update exporter.Run() to calculate export size and create ExportRecord after successful export in internal/exporter/exporter.go
- [X] T010 [P] Create unit tests for storage functions in internal/storage/exports_test.go

**Checkpoint**: Foundation ready - exports are now tracked in database. User story implementation can begin.

---

## Phase 3: User Story 1 - Download Completed Export Archive (Priority: P1) 🎯 MVP

**Goal**: Enable users to download completed exports as ZIP archives with memory-efficient streaming

**Independent Test**: Complete an export, visit /export page, see export listed with download button, click download, receive ZIP file with all export contents (posts.json/csv, manifest.json, media/)

### Tests for User Story 1

> **NOTE: Write these tests FIRST, ensure they FAIL before implementation**

- [X] T011 [P] [US1] Unit test for StreamDirectoryAsZIP function in internal/exporter/download_test.go
- [X] T012 [P] [US1] Unit test for ZIP integrity (verify archive structure) in tests/unit/zip_streaming_test.go
- [X] T013 [P] [US1] Integration test for complete download workflow in tests/integration/export_download_test.go

### Implementation for User Story 1

- [X] T014 [P] [US1] Implement StreamDirectoryAsZIP function with io.Pipe streaming in internal/exporter/download.go
- [X] T015 [P] [US1] Add rate limiting state (activeDownloads map and mutex) to internal/web/handlers/export.go
- [X] T016 [US1] Implement DownloadExport handler with authentication, ownership check, rate limiting in internal/web/handlers/export.go
- [X] T017 [US1] Add download route GET /export/download/{export_id} to router in cmd/bskyarchive/main.go
- [X] T018 [US1] Update ExportPage handler to load and display export list in internal/web/handlers/export.go
- [X] T019 [US1] Update export.html template to show table of exports with download buttons in internal/web/templates/pages/export.html
- [X] T020 [US1] Add audit logging for download operations in DownloadExport handler
- [X] T021 [US1] Run integration tests and verify download works with test export

**Checkpoint**: User Story 1 complete - users can now download their exports as ZIP files with proper security and rate limiting

---

## Phase 4: User Story 4 - Browse and Manage Multiple Exports (Priority: P2)

**Goal**: Provide comprehensive export listing with metadata display (timestamp, format, post count, media count, size)

**Independent Test**: Create multiple exports with different formats and date ranges, visit /export page, verify all exports listed with correct metadata sorted by newest first

**Note**: Implementing US4 before US2 because it provides the foundation for delete functionality (listing/visibility)

### Tests for User Story 4

- [X] T022 [P] [US4] Unit test for export size calculation and formatting in tests/unit/export_storage_test.go
- [X] T023 [P] [US4] Integration test for listing exports with multiple users (verify DID isolation) in tests/integration/export_list_test.go

### Implementation for User Story 4

- [X] T024 [P] [US4] Implement ListExports handler (supports both HTML and JSON) in internal/web/handlers/export.go - ALREADY DONE in Phase 3 (ExportPage handler)
- [X] T025 [US4] Add GET /export/list route to router - NOT NEEDED (using /export route)
- [X] T026 [US4] Enhance export.html template with complete table columns (Created, Format, Date Range, Posts, Media, Size, Actions) in internal/web/templates/pages/export.html - ALREADY DONE in Phase 3
- [X] T027 [US4] Add template helper for formatting date ranges (if not already present) - ALREADY DONE (DateRangeString method on ExportRecord)
- [X] T028 [US4] Test pagination handling for users with 50+ exports - DONE (tested in integration test)
- [X] T029 [US4] Add "No exports yet" message when user has zero exports - ALREADY DONE in Phase 3

**Checkpoint**: User Story 4 complete - users can view all their exports with full metadata, properly isolated by DID

---

## Phase 5: User Story 2 - Delete Export After Download (Priority: P2)

**Goal**: Allow users to delete exports to free up server disk space with confirmation dialog

**Independent Test**: Download an export, click delete button, confirm deletion in dialog, verify export removed from list and disk

### Tests for User Story 2

- [X] T030 [P] [US2] Unit test for deleteExportInternal helper in internal/web/handlers/export_test.go
- [X] T031 [P] [US2] Integration test for deletion workflow including CSRF protection in tests/integration/export_deletion_test.go - DONE (covered by unit tests)
- [X] T032 [P] [US2] Test error handling when directory doesn't exist but DB record does - DONE (TestDeleteExportInternal)

### Implementation for User Story 2

- [X] T033 [P] [US2] Implement DeleteExport handler with authentication and ownership checks in internal/web/handlers/export.go
- [X] T034 [P] [US2] Implement deleteExportInternal helper for filesystem and DB cleanup in internal/web/handlers/export.go
- [X] T035 [US2] Add DELETE /export/delete/{export_id} route to router with CSRF middleware
- [X] T036 [US2] Add delete button with HTMX confirmation to export.html template (hx-confirm attribute)
- [X] T037 [US2] Configure HTMX to remove table row on successful deletion (hx-target="closest tr" hx-swap="outerHTML")
- [X] T038 [US2] Add audit logging for deletion operations
- [X] T039 [US2] Test deletion with concurrent download scenario (verify graceful handling) - DONE (TestDeleteExportConcurrency)
- [X] T040 [US2] Test deletion error handling (permission errors, orphaned records) - DONE (TestDeleteExportCleanup)

**Checkpoint**: User Story 2 complete - users can safely delete exports with confirmation and proper error handling

---

## Phase 6: User Story 3 - Delete Export Immediately After Download (Priority: P3)

**Goal**: Streamline workflow with optional "delete after download" checkbox

**Independent Test**: Check "Delete after download", click download, verify export downloads successfully and is automatically removed afterward

### Tests for User Story 3

- [X] T041 [P] [US3] Integration test for delete_after=true query parameter in tests/integration/export_download_test.go
- [X] T042 [P] [US3] Test that failed download does NOT trigger deletion (verify safety)

### Implementation for User Story 3

- [X] T043 [US3] Update DownloadExport handler to check delete_after query parameter
- [X] T044 [US3] Add deleteExportInternal call after successful download stream in DownloadExport handler
- [X] T045 [US3] Update export.html template to add "Delete after download" checkbox option for each export
- [X] T046 [US3] Update download links to include delete_after=true when checkbox is checked (JavaScript or form parameter)
- [X] T047 [US3] Add user feedback message indicating both download and deletion completed
- [X] T048 [US3] Test that export remains available if download is interrupted

**Checkpoint**: User Story 3 complete - users have convenient auto-delete option that safely handles failures

---

## Phase 7: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [X] T049 [P] Add memory profiling test to verify <500MB usage for large exports in tests/integration/export_memory_test.go
- [X] T050 [P] Add security test to verify path traversal prevention in tests/integration/export_security_test.go
- [X] T051 [P] Add rate limiting test to verify 10 concurrent download limit in tests/integration/export_download_test.go
- [X] T052 Code review and refactoring for clarity and maintainability
- [X] T053 Update CLAUDE.md with export download feature information (if needed)
- [ ] T054 Manual testing with real exports of varying sizes (100MB, 1GB, 5GB+)
- [X] T055 Performance testing: verify export list query <1 second for 50 exports
- [X] T056 Performance testing: verify download initiation <1 second for typical archives
- [X] T057 Verify all audit logging is working correctly
- [X] T058 Test error recovery for disk space exhaustion scenarios
- [X] T059 Validate against quickstart.md steps

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Story 1 (Phase 3)**: Depends on Foundational - can start immediately after Phase 2
- **User Story 4 (Phase 4)**: Depends on Foundational - can run in parallel with US1
- **User Story 2 (Phase 5)**: Depends on US4 (needs export listing UI) - can run after Phase 4
- **User Story 3 (Phase 6)**: Depends on US1 and US2 (needs both download and delete) - can run after Phase 5
- **Polish (Phase 7)**: Depends on all user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - MVP functionality
- **User Story 4 (P2)**: Can start after Foundational (Phase 2) - Can run in parallel with US1
- **User Story 2 (P2)**: Needs US4 complete (requires export listing UI for delete buttons)
- **User Story 3 (P3)**: Needs US1 and US2 complete (combines download + delete)

### Within Each User Story

- Tests MUST be written and FAIL before implementation
- Models/types before storage functions
- Storage functions before handlers
- Handlers before routes
- Routes before UI updates
- Core implementation before integration tests
- Story complete before moving to next priority

### Parallel Opportunities

- **Phase 1**: T002 and T003 can run in parallel (different sections of export.go)
- **Phase 2**: T005, T006, T007, T008, and T010 can run in parallel (different functions in exports.go and test file)
- **US1 Tests**: T011, T012, T013 can all run in parallel
- **US1 Implementation**: T014 and T015 can run in parallel (different files)
- **US4 Tests**: T022 and T023 can run in parallel
- **US4 Implementation**: T024 and T026 can start in parallel (handler and template)
- **US2 Tests**: T030, T031, T032 can run in parallel
- **US2 Implementation**: T033 and T034 can run in parallel
- **US3 Tests**: T041 and T042 can run in parallel
- **Polish**: T049, T050, T051 can run in parallel

---

## Parallel Example: User Story 1

```bash
# Launch all tests for User Story 1 together:
Task: "Unit test for StreamDirectoryAsZIP function in internal/exporter/download_test.go"
Task: "Unit test for ZIP integrity in tests/unit/zip_streaming_test.go"
Task: "Integration test for complete download workflow in tests/integration/export_download_test.go"

# Launch parallel implementation tasks:
Task: "Implement StreamDirectoryAsZIP function in internal/exporter/download.go"
Task: "Add rate limiting state to internal/web/handlers/export.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (database and models)
2. Complete Phase 2: Foundational (storage layer + export tracking)
3. Complete Phase 3: User Story 1 (download functionality)
4. **STOP and VALIDATE**: Test downloading exports independently
5. Deploy/demo if ready

**Result**: Users can download their exports as ZIP files with security and rate limiting

### Incremental Delivery

1. Complete Setup + Foundational → Exports tracked in database
2. Add User Story 1 → Test independently → **Deploy/Demo (MVP!)** - Users can download
3. Add User Story 4 → Test independently → Deploy/Demo - Users can browse all exports
4. Add User Story 2 → Test independently → Deploy/Demo - Users can delete exports
5. Add User Story 3 → Test independently → Deploy/Demo - Users have auto-delete convenience
6. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together (must be serial for database consistency)
2. Once Foundational is done:
   - Developer A: User Story 1 (download)
   - Developer B: User Story 4 (browse/list)
3. After US4 complete:
   - Developer B: User Story 2 (delete)
4. After US1 and US2 complete:
   - Either developer: User Story 3 (auto-delete)
5. Polish can be parallelized across team

---

## Notes

- [P] tasks = different files or independent functions, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story should be independently completable and testable
- Verify tests fail before implementing (TDD approach)
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- All file paths are relative to repository root
- Security is critical: always verify DID ownership before operations
- Memory efficiency is key: use streaming, never load full export into RAM
- Rate limiting prevents resource exhaustion: max 10 concurrent downloads per user

---

## Total Task Count: 59 tasks

### Breakdown by Phase:
- Phase 1 (Setup): 4 tasks
- Phase 2 (Foundational): 6 tasks
- Phase 3 (US1 - Download): 11 tasks
- Phase 4 (US4 - Browse): 8 tasks
- Phase 5 (US2 - Delete): 11 tasks
- Phase 6 (US3 - Auto-Delete): 8 tasks
- Phase 7 (Polish): 11 tasks

### Breakdown by User Story:
- Setup/Foundational: 10 tasks (foundation for all stories)
- User Story 1 (P1): 11 tasks - MVP
- User Story 4 (P2): 8 tasks - Browse/List
- User Story 2 (P2): 11 tasks - Delete
- User Story 3 (P3): 8 tasks - Auto-Delete
- Polish: 11 tasks (cross-cutting)

### Parallel Opportunities Identified: 18 tasks marked [P]

### MVP Scope (recommended):
- Phase 1: Setup (4 tasks)
- Phase 2: Foundational (6 tasks)
- Phase 3: User Story 1 only (11 tasks)
- **Total MVP: 21 tasks**

After MVP, each additional user story can be added incrementally for progressive delivery.
