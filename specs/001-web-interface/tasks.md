# Implementation Tasks: Web Interface

**Feature**: 001-web-interface
**Branch**: `001-web-interface`
**Generated**: 2025-10-30
**Plan**: [plan.md](./plan.md) | **Spec**: [spec.md](./spec.md)

## Overview

This document breaks down the web interface feature into executable tasks organized by user story. Each phase represents an independently testable increment that delivers value.

**Total Tasks**: 59
**User Stories**: 3 (P1, P2, P3)
**Parallel Opportunities**: 28 parallelizable tasks

---

## User Story Mapping

| Story | Priority | Description | Tasks |
|-------|----------|-------------|-------|
| US1 | P1 | Authentication & Landing | 18 |
| US2 | P2 | Archive Management Pages | 26 |
| US3 | P3 | About Page & External Links | 5 |
| Setup | - | Project initialization | 6 |
| Foundation | - | Shared infrastructure | 4 |

---

## Dependencies & Execution Order

```
Phase 1 (Setup) → Phase 2 (Foundation) → Phase 3 (US1) → Phase 4 (US2) → Phase 5 (US3)
                                              ↓              ↓              ↓
                                           Complete     Complete       Complete
                                         independently independently independently
```

**Blocking Dependencies**:
- Phase 1 (Setup) MUST complete before any other phase
- Phase 2 (Foundation) MUST complete before user stories
- User Stories (US1, US2, US3) can be implemented independently after Foundation

**MVP Scope**: Phase 1 + Phase 2 + Phase 3 (US1 only) = Basic authentication and landing page

---

## Phase 1: Setup (Project Initialization)

**Goal**: Initialize Go project with all dependencies and directory structure

**Independent Test**: Run `go mod tidy` and `go build` successfully

### Tasks

- [x] T001 Initialize Go module with `go mod init github.com/shindakun/bskyarchive`
- [x] T002 Install all Go dependencies per plan.md (chi, gorilla/sessions, gorilla/csrf, bskyoauth, indigo, sqlite, yaml.v3, uuid)
- [x] T003 [P] Create directory structure per plan.md: cmd/bskyarchive, internal/{web,auth,archiver,storage,models}, tests/{unit,integration,contract}
- [x] T004 [P] Create web subdirectories: internal/web/{handlers,middleware,templates,static}
- [x] T005 [P] Create template subdirectories: internal/web/templates/{layouts,pages,partials}
- [x] T006 [P] Create static asset subdirectories: internal/web/static/{css,js,images}

**Parallel Execution**: T003, T004, T005, T006 can run concurrently (different directories)

---

## Phase 2: Foundation (Shared Infrastructure)

**Goal**: Implement core infrastructure needed by all user stories

**Independent Test**: Database connections succeed, config loads, sessions initialize

### Tasks

- [x] T007 Create config.yaml with server, archive, oauth, and rate_limit sections per research.md
- [x] T008 Implement configuration loading in internal/config/config.go with environment variable expansion
- [x] T009 Implement database initialization in internal/storage/db.go with production SQLite settings (WAL, NORMAL sync, 5s timeout, private cache, memory temp)
- [x] T010 [P] Download static assets: Pico CSS (pico.min.css) and HTMX (htmx.min.js) to internal/web/static/

**Parallel Execution**: T010 can run while T007-T009 are being implemented

---

## Phase 3: User Story 1 - Authentication & Landing (P1)

**Goal**: Users can visit landing page, authenticate via Bluesky OAuth, and reach dashboard

**Independent Test**:
1. Navigate to `/` → see landing page with login button
2. Click login → redirected to Bluesky OAuth
3. Complete OAuth → redirected to `/dashboard` with active session
4. Return to `/` while authenticated → auto-redirect to `/dashboard`
5. Session expires → redirect to `/` with message

### 3.1: Data Models (US1)

- [ ] T011 [P] [US1] Create Session model in internal/models/session.go with DID, Handle, AccessToken, ExpiresAt fields
- [ ] T012 [P] [US1] Create Profile model in internal/models/profile.go for user profile snapshots

**Parallel Execution**: T011 and T012 can run concurrently (different files)

### 3.2: Database Migrations (US1)

- [ ] T013 [US1] Create migration 001_initial.sql with sessions and profiles tables per data-model.md
- [ ] T014 [US1] Implement migration runner in internal/storage/db.go (runMigrations function)

### 3.3: Session Management (US1)

- [ ] T015 [US1] Implement session initialization in internal/auth/session.go (InitSessions with 7-day expiration, HTTP-only cookies)
- [ ] T016 [US1] Implement SaveSession function in internal/auth/session.go to store DID, handle, access token
- [ ] T017 [US1] Implement GetSession function in internal/auth/session.go to retrieve session data
- [ ] T018 [US1] Implement ClearSession function in internal/auth/session.go for logout

### 3.4: OAuth Integration (US1)

- [ ] T019 [US1] Implement OAuth client initialization in internal/auth/oauth.go (InitOAuth with callback URL and scopes)
- [ ] T020 [US1] Implement HandleOAuthLogin in internal/auth/oauth.go (generate auth URL, store state/verifier in session)
- [ ] T021 [US1] Implement HandleOAuthCallback in internal/auth/oauth.go (verify state, exchange code for tokens, save session)
- [ ] T022 [US1] Implement HandleLogout in internal/auth/oauth.go (clear session, redirect to landing)

### 3.5: Middleware (US1)

- [ ] T023 [US1] Implement RequireAuth middleware in internal/web/middleware/auth.go (check session, redirect if not authenticated)
- [ ] T024 [P] [US1] Implement logging middleware in internal/web/middleware/logging.go (log method, path, status, duration, DID)

**Parallel Execution**: T023 and T024 can run concurrently (different files)

### 3.6: Templates & Static Assets (US1)

- [ ] T025 [US1] Create base layout template in internal/web/templates/layouts/base.html with dark theme, Pico CSS, HTMX script
- [ ] T026 [US1] Create landing page template in internal/web/templates/pages/landing.html with login button and error/message display
- [ ] T027 [US1] Create dashboard page template in internal/web/templates/pages/dashboard.html (placeholder for US2)
- [ ] T028 [P] [US1] Create custom.css in internal/web/static/css/custom.css with dark theme overrides (Bluesky blue primary color)

**Parallel Execution**: T028 can run while T025-T027 are being implemented

### 3.7: HTTP Handlers (US1)

- [ ] T029 [US1] Implement Landing handler in internal/web/handlers/landing.go (check auth, redirect if authenticated, render landing template)
- [ ] T030 [US1] Implement Dashboard handler in internal/web/handlers/dashboard.go (stub for US2, render dashboard template)

### 3.8: Router & Main Application (US1)

- [ ] T031 [US1] Implement router in internal/web/router.go (chi router with public and protected route groups)
- [ ] T032 [US1] Implement main application in cmd/bskyarchive/main.go (load config, init DB, init sessions, init OAuth, start server, graceful shutdown)

---

## Phase 4: User Story 2 - Archive Management Pages (P2)

**Goal**: Authenticated users can initiate archive operations, monitor progress, and browse archived content

**Independent Test**:
1. Login as authenticated user
2. Navigate to `/archive` → see archive status
3. Click "Start Archive" → operation starts
4. Poll `/archive/status` → see progress updates
5. Navigate to `/browse` → see archived posts
6. Search posts → see filtered results

### 4.1: Data Models (US2)

- [ ] T033 [P] [US2] Create Post model in internal/models/post.go with URI, CID, Text, CreatedAt, engagement metrics
- [ ] T034 [P] [US2] Create Media model in internal/models/media.go with PostURI, LocalPath, MimeType
- [ ] T035 [P] [US2] Create ArchiveOperation model in internal/models/operation.go with Status, ProgressCurrent, ProgressTotal
- [ ] T036 [P] [US2] Create ArchiveStatus model in internal/models/status.go (derived data) with TotalPosts, TotalMedia, LastSyncAt

**Parallel Execution**: T033, T034, T035, T036 can all run concurrently (different files)

### 4.2: Database Migrations (US2)

- [ ] T037 [US2] Create migration 002_posts.sql with posts table, indexes, and FTS5 virtual table with triggers per data-model.md
- [ ] T038 [US2] Create migration 003_media.sql with media table and foreign key to posts
- [ ] T039 [US2] Create migration 004_operations.sql with archive_operations table

### 4.3: Storage Layer (US2)

- [ ] T040 [US2] Implement SavePost in internal/storage/posts.go with upsert logic
- [ ] T041 [US2] Implement GetPost in internal/storage/posts.go by URI
- [ ] T042 [US2] Implement ListPosts in internal/storage/posts.go with pagination (PagedPostsResponse)
- [ ] T043 [US2] Implement SearchPosts in internal/storage/search.go using FTS5 (SearchPostsResponse)
- [ ] T044 [US2] Implement SaveProfile in internal/storage/profiles.go for profile snapshots
- [ ] T045 [US2] Implement GetLatestProfile in internal/storage/profiles.go
- [ ] T046 [US2] Implement SaveMedia in internal/storage/media.go with local path generation
- [ ] T047 [US2] Implement ListMediaForPost in internal/storage/media.go
- [ ] T048 [US2] Implement CreateOperation in internal/storage/operations.go
- [ ] T049 [US2] Implement UpdateOperation in internal/storage/operations.go
- [ ] T050 [US2] Implement GetActiveOperation in internal/storage/operations.go
- [ ] T051 [US2] Implement GetArchiveStatus in internal/storage/status.go (aggregated query across posts, media, operations)

### 4.4: AT Protocol Integration (US2)

- [ ] T052 [US2] Implement AT Protocol client wrapper in internal/archiver/client.go (NewATProtoClient with auth)
- [ ] T053 [US2] Implement FetchPosts in internal/archiver/collector.go (paginated getAuthorFeed)
- [ ] T054 [US2] Implement FetchProfile in internal/archiver/collector.go (getProfile)
- [ ] T055 [US2] Implement DownloadMedia in internal/archiver/media.go with SHA-256 hash-based paths
- [ ] T056 [US2] Implement rate limiter in internal/archiver/ratelimit.go (token bucket, 300 req/5min)

### 4.5: Background Worker (US2)

- [ ] T057 [US2] Implement archiveWorker in internal/archiver/worker.go (goroutine with context cancellation, progress updates)
- [ ] T058 [US2] Implement StartArchive in internal/archiver/worker.go (check for active op, create operation, launch worker)

### 4.6: Templates (US2)

- [ ] T059 [US2] Update dashboard template internal/web/templates/pages/dashboard.html with archive status display
- [ ] T060 [US2] Create archive management template in internal/web/templates/pages/archive.html with start buttons and status polling
- [ ] T061 [US2] Create browse template in internal/web/templates/pages/browse.html with post list, pagination, search form
- [ ] T062 [US2] Create archive status partial in internal/web/templates/partials/archive-status.html for HTMX updates

### 4.7: HTTP Handlers (US2)

- [ ] T063 [US2] Update Dashboard handler in internal/web/handlers/dashboard.go to fetch and display ArchiveStatus
- [ ] T064 [US2] Implement Archive handler in internal/web/handlers/archive.go (render archive page)
- [ ] T065 [US2] Implement StartArchive handler in internal/web/handlers/archive.go (POST /archive/start, return HTML fragment or JSON)
- [ ] T066 [US2] Implement ArchiveStatus handler in internal/web/handlers/archive.go (GET /archive/status, poll active operation)
- [ ] T067 [US2] Implement Browse handler in internal/web/handlers/browse.go (paginated post list with search support)
- [ ] T068 [US2] Add routes to router in internal/web/router.go for archive and browse handlers

---

## Phase 5: User Story 3 - About Page & External Links (P3)

**Goal**: Users can view project information and navigate to author's Bluesky and GitHub

**Independent Test**:
1. Navigate to `/about` (authenticated or not)
2. See project description
3. Click Bluesky link → opens author's profile in new tab
4. Click GitHub link → opens repository in new tab
5. Verify dark theme consistency

### 5.1: Templates (US3)

- [ ] T069 [US3] Create about page template in internal/web/templates/pages/about.html with project description, author Bluesky link, GitHub repo link

### 5.2: HTTP Handlers (US3)

- [ ] T070 [US3] Implement About handler in internal/web/handlers/about.go (render about template with version, author, repo URL)

### 5.3: Configuration (US3)

- [ ] T071 [US3] Add about section to config.yaml with version, author_bsky_handle, github_repo_url

### 5.4: Router Integration (US3)

- [ ] T072 [US3] Add /about route to router in internal/web/router.go (public route)

### 5.5: Navigation (US3)

- [ ] T073 [US3] Create nav partial in internal/web/templates/partials/nav.html with links to landing, dashboard, archive, browse, about

---

## Phase 6: Polish & Cross-Cutting Concerns

**Goal**: Production-ready quality, error handling, testing

**Independent Test**: All tests pass, error scenarios handled gracefully, production build succeeds

### 6.1: Error Handling

- [ ] T074 [P] Create error page templates in internal/web/templates/pages/: 401.html, 404.html, 500.html
- [ ] T075 Implement error middleware in internal/web/middleware/errors.go (catch panics, render error pages)

**Parallel Execution**: T074 can run while T075 is being implemented

### 6.2: CSRF Protection

- [ ] T076 Implement CSRF middleware in internal/web/middleware/csrf.go (gorilla/csrf integration)
- [ ] T077 Add CSRF token to all forms in templates (landing, archive, browse)

### 6.3: Static Asset Serving

- [ ] T078 Add static file handler to router in internal/web/router.go (serve /static/*)
- [ ] T079 Add media file handler to router in internal/web/router.go (serve /media/*, auth required)

### 6.4: Minimal JavaScript

- [ ] T080 [P] Create app.js in internal/web/static/js/app.js with confirmation dialogs for destructive actions

**Parallel Execution**: T080 can run concurrently with other polish tasks

### 6.5: Testing (Optional - only if TDD requested)

- [ ] T081 Create setupTestDB helper in tests/unit/storage_test.go (in-memory SQLite)
- [ ] T082 Write unit tests for storage layer in tests/unit/storage_test.go (SavePost, GetPost, ListPosts, SearchPosts)
- [ ] T083 Write unit tests for session management in tests/unit/auth_test.go
- [ ] T084 Write contract tests for HTTP handlers in tests/contract/handlers_test.go (landing, dashboard, archive, browse, about)

### 6.6: Documentation

- [ ] T085 [P] Create README.md with project description, installation, usage, development instructions

**Parallel Execution**: T085 can run concurrently with other tasks

---

## Parallel Execution Examples

### Setup Phase (Maximum Parallelism)
```bash
# All directory creation tasks can run simultaneously
T003 & T004 & T005 & T006
```

### Foundation Phase
```bash
# Config and database can be implemented while downloading assets
T007 T008 T009 & T010
```

### US1 Phase (Selected Parallel Tasks)
```bash
# Models
T011 & T012

# Middleware
T023 & T024

# Static Assets
T025 T026 T027 & T028
```

### US2 Phase (Maximum Parallelism for Models)
```bash
# All models can be created simultaneously
T033 & T034 & T035 & T036

# Storage layer functions can be parallelized by file
T040 T041 T042 & T043 & T044 T045 & T046 T047 & T048 T049 T050 & T051

# Templates can be created in parallel
T059 & T060 & T061 & T062
```

### Polish Phase
```bash
# Error templates
T074 & T075

# Documentation
T085 (can run anytime)
```

---

## Implementation Strategy

### MVP (Minimum Viable Product)
**Scope**: Phase 1 + Phase 2 + Phase 3 (US1 only)
**Deliverable**: Working authentication and landing page
**Value**: Users can log in and see their identity confirmed

### Incremental Delivery
1. **Sprint 1**: Setup + Foundation + US1 (T001-T032) → Login works
2. **Sprint 2**: US2 (T033-T068) → Archive operations work
3. **Sprint 3**: US3 + Polish (T069-T085) → Complete feature

### Testing Strategy (If TDD Requested)
- Write storage tests (T081-T082) before implementing storage layer (T040-T051)
- Write auth tests (T083) before implementing auth (T015-T022)
- Write handler tests (T084) before implementing handlers (T029-T070)

---

## Task Format Reference

All tasks follow this format:
```
- [ ] [TaskID] [P] [Story] Description with file path
```

**Legend**:
- `[P]`: Parallelizable (can run concurrently with other [P] tasks)
- `[Story]`: User story label (US1, US2, US3)
- TaskID: Sequential execution order (T001-T085)

---

## Summary

- **Total Tasks**: 85
- **Parallelizable Tasks**: 28 (33%)
- **User Story 1 (P1)**: 22 tasks (T011-T032)
- **User Story 2 (P2)**: 36 tasks (T033-T068)
- **User Story 3 (P3)**: 5 tasks (T069-T073)
- **Setup**: 6 tasks (T001-T006)
- **Foundation**: 4 tasks (T007-T010)
- **Polish**: 12 tasks (T074-T085)

**MVP Scope**: 32 tasks (Setup + Foundation + US1)
**Full Feature**: 85 tasks

**Suggested First Task**: T001 (Initialize Go module)
**Suggested MVP Completion**: T032 (Main application with working auth)
