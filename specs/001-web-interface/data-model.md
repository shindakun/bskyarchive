# Data Model: Web Interface

**Phase 1 Output** | **Date**: 2025-10-30 | **Plan**: [plan.md](./plan.md)

## Overview

This document defines all data structures for the Bluesky Personal Archive Tool. It includes models for archive data (posts, profiles, media), user sessions, and operational tracking. Each model includes Go struct definitions, SQL schemas, validation rules, and state transitions where applicable.

---

## 1. Session & Authentication

### Session Model

Represents an authenticated user's session with OAuth tokens and identity information.

**Go Struct**:
```go
type Session struct {
    ID           string    `json:"id"`
    DID          string    `json:"did"`           // Decentralized Identifier
    Handle       string    `json:"handle"`        // Bluesky handle (e.g., "user.bsky.social")
    DisplayName  string    `json:"display_name"`  // Optional display name
    AccessToken  string    `json:"-"`             // Never serialize to JSON
    RefreshToken string    `json:"-"`             // Never serialize to JSON
    ExpiresAt    time.Time `json:"expires_at"`
    CreatedAt    time.Time `json:"created_at"`
}
```

**SQL Schema** (optional, if using database-backed sessions):
```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    handle TEXT NOT NULL,
    display_name TEXT,
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sessions_did ON sessions(did);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

**Validation Rules**:
- `id`: Required, UUID format
- `did`: Required, must start with "did:plc:" or "did:web:"
- `handle`: Required, valid Bluesky handle format
- `access_token`: Required, non-empty
- `expires_at`: Required, must be future timestamp
- `created_at`: Auto-generated on creation

**Session States**:
- **Active**: Current timestamp < expires_at
- **Expired**: Current timestamp >= expires_at
- **Revoked**: Manually deleted/cleared by user

**State Transitions**:
```
[Created] → [Active] → [Expired]
         ↓
      [Revoked]
```

---

## 2. Archive Data

### Post Model

Represents a single Bluesky post with all metadata, engagement metrics, and relationships.

**Go Struct**:
```go
type Post struct {
    URI         string          `json:"uri" db:"uri"`
    CID         string          `json:"cid" db:"cid"`
    DID         string          `json:"did" db:"did"`
    Text        string          `json:"text" db:"text"`
    CreatedAt   time.Time       `json:"created_at" db:"created_at"`
    IndexedAt   time.Time       `json:"indexed_at" db:"indexed_at"`
    HasMedia    bool            `json:"has_media" db:"has_media"`
    LikeCount   int             `json:"like_count" db:"like_count"`
    RepostCount int             `json:"repost_count" db:"repost_count"`
    ReplyCount  int             `json:"reply_count" db:"reply_count"`
    IsReply     bool            `json:"is_reply" db:"is_reply"`
    ReplyParent string          `json:"reply_parent,omitempty" db:"reply_parent"`
    EmbedType   string          `json:"embed_type,omitempty" db:"embed_type"`
    EmbedData   json.RawMessage `json:"embed_data,omitempty" db:"embed_data"`
    Labels      json.RawMessage `json:"labels,omitempty" db:"labels"`
    ArchivedAt  time.Time       `json:"archived_at" db:"archived_at"`
}
```

**SQL Schema**:
```sql
CREATE TABLE posts (
    uri TEXT PRIMARY KEY,
    cid TEXT NOT NULL,
    did TEXT NOT NULL,
    text TEXT,
    created_at TIMESTAMP NOT NULL,
    indexed_at TIMESTAMP NOT NULL,
    has_media BOOLEAN DEFAULT 0,
    like_count INTEGER DEFAULT 0,
    repost_count INTEGER DEFAULT 0,
    reply_count INTEGER DEFAULT 0,
    is_reply BOOLEAN DEFAULT 0,
    reply_parent TEXT,
    embed_type TEXT,
    embed_data JSON,
    labels JSON,
    archived_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_posts_did ON posts(did);
CREATE INDEX idx_posts_created_at ON posts(created_at DESC);
CREATE INDEX idx_posts_has_media ON posts(has_media);
CREATE INDEX idx_posts_is_reply ON posts(is_reply);

-- Full-text search
CREATE VIRTUAL TABLE posts_fts USING fts5(
    uri UNINDEXED,
    text,
    content='posts',
    content_rowid='rowid'
);

-- Triggers to keep FTS in sync
CREATE TRIGGER posts_ai AFTER INSERT ON posts BEGIN
    INSERT INTO posts_fts(rowid, uri, text)
    VALUES (new.rowid, new.uri, new.text);
END;

CREATE TRIGGER posts_ad AFTER DELETE ON posts BEGIN
    INSERT INTO posts_fts(posts_fts, rowid, uri, text)
    VALUES('delete', old.rowid, old.uri, old.text);
END;

CREATE TRIGGER posts_au AFTER UPDATE ON posts BEGIN
    INSERT INTO posts_fts(posts_fts, rowid, uri, text)
    VALUES('delete', old.rowid, old.uri, old.text);
    INSERT INTO posts_fts(rowid, uri, text)
    VALUES (new.rowid, new.uri, new.text);
END;
```

**Validation Rules**:
- `uri`: Required, unique, format: "at://did:plc:xxx/app.bsky.feed.post/xxx"
- `cid`: Required, Content Identifier from AT Protocol
- `did`: Required, must match session DID for user's own archive
- `text`: Optional, max 300 characters (Bluesky limit)
- `created_at`: Required, original post timestamp
- `indexed_at`: Required, when AT Protocol indexed the post
- `embed_type`: Optional, one of: "images", "external", "record", "record_with_media"
- `embed_data`: Optional, JSON blob matching embed_type schema
- `labels`: Optional, JSON array of content labels

**Embed Types & Data Structures**:

```go
// Images embed
type ImagesEmbed struct {
    Images []Image `json:"images"`
}

type Image struct {
    Alt      string `json:"alt"`
    Image    Blob   `json:"image"`
    AspectRatio *AspectRatio `json:"aspectRatio,omitempty"`
}

type Blob struct {
    Type     string `json:"$type"`
    Ref      string `json:"ref"`
    MimeType string `json:"mimeType"`
    Size     int    `json:"size"`
}

type AspectRatio struct {
    Width  int `json:"width"`
    Height int `json:"height"`
}

// External link embed
type ExternalEmbed struct {
    External External `json:"external"`
}

type External struct {
    URI         string `json:"uri"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Thumb       *Blob  `json:"thumb,omitempty"`
}

// Record embed (quote post)
type RecordEmbed struct {
    Record RecordRef `json:"record"`
}

type RecordRef struct {
    URI string `json:"uri"`
    CID string `json:"cid"`
}
```

---

### Profile Model

Represents a snapshot of a user's Bluesky profile at a specific point in time.

**Go Struct**:
```go
type Profile struct {
    DID            string    `json:"did" db:"did"`
    Handle         string    `json:"handle" db:"handle"`
    DisplayName    string    `json:"display_name" db:"display_name"`
    Description    string    `json:"description" db:"description"`
    AvatarURL      string    `json:"avatar_url" db:"avatar_url"`
    BannerURL      string    `json:"banner_url" db:"banner_url"`
    FollowersCount int       `json:"followers_count" db:"followers_count"`
    FollowsCount   int       `json:"follows_count" db:"follows_count"`
    PostsCount     int       `json:"posts_count" db:"posts_count"`
    SnapshotAt     time.Time `json:"snapshot_at" db:"snapshot_at"`
}
```

**SQL Schema**:
```sql
CREATE TABLE profiles (
    did TEXT NOT NULL,
    handle TEXT NOT NULL,
    display_name TEXT,
    description TEXT,
    avatar_url TEXT,
    banner_url TEXT,
    followers_count INTEGER DEFAULT 0,
    follows_count INTEGER DEFAULT 0,
    posts_count INTEGER DEFAULT 0,
    snapshot_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (did, snapshot_at)
);

CREATE INDEX idx_profiles_did ON profiles(did);
CREATE INDEX idx_profiles_snapshot_at ON profiles(snapshot_at DESC);
```

**Validation Rules**:
- `did`: Required, unique per snapshot
- `handle`: Required, valid Bluesky handle
- `display_name`: Optional, max 64 characters
- `description`: Optional, max 256 characters
- `avatar_url`: Optional, valid URL
- `banner_url`: Optional, valid URL
- Counts: Non-negative integers

**Usage Pattern**:
- Snapshot taken at start of each archive operation
- Historical tracking of profile changes over time
- Latest snapshot: `SELECT * FROM profiles WHERE did = ? ORDER BY snapshot_at DESC LIMIT 1`

---

### Media Model

Represents media (images, videos) embedded in posts, with local storage information.

**Go Struct**:
```go
type Media struct {
    ID           int       `json:"id" db:"id"`
    PostURI      string    `json:"post_uri" db:"post_uri"`
    MediaURL     string    `json:"media_url" db:"media_url"`
    LocalPath    string    `json:"local_path" db:"local_path"`
    AltText      string    `json:"alt_text" db:"alt_text"`
    MimeType     string    `json:"mime_type" db:"mime_type"`
    SizeBytes    int64     `json:"size_bytes" db:"size_bytes"`
    Width        int       `json:"width" db:"width"`
    Height       int       `json:"height" db:"height"`
    DownloadedAt time.Time `json:"downloaded_at" db:"downloaded_at"`
}
```

**SQL Schema**:
```sql
CREATE TABLE media (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    post_uri TEXT NOT NULL,
    media_url TEXT NOT NULL,
    local_path TEXT NOT NULL,
    alt_text TEXT,
    mime_type TEXT NOT NULL,
    size_bytes INTEGER NOT NULL,
    width INTEGER DEFAULT 0,
    height INTEGER DEFAULT 0,
    downloaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (post_uri) REFERENCES posts(uri) ON DELETE CASCADE
);

CREATE INDEX idx_media_post_uri ON media(post_uri);
CREATE INDEX idx_media_mime_type ON media(mime_type);
CREATE UNIQUE INDEX idx_media_local_path ON media(local_path);
```

**Validation Rules**:
- `post_uri`: Required, must reference existing post
- `media_url`: Required, valid HTTPS URL
- `local_path`: Required, unique, format: "archive/media/YYYY/MM/hash.ext"
- `mime_type`: Required, one of: "image/jpeg", "image/png", "image/gif", "image/webp", "video/mp4"
- `size_bytes`: Required, positive integer
- `width`, `height`: Optional, positive integers for images
- `alt_text`: Optional, accessibility description

**Local Path Format**:
```
archive/media/YYYY/MM/[content-hash].[ext]

Example:
archive/media/2025/10/abc123def456.jpg
```

**Content Hash**: First 16 characters of SHA-256 hash of file content

---

## 3. Operational Tracking

### ArchiveOperation Model

Tracks long-running archive operations with progress and status.

**Go Struct**:
```go
type ArchiveOperation struct {
    ID              string     `json:"id" db:"id"`
    DID             string     `json:"did" db:"did"`
    OperationType   string     `json:"operation_type" db:"operation_type"`
    Status          string     `json:"status" db:"status"`
    ProgressCurrent int        `json:"progress_current" db:"progress_current"`
    ProgressTotal   int        `json:"progress_total" db:"progress_total"`
    StartedAt       time.Time  `json:"started_at" db:"started_at"`
    CompletedAt     *time.Time `json:"completed_at,omitempty" db:"completed_at"`
    ErrorMessage    string     `json:"error_message,omitempty" db:"error_message"`
}
```

**SQL Schema**:
```sql
CREATE TABLE archive_operations (
    id TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    status TEXT NOT NULL,
    progress_current INTEGER DEFAULT 0,
    progress_total INTEGER DEFAULT 0,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT
);

CREATE INDEX idx_operations_did ON archive_operations(did);
CREATE INDEX idx_operations_status ON archive_operations(status);
CREATE INDEX idx_operations_started_at ON archive_operations(started_at DESC);
```

**Validation Rules**:
- `id`: Required, UUID format
- `did`: Required, must match authenticated user
- `operation_type`: Required, one of: "full_sync", "incremental_sync"
- `status`: Required, one of: "running", "completed", "failed", "cancelled"
- `progress_current`: Non-negative integer, <= progress_total
- `progress_total`: Non-negative integer (0 if unknown)
- `started_at`: Auto-generated on creation
- `completed_at`: Set when status changes to completed/failed/cancelled
- `error_message`: Only set when status = "failed"

**Operation Types**:

1. **full_sync**:
   - First-time archive or complete re-sync
   - Fetches all posts from the beginning
   - Downloads all media

2. **incremental_sync**:
   - Updates existing archive
   - Only fetches posts newer than last sync
   - Downloads only new media

**Status States & Transitions**:

```
[Created: running] → [Completed: completed]
                  → [Failed: failed]
                  → [Cancelled: cancelled]
```

**State Definitions**:
- **running**: Operation in progress, worker actively fetching data
- **completed**: Operation finished successfully, all data archived
- **failed**: Operation encountered unrecoverable error
- **cancelled**: User or system cancelled operation

**Progress Tracking**:
- `progress_current`: Number of posts processed so far
- `progress_total`: Total posts to process (0 if unknown during full sync)
- Percentage: `(progress_current / progress_total) * 100` when total > 0
- Indeterminate: Show spinner when total = 0

---

### ArchiveStatus Model

Aggregated view of archive state for a user (derived from other tables).

**Go Struct**:
```go
type ArchiveStatus struct {
    DID               string     `json:"did"`
    TotalPosts        int        `json:"total_posts"`
    TotalMedia        int        `json:"total_media"`
    LastSyncAt        *time.Time `json:"last_sync_at,omitempty"`
    LastSyncType      string     `json:"last_sync_type,omitempty"`
    HasActiveOperation bool       `json:"has_active_operation"`
    ActiveOperation   *ArchiveOperation `json:"active_operation,omitempty"`
}
```

**Derived Query** (no dedicated table):
```go
func GetArchiveStatus(ctx context.Context, db *sql.DB, did string) (*ArchiveStatus, error) {
    var status ArchiveStatus
    status.DID = did

    // Count posts
    db.QueryRow("SELECT COUNT(*) FROM posts WHERE did = ?", did).Scan(&status.TotalPosts)

    // Count media
    db.QueryRow(`
        SELECT COUNT(*) FROM media
        WHERE post_uri IN (SELECT uri FROM posts WHERE did = ?)
    `, did).Scan(&status.TotalMedia)

    // Get last completed operation
    var lastSync time.Time
    var lastType string
    err := db.QueryRow(`
        SELECT completed_at, operation_type
        FROM archive_operations
        WHERE did = ? AND status = 'completed'
        ORDER BY completed_at DESC LIMIT 1
    `, did).Scan(&lastSync, &lastType)
    if err == nil {
        status.LastSyncAt = &lastSync
        status.LastSyncType = lastType
    }

    // Check for active operation
    var opID string
    err = db.QueryRow(`
        SELECT id FROM archive_operations
        WHERE did = ? AND status = 'running'
    `, did).Scan(&opID)
    if err == nil {
        status.HasActiveOperation = true
        // Fetch full operation details
        status.ActiveOperation, _ = GetOperation(ctx, db, opID)
    }

    return &status, nil
}
```

**Validation Rules**: N/A (read-only derived data)

---

## 4. API Response Wrappers

### PagedPostsResponse

Used for paginated post listings in browse interface.

**Go Struct**:
```go
type PagedPostsResponse struct {
    Posts      []Post `json:"posts"`
    Page       int    `json:"page"`
    PageSize   int    `json:"page_size"`
    TotalCount int    `json:"total_count"`
    TotalPages int    `json:"total_pages"`
    HasNext    bool   `json:"has_next"`
    HasPrev    bool   `json:"has_prev"`
}
```

**Usage**:
```go
func ListPosts(ctx context.Context, db *sql.DB, did string, page, pageSize int) (*PagedPostsResponse, error) {
    // Count total
    var totalCount int
    db.QueryRow("SELECT COUNT(*) FROM posts WHERE did = ?", did).Scan(&totalCount)

    // Calculate pagination
    totalPages := (totalCount + pageSize - 1) / pageSize
    offset := (page - 1) * pageSize

    // Fetch posts
    rows, _ := db.Query(`
        SELECT * FROM posts
        WHERE did = ?
        ORDER BY created_at DESC
        LIMIT ? OFFSET ?
    `, did, pageSize, offset)
    // ... scan posts

    return &PagedPostsResponse{
        Posts:      posts,
        Page:       page,
        PageSize:   pageSize,
        TotalCount: totalCount,
        TotalPages: totalPages,
        HasNext:    page < totalPages,
        HasPrev:    page > 1,
    }, nil
}
```

---

### SearchPostsResponse

Used for full-text search results.

**Go Struct**:
```go
type SearchPostsResponse struct {
    Posts      []Post `json:"posts"`
    Query      string `json:"query"`
    Page       int    `json:"page"`
    PageSize   int    `json:"page_size"`
    TotalCount int    `json:"total_count"`
    HasMore    bool   `json:"has_more"`
}
```

**Usage**:
```go
func SearchPosts(ctx context.Context, db *sql.DB, query string, page, pageSize int) (*SearchPostsResponse, error) {
    // Count matches
    var totalCount int
    db.QueryRow(`
        SELECT COUNT(*) FROM posts p
        JOIN posts_fts fts ON p.rowid = fts.rowid
        WHERE posts_fts MATCH ?
    `, query).Scan(&totalCount)

    // Fetch results
    offset := (page - 1) * pageSize
    rows, _ := db.Query(`
        SELECT p.* FROM posts p
        JOIN posts_fts fts ON p.rowid = fts.rowid
        WHERE posts_fts MATCH ?
        ORDER BY p.created_at DESC
        LIMIT ? OFFSET ?
    `, query, pageSize, offset)
    // ... scan posts

    return &SearchPostsResponse{
        Posts:      posts,
        Query:      query,
        Page:       page,
        PageSize:   pageSize,
        TotalCount: totalCount,
        HasMore:    (page * pageSize) < totalCount,
    }, nil
}
```

---

## 5. Storage Layer Interface

Defines the contract for data persistence operations.

**Go Interface**:
```go
type Store interface {
    // Posts
    SavePost(ctx context.Context, post *Post) error
    GetPost(ctx context.Context, uri string) (*Post, error)
    ListPosts(ctx context.Context, did string, page, pageSize int) (*PagedPostsResponse, error)
    SearchPosts(ctx context.Context, query string, page, pageSize int) (*SearchPostsResponse, error)
    DeletePost(ctx context.Context, uri string) error

    // Profiles
    SaveProfile(ctx context.Context, profile *Profile) error
    GetLatestProfile(ctx context.Context, did string) (*Profile, error)
    GetProfileHistory(ctx context.Context, did string) ([]Profile, error)

    // Media
    SaveMedia(ctx context.Context, media *Media) error
    GetMedia(ctx context.Context, id int) (*Media, error)
    ListMediaForPost(ctx context.Context, postURI string) ([]Media, error)
    DeleteMedia(ctx context.Context, id int) error

    // Operations
    CreateOperation(ctx context.Context, op *ArchiveOperation) error
    UpdateOperation(ctx context.Context, op *ArchiveOperation) error
    GetOperation(ctx context.Context, operationID string) (*ArchiveOperation, error)
    GetActiveOperation(ctx context.Context, did string) (*ArchiveOperation, error)
    ListOperations(ctx context.Context, did string, limit int) ([]ArchiveOperation, error)

    // Aggregates
    GetArchiveStatus(ctx context.Context, did string) (*ArchiveStatus, error)

    // Database Management
    Close() error
    Ping(ctx context.Context) error
}
```

**Implementation**: See [internal/storage/](../../internal/storage/) for SQLite implementation.

---

## Summary

This data model supports the complete lifecycle of Bluesky archive operations:

1. **Authentication Flow**: Session model with OAuth tokens
2. **Data Collection**: Post, Profile, and Media models with full AT Protocol metadata
3. **Operation Tracking**: ArchiveOperation for async background jobs
4. **User Interface**: ArchiveStatus, PagedPostsResponse, SearchPostsResponse for web display
5. **Storage Contract**: Store interface for persistence operations

All models include:
- Go struct definitions with JSON/DB tags
- SQL schemas with proper indexes and constraints
- Validation rules
- State transitions where applicable
- Full-text search integration for posts

The design prioritizes:
- **Data integrity**: Foreign keys, constraints, unique indexes
- **Performance**: Strategic indexes for common queries
- **Search**: FTS5 integration with automatic triggers
- **Scalability**: Pagination support, efficient queries
- **Local-first**: Single SQLite database, no external dependencies
