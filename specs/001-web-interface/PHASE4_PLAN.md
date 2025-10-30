# Phase 4 Implementation Plan
## User Story 2: Archive Management Pages (P2)

**Date**: 2025-10-30
**Status**: Planning
**Tasks**: T033-T068 (36 tasks)

---

## Overview

Phase 4 implements the core archival functionality - the actual purpose of this tool! Users will be able to:
1. Initiate archive operations to fetch their Bluesky posts
2. Monitor progress in real-time with HTMX polling
3. Browse archived posts with pagination
4. Search posts using SQLite FTS5 full-text search

---

## Implementation Order

### Stage 1: Foundation (Data & Storage) - T033-T051
**Goal**: Set up data models, database schema, and storage layer

#### 1.1 Data Models (Parallel: T033-T036)
- ✅ Sessions/Profiles already exist from Phase 3
- **T033**: Post model (URI, CID, text, metrics, timestamps)
- **T034**: Media model (hash, PostURI, paths, dimensions)
- **T035**: ArchiveOperation model (status, progress tracking)
- **T036**: ArchiveStatus model (aggregated statistics)

**Note**: These posts/media tables are ALREADY in db.go from Phase 2! We may just need models.

#### 1.2 Database Migrations (Sequential: T037-T039)
- **CHECK FIRST**: Posts, media, operations tables already exist in db.go
- If needed, verify FTS5 setup and indices
- May skip or just add validation tests

#### 1.3 Storage Layer (Sequential: T040-T051)
- **Posts**: SavePost, GetPost, ListPosts (with pagination)
- **Search**: SearchPosts using FTS5
- **Profiles**: SaveProfile, GetLatestProfile (snapshot history)
- **Media**: SaveMedia (with SHA-256 hash paths), ListMediaForPost
- **Operations**: CreateOperation, UpdateOperation, GetActiveOperation
- **Status**: GetArchiveStatus (aggregated stats)

**Dependencies**: Models → Storage functions

---

### Stage 2: AT Protocol Integration - T052-T056
**Goal**: Connect to Bluesky and fetch data

#### 2.1 AT Protocol Client (T052-T056)
- **T052**: NewATProtoClient wrapper (using indigo SDK)
- **T053**: FetchPosts (paginated getAuthorFeed with cursor)
- **T054**: FetchProfile (getProfile from Bluesky)
- **T055**: DownloadMedia (fetch blobs, SHA-256 hash, save to disk)
- **T056**: Rate limiter (token bucket, 300 req/5min per Bluesky limits)

**Key Considerations**:
- Use indigo SDK: `github.com/bluesky-social/indigo`
- DPoP tokens from bskyoauth session
- Cursor-based pagination for posts
- Content-addressable storage for media (SHA-256 hash)

**Dependencies**: Storage layer → AT Protocol integration

---

### Stage 3: Background Worker - T057-T058
**Goal**: Async archival with progress tracking

#### 3.1 Worker Implementation (T057-T058)
- **T057**: archiveWorker goroutine
  - Context cancellation support
  - Progress updates to database
  - Error handling and retry logic
- **T058**: StartArchive orchestration
  - Check for active operations
  - Create new operation record
  - Launch worker goroutine
  - Return operation ID

**Pattern**:
```go
func StartArchive(ctx context.Context, did string) (operationID string, error) {
    // Check for active operation
    // Create operation record
    // Launch goroutine
    go archiveWorker(ctx, operationID, did)
    return operationID, nil
}
```

**Dependencies**: Storage + AT Protocol → Worker

---

### Stage 4: UI Layer - T059-T068
**Goal**: User interface for archive management and browsing

#### 4.1 Templates (Parallel: T059-T062)
- **T059**: Update dashboard.html (show archive stats)
- **T060**: Create archive.html (start button, status display)
- **T061**: Create browse.html (post list, pagination, search)
- **T062**: Create archive-status.html partial (for HTMX polling)

**HTMX Pattern**:
```html
<!-- Poll for status updates every 2 seconds -->
<div hx-get="/archive/status"
     hx-trigger="every 2s"
     hx-swap="outerHTML">
    <!-- Status content here -->
</div>
```

#### 4.2 HTTP Handlers (Sequential: T063-T068)
- **T063**: Update Dashboard handler (fetch ArchiveStatus)
- **T064**: Archive handler (render archive page)
- **T065**: StartArchive handler (POST /archive/start)
  - Initiate worker
  - Return HTMX fragment or JSON
- **T066**: ArchiveStatus handler (GET /archive/status)
  - Poll active operation
  - Return progress HTML fragment
- **T067**: Browse handler (GET /browse?page=1&q=search)
  - Paginated post list
  - FTS5 search support
- **T068**: Add routes to main.go

**Dependencies**: Templates + Storage → Handlers

---

## Critical Technical Decisions

### 1. Database Schema
**Status**: Already exists in db.go from Phase 2!
- Posts table with FTS5 virtual table
- Media table with foreign keys
- Operations table for tracking
- Indices already defined

**Action**: Verify and test existing schema

### 2. AT Protocol Client
**Library**: `github.com/bluesky-social/indigo`
**Authentication**: Use access token from session
**Key methods**:
- `app.bsky.feed.getAuthorFeed` - fetch posts
- `app.bsky.actor.getProfile` - fetch profile
- `com.atproto.sync.getBlob` - download media

### 3. Rate Limiting
**Limits**: 300 requests per 5 minutes (Bluesky)
**Implementation**: Token bucket algorithm
- Refill rate: 1 token per second
- Bucket size: 300 tokens
- Cost per request: 1 token

### 4. Media Storage
**Pattern**: Content-addressable by SHA-256
```
data/media/
  ab/cd/abcd1234...5678.jpg
```
- First 2 chars = directory level 1
- Next 2 chars = directory level 2
- Prevents too many files in one directory

### 5. Progress Tracking
**Pattern**: Update operation record every N posts
```go
if count % 10 == 0 {
    UpdateOperation(operationID, count, total)
}
```

---

## Testing Strategy

### Unit Tests
- Storage layer functions
- Rate limiter
- Media hash generation

### Integration Tests
- Worker flow (mock AT Protocol)
- HTMX polling behavior
- Search functionality

### Manual Tests
1. Start archive operation
2. Monitor progress in real-time
3. Cancel operation (if implemented)
4. Browse archived posts
5. Search posts by keyword
6. Verify media downloads

---

## Risks & Mitigations

### Risk 1: AT Protocol API changes
**Mitigation**: Use indigo SDK (maintained by Bluesky team)

### Risk 2: Long-running operations
**Mitigation**: Background goroutine with progress updates

### Risk 3: Rate limiting
**Mitigation**: Built-in rate limiter, exponential backoff

### Risk 4: Media storage fills disk
**Mitigation**: Config option for max archive size (already in config.yaml)

### Risk 5: Database locks during writes
**Mitigation**: WAL mode already enabled, batch writes where possible

---

## Success Criteria

Phase 4 is complete when:
1. ✅ User can click "Start Archive" and operation begins
2. ✅ Progress bar updates in real-time via HTMX
3. ✅ Posts are fetched from Bluesky and saved to SQLite
4. ✅ Media files are downloaded and stored locally
5. ✅ User can browse paginated list of posts
6. ✅ User can search posts using full-text search
7. ✅ Dashboard shows archive statistics
8. ✅ All 36 tasks (T033-T068) marked complete
9. ✅ Project builds and runs without errors
10. ✅ Integration test passes

---

## Estimated Effort

**Total Tasks**: 36
**Parallelizable**: ~40% (15 tasks with [P] marker)
**Sequential Dependencies**: 21 tasks

**Breakdown**:
- Data Models: 4 tasks (parallel)
- Migrations: 3 tasks (quick if schema exists)
- Storage: 12 tasks (sequential, but straightforward)
- AT Protocol: 5 tasks (moderate complexity)
- Worker: 2 tasks (moderate complexity)
- Templates: 4 tasks (parallel)
- Handlers: 6 tasks (sequential)

**Critical Path**: Storage → AT Protocol → Worker → Handlers

---

## Next Steps

1. Verify database schema exists (check db.go)
2. Create data models (T033-T036)
3. Implement storage layer (T040-T051)
4. Implement AT Protocol client (T052-T056)
5. Implement background worker (T057-T058)
6. Create templates (T059-T062)
7. Implement handlers (T063-T068)
8. Test end-to-end flow
9. Commit and push

**Ready to begin implementation!** 🚀
