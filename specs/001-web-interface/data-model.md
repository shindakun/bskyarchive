# Data Model: Web Interface

**Feature**: Web Interface (001-web-interface)
**Date**: 2025-10-30
**Status**: Complete

## Overview

This document defines all data structures used by the Bluesky Personal Archive Tool. These models represent:
1. **Archive Data**: Posts, profiles, and media fetched from Bluesky via AT Protocol
2. **Session Management**: User authentication and session state
3. **Operational State**: Archive operations and progress tracking
4. **View Models**: Data structures for rendering HTML templates

All archive data is stored in SQLite with full-text search indexing.

## Core Entities

### Archive Data Models

#### Post

Represents a Bluesky post archived from AT Protocol.

**Fields**:
- `URI` (string): AT Protocol URI (e.g., "at://did:plc:xyz/app.bsky.feed.post/abc123")
- `CID` (string): Content Identifier (IPFS-style hash)
- `DID` (string): Author's Decentralized Identifier
- `Text` (string): Post text content (max 300 chars per AT Protocol)
- `CreatedAt` (time.Time): When the post was originally created on Bluesky
- `IndexedAt` (time.Time): When Bluesky indexed the post
- `HasMedia` (bool): Whether post has embedded media
- `LikeCount` (int): Number of likes
- `RepostCount` (int): Number of reposts
- `ReplyCount` (int): Number of replies
- `IsReply` (bool): Whether this post is a reply to another
- `ReplyParent` (string): URI of parent post if reply
- `EmbedType` (string): Type of embed ("images", "external", "record", null)
- `EmbedData` (JSON): Full embed structure from AT Protocol
- `Labels` (JSON): Content labels/warnings
- `ArchivedAt` (time.Time): When this post was archived locally

**Validation Rules**:
- URI must not be empty and match AT Protocol format
- CID must not be empty
- Text max length: 300 characters
- CreatedAt must be in the past
- Counts must be >= 0
- If IsReply is true, ReplyParent must not be empty

**Relationships**:
- Has many Media (one-to-many)
- Indexed by posts_fts for full-text search

**Go Struct**:
```go
type Post struct {
    URI         string
    CID         string
    DID         string
    Text        string
    CreatedAt   time.Time
    IndexedAt   time.Time
    HasMedia    bool
    LikeCount   int
    RepostCount int
    ReplyCount  int
    IsReply     bool
    ReplyParent string
    EmbedType   string
    EmbedData   json.RawMessage
    Labels      json.RawMessage
    ArchivedAt  time.Time
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

CREATE VIRTUAL TABLE posts_fts USING fts5(
  uri UNINDEXED,
  text,
  content='posts',
  content_rowid='rowid'
);
```

---

#### Profile

Represents a snapshot of a Bluesky user profile at a point in time.

**Fields**:
- `DID` (string): Decentralized Identifier
- `Handle` (string): User's handle (e.g., "user.bsky.social")
- `DisplayName` (string): User's display name
- `Description` (string): Profile bio/description
- `AvatarURL` (string): URL to avatar image
- `BannerURL` (string): URL to banner image
- `FollowersCount` (int): Number of followers
- `FollowsCount` (int): Number of accounts following
- `PostsCount` (int): Total number of posts
- `SnapshotAt` (time.Time): When this snapshot was taken

**Validation Rules**:
- DID must not be empty
- Handle must match Bluesky format (alphanumeric + dots)
- Counts must be >= 0
- SnapshotAt must be in the past

**Relationships**:
- Linked to Posts via DID (one-to-many)
- Multiple snapshots per user over time

**Go Struct**:
```go
type Profile struct {
    DID            string
    Handle         string
    DisplayName    string
    Description    string
    AvatarURL      string
    BannerURL      string
    FollowersCount int
    FollowsCount   int
    PostsCount     int
    SnapshotAt     time.Time
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
  followers_count INTEGER,
  follows_count INTEGER,
  posts_count INTEGER,
  snapshot_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (did, snapshot_at)
);
```

---

#### Media

Represents media files (images, videos) embedded in posts.

**Fields**:
- `ID` (int): Auto-incrementing primary key
- `PostURI` (string): URI of the post this media belongs to
- `MediaURL` (string): Original Bluesky CDN URL
- `LocalPath` (string): Local filesystem path (e.g., "archive/media/2024/10/abc123.jpg")
- `AltText` (string): Alternative text for accessibility
- `MimeType` (string): MIME type (e.g., "image/jpeg", "video/mp4")
- `SizeBytes` (int64): File size in bytes
- `DownloadedAt` (time.Time): When the file was downloaded

**Validation Rules**:
- PostURI must not be empty and reference valid post
- MediaURL must be valid HTTP(S) URL
- LocalPath must not be empty after download
- SizeBytes must be > 0
- MimeType must be valid

**Relationships**:
- Belongs to Post (many-to-one)

**Go Struct**:
```go
type Media struct {
    ID           int
    PostURI      string
    MediaURL     string
    LocalPath    string
    AltText      string
    MimeType     string
    SizeBytes    int64
    DownloadedAt time.Time
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
  mime_type TEXT,
  size_bytes INTEGER,
  downloaded_at TIMESTAMP,
  FOREIGN KEY (post_uri) REFERENCES posts(uri)
);
```

---

### Session & Operational Models

#### 1. UserSession

Represents an authenticated user's session stored in encrypted cookies.

**Fields**:
- `DID` (string): Decentralized Identifier for the Bluesky user
- `Handle` (string): User's Bluesky handle (e.g., "user.bsky.social")
- `DisplayName` (string): User's display name
- `AccessToken` (string): OAuth access token (encrypted in session)
- `RefreshToken` (string): OAuth refresh token (encrypted in session)
- `TokenExpiry` (time.Time): When the access token expires
- `SessionCreatedAt` (time.Time): When the session was created
- `SessionExpiresAt` (time.Time): When the session expires (7 days from last activity)
- `CSRFToken` (string): Per-session CSRF token

**Validation Rules**:
- DID must not be empty
- Handle must match Bluesky handle format (alphanumeric + dots)
- AccessToken must not be empty
- SessionExpiresAt must be <= 7 days from SessionCreatedAt

**State Transitions**:
- **Created**: OAuth callback success → session created with tokens
- **Active**: User makes requests → session expiry extended (rolling window)
- **Expired**: 7 days of inactivity → session invalid → redirect to login
- **Invalidated**: User logs out → session destroyed

**Storage**: Encrypted in gorilla/sessions cookie store

**Go Struct**:
```go
type UserSession struct {
    DID              string
    Handle           string
    DisplayName      string
    AccessToken      string    // encrypted
    RefreshToken     string    // encrypted
    TokenExpiry      time.Time
    SessionCreatedAt time.Time
    SessionExpiresAt time.Time
    CSRFToken        string
}
```

### 2. ArchiveStatus

Represents the current state of a user's archive (read from backend service).

**Fields**:
- `DID` (string): User's DID
- `Handle` (string): User's handle
- `LastSyncTime` (time.Time): Timestamp of last successful sync (nil if never synced)
- `TotalPosts` (int): Total number of archived posts
- `TotalMedia` (int): Total number of archived media files
- `TotalLikes` (int): Total number of liked posts archived
- `TotalReposts` (int): Total number of reposts archived
- `CurrentOperation` (*ArchiveOperation): Current in-progress operation (nil if none)
- `ArchiveSize` (int64): Total size of archive in bytes

**Validation Rules**:
- DID must not be empty
- Counts must be >= 0
- LastSyncTime must be in the past if not nil

**Relationships**:
- Has one CurrentOperation (nullable)

**Go Struct**:
```go
type ArchiveStatus struct {
    DID              string
    Handle           string
    LastSyncTime     *time.Time
    TotalPosts       int
    TotalMedia       int
    TotalLikes       int
    TotalReposts     int
    CurrentOperation *ArchiveOperation
    ArchiveSize      int64
}
```

### 3. ArchiveOperation

Represents an in-progress or completed archival task.

**Fields**:
- `OperationID` (string): Unique identifier for this operation
- `DID` (string): User's DID
- `Type` (string): "full" or "incremental"
- `Status` (string): "queued", "running", "completed", "failed"
- `Progress` (float64): Percentage complete (0.0 to 100.0)
- `StartedAt` (time.Time): When the operation started
- `CompletedAt` (*time.Time): When the operation completed (nil if running)
- `PostsFetched` (int): Number of posts fetched so far
- `MediaDownloaded` (int): Number of media files downloaded so far
- `ErrorMessage` (string): Error details if status is "failed"

**Validation Rules**:
- OperationID must not be empty
- Type must be "full" or "incremental"
- Status must be one of: queued, running, completed, failed
- Progress must be between 0.0 and 100.0
- StartedAt must be in the past
- If status is "completed" or "failed", CompletedAt must not be nil

**State Transitions**:
- **Queued**: User initiates sync → operation created
- **Running**: Backend starts processing → status updated
- **Completed**: All data fetched successfully → status set to completed
- **Failed**: Error occurs → status set to failed with error message

**Go Struct**:
```go
type ArchiveOperation struct {
    OperationID     string
    DID             string
    Type            string // "full" or "incremental"
    Status          string // "queued", "running", "completed", "failed"
    Progress        float64
    StartedAt       time.Time
    CompletedAt     *time.Time
    PostsFetched    int
    MediaDownloaded int
    ErrorMessage    string
}
```

### 4. PostSummary

Lightweight representation of an archived post for browse/list views (full post data comes from backend).

**Fields**:
- `URI` (string): AT Protocol URI for the post
- `CID` (string): Content Identifier
- `Text` (string): Post text content
- `CreatedAt` (time.Time): When the post was created
- `AuthorDID` (string): Author's DID
- `AuthorHandle` (string): Author's handle
- `AuthorDisplayName` (string): Author's display name
- `HasMedia` (bool): Whether post has embedded media
- `MediaCount` (int): Number of media items
- `LikeCount` (int): Number of likes
- `RepostCount` (int): Number of reposts
- `ReplyCount` (int): Number of replies
- `IsReply` (bool): Whether this is a reply to another post
- `IsRepost` (bool): Whether this is a repost

**Validation Rules**:
- URI must not be empty
- CID must not be empty
- Text max length: 300 characters (AT Protocol limit)
- CreatedAt must be in the past
- Counts must be >= 0

**Go Struct**:
```go
type PostSummary struct {
    URI               string
    CID               string
    Text              string
    CreatedAt         time.Time
    AuthorDID         string
    AuthorHandle      string
    AuthorDisplayName string
    HasMedia          bool
    MediaCount        int
    LikeCount         int
    RepostCount       int
    ReplyCount        int
    IsReply           bool
    IsRepost          bool
}
```

### 5. PostPage

Paginated collection of posts for browse interface.

**Fields**:
- `Posts` ([]PostSummary): List of posts for current page
- `CurrentPage` (int): Current page number (1-indexed)
- `PageSize` (int): Number of posts per page
- `TotalPosts` (int): Total number of posts in archive
- `TotalPages` (int): Total number of pages
- `HasPrevious` (bool): Whether there's a previous page
- `HasNext` (bool): Whether there's a next page

**Validation Rules**:
- CurrentPage must be >= 1
- PageSize must be > 0
- TotalPosts must be >= 0
- TotalPages must be >= 0
- If CurrentPage > 1, HasPrevious should be true
- If CurrentPage < TotalPages, HasNext should be true

**Go Struct**:
```go
type PostPage struct {
    Posts       []PostSummary
    CurrentPage int
    PageSize    int
    TotalPosts  int
    TotalPages  int
    HasPrevious bool
    HasNext     bool
}
```

## View Models

View models are structs passed to templates for rendering. They combine domain data with UI-specific information.

### 6. LandingPageData

**Fields**:
- `IsAuthenticated` (bool): Whether user has active session
- `CSRFToken` (string): CSRF token for login form

**Go Struct**:
```go
type LandingPageData struct {
    IsAuthenticated bool
    CSRFToken       string
}
```

### 7. DashboardPageData

**Fields**:
- `User` (UserSession): Current user session
- `ArchiveStatus` (ArchiveStatus): User's archive status
- `RecentPosts` ([]PostSummary): 5 most recent posts
- `CSRFToken` (string): CSRF token for forms

**Go Struct**:
```go
type DashboardPageData struct {
    User          UserSession
    ArchiveStatus ArchiveStatus
    RecentPosts   []PostSummary
    CSRFToken     string
}
```

### 8. ArchivePageData

**Fields**:
- `User` (UserSession): Current user session
- `ArchiveStatus` (ArchiveStatus): User's archive status
- `CanStartSync` (bool): Whether user can start a new sync (no operation running)
- `CSRFToken` (string): CSRF token for sync forms

**Go Struct**:
```go
type ArchivePageData struct {
    User          UserSession
    ArchiveStatus ArchiveStatus
    CanStartSync  bool
    CSRFToken     string
}
```

### 9. BrowsePageData

**Fields**:
- `User` (UserSession): Current user session
- `PostPage` (PostPage): Paginated posts
- `CSRFToken` (string): CSRF token (for future filtering forms)

**Go Struct**:
```go
type BrowsePageData struct {
    User      UserSession
    PostPage  PostPage
    CSRFToken string
}
```

### 10. AboutPageData

**Fields**:
- `IsAuthenticated` (bool): Whether user has active session
- `User` (*UserSession): Current user session (nullable)
- `ProjectName` (string): "Bluesky Personal Archive Tool"
- `Version` (string): Application version
- `GitHubURL` (string): Link to GitHub repository
- `BlueskyHandle` (string): Author's Bluesky handle
- `BlueskyURL` (string): Link to author's Bluesky profile

**Go Struct**:
```go
type AboutPageData struct {
    IsAuthenticated bool
    User            *UserSession
    ProjectName     string
    Version         string
    GitHubURL       string
    BlueskyHandle   string
    BlueskyURL      string
}
```

## Storage Layer Interface

The storage layer provides methods for persisting and retrieving data from SQLite. All methods are implemented in `internal/storage/`.

### Storage Interface

**Methods**:

```go
type Store interface {
    // Posts
    SavePost(ctx context.Context, post *Post) error
    GetPost(ctx context.Context, uri string) (*Post, error)
    ListPosts(ctx context.Context, did string, page, pageSize int) (*PostPage, error)
    SearchPosts(ctx context.Context, query string, page, pageSize int) (*PostPage, error)

    // Profiles
    SaveProfile(ctx context.Context, profile *Profile) error
    GetLatestProfile(ctx context.Context, did string) (*Profile, error)

    // Media
    SaveMedia(ctx context.Context, media *Media) error
    ListMediaForPost(ctx context.Context, postURI string) ([]Media, error)

    // Operations
    CreateOperation(ctx context.Context, op *ArchiveOperation) error
    UpdateOperation(ctx context.Context, op *ArchiveOperation) error
    GetOperation(ctx context.Context, operationID string) (*ArchiveOperation, error)
    GetActiveOperation(ctx context.Context, did string) (*ArchiveOperation, error)

    // Stats
    GetArchiveStatus(ctx context.Context, did string) (*ArchiveStatus, error)
}
```

**Error Cases**:
- `ErrNotFound`: Resource not found
- `ErrDuplicate`: Attempt to insert duplicate primary key
- `ErrOperationInProgress`: Cannot start new operation while one is running
- `ErrInvalidInput`: Invalid parameters (e.g., negative page number)

## Enumerations

### SessionStatus

```go
const (
    SessionStatusActive    = "active"
    SessionStatusExpired   = "expired"
    SessionStatusInvalidated = "invalidated"
)
```

### OperationType

```go
const (
    OperationTypeFull        = "full"
    OperationTypeIncremental = "incremental"
)
```

### OperationStatus

```go
const (
    OperationStatusQueued    = "queued"
    OperationStatusRunning   = "running"
    OperationStatusCompleted = "completed"
    OperationStatusFailed    = "failed"
)
```

## Database Schema (If Needed)

The web interface layer primarily uses session cookies for state. However, if we need to persist session data server-side (e.g., for multiple devices), here's a schema:

### sessions table

```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    did TEXT NOT NULL,
    handle TEXT NOT NULL,
    display_name TEXT,
    access_token TEXT NOT NULL,      -- encrypted
    refresh_token TEXT NOT NULL,     -- encrypted
    token_expiry TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    csrf_token TEXT NOT NULL,
    FOREIGN KEY (did) REFERENCES archives(did)
);

CREATE INDEX idx_sessions_did ON sessions(did);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

**Note**: For MVP, we'll use encrypted cookies only. Server-side session storage is a future enhancement if multi-device support is needed.

## Validation Summary

All entities have clear validation rules:
- **UserSession**: Non-empty DID/Handle/Token, valid expiration dates
- **ArchiveStatus**: Non-negative counts, valid DID
- **ArchiveOperation**: Valid status enum, progress 0-100, operation ID present
- **PostSummary**: Non-empty URI/CID, valid timestamps, non-negative counts
- **PostPage**: Valid pagination parameters (page >= 1, size > 0)

All view models include CSRF tokens for form protection.

## Summary

Data models defined for:
1. **Archive Data**: Post, Profile, Media (AT Protocol entities stored in SQLite)
2. **Session Management**: UserSession with 7-day expiration
3. **Operational State**: ArchiveOperation for tracking sync progress
4. **Archive Display**: ArchiveStatus for UI dashboard
5. **Content Browsing**: PostSummary, PostPage for paginated lists
6. **Template Rendering**: View models for each page type
7. **Storage Layer**: Store interface for database operations

All models support the functional requirements from spec.md and align with constitution principles:
- **Privacy**: All data stored locally in SQLite
- **Comprehensive Archival**: Post, Profile, Media models capture complete user history
- **Efficient Search**: FTS5 indexing for <100ms search queries
- **Clear Data Structures**: Well-defined schemas with validation rules

Ready to proceed to API contracts.
