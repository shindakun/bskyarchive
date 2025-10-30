# Bluesky Personal Archive Tool - Design Document

## Project Overview

A comprehensive personal archival tool for Bluesky that periodically backs up your posts, interactions, and profile data. The tool provides searchable archives, multiple export formats, and a beautiful web interface to browse your social media history.

## Core Features

### 1. Authentication & Authorization
- OAuth 2.0 flow using bskyoauth library
- Secure session management
- Support for multiple accounts
- Token refresh handling

### 2. Data Collection
- **Posts Archive**
  - All your posts with timestamps
  - Embedded media (images, videos)
  - Thread context (replies, quote posts)
  - Engagement metrics (likes, reposts, replies)
  
- **Interactions Archive**
  - Your likes history
  - Your reposts
  - Mentions you received
  - Conversations/threads you participated in

- **Profile Data**
  - Profile information snapshots over time
  - Follower/following lists with timestamps
  - Bio and display name history

- **Media Backup**
  - Download and store images locally
  - Video archival (if applicable)
  - Alt text preservation

### 3. Storage Backend

```
archive/
├── config/
│   ├── accounts.json          # Account configurations
│   └── settings.json          # App settings
├── data/
│   ├── {did}/                 # Per-account directories
│   │   ├── posts/
│   │   │   ├── 2024/
│   │   │   │   ├── 01.json   # Monthly post archives
│   │   │   │   └── 02.json
│   │   │   └── index.db      # SQLite index for fast search
│   │   ├── media/
│   │   │   ├── images/
│   │   │   └── videos/
│   │   ├── profile/
│   │   │   └── snapshots.json
│   │   ├── interactions/
│   │   │   ├── likes.json
│   │   │   └── reposts.json
│   │   └── metadata.json      # Archive metadata
├── exports/                   # Generated exports
└── static/                    # Web UI assets
```

### 4. Export Formats

- **JSON** - Complete structured data
- **Markdown** - Readable text format with frontmatter
- **HTML** - Static website with search
- **CSV** - Spreadsheet-compatible format
- **PDF** - Printable archive (optional)

### 5. Search & Browse

- Full-text search across all posts
- Filter by date range
- Filter by engagement metrics
- Filter by media presence
- Tag-based organization
- Thread reconstruction

### 6. Web Interface

- Dashboard showing archive statistics
- Timeline view of posts
- Calendar view by posting date
- Search interface
- Export management
- Settings & configuration

### 7. Scheduling & Automation

- Configurable backup intervals (hourly, daily, weekly)
- Incremental backups (only fetch new content)
- Background job scheduler
- Email/webhook notifications on completion

## Technical Architecture

### Components

```
bluesky-archiver/
├── cmd/
│   ├── archiver/              # Main CLI application
│   └── server/                # Web server
├── internal/
│   ├── auth/                  # OAuth handling
│   ├── collector/             # Data fetching from Bluesky
│   ├── storage/               # File & database operations
│   ├── search/                # Search indexing & queries
│   ├── exporter/              # Export format handlers
│   ├── scheduler/             # Background job management
│   └── web/                   # HTTP handlers & templates
├── pkg/
│   ├── models/                # Data models
│   └── config/                # Configuration management
├── web/
│   ├── templates/             # HTML templates
│   ├── static/                # CSS, JS, images
│   └── assets/                # Build assets
├── migrations/                # Database migrations
└── scripts/                   # Utility scripts
```

### Technology Stack

- **Language**: Go 1.21+
- **OAuth**: github.com/shindakun/bskyoauth
- **Database**: SQLite (for indexing), JSON files (for raw data)
- **Web Framework**: net/http (stdlib) or Echo/Gin
- **Search**: bleve (full-text search)
- **Templating**: html/template
- **Scheduling**: robfig/cron
- **CLI**: spf13/cobra
- **Frontend**: Vanilla JS, Picocss, and HTMX(lightweight)

## Data Models

### Post Model
```go
type Post struct {
    URI       string            `json:"uri"`
    CID       string            `json:"cid"`
    DID       string            `json:"did"`
    Text      string            `json:"text"`
    CreatedAt time.Time         `json:"createdAt"`
    Langs     []string          `json:"langs,omitempty"`
    
    // Embeds
    Images    []Image           `json:"images,omitempty"`
    External  *ExternalEmbed    `json:"external,omitempty"`
    
    // Engagement
    LikeCount   int             `json:"likeCount"`
    RepostCount int             `json:"repostCount"`
    ReplyCount  int             `json:"replyCount"`
    
    // Thread context
    ReplyTo     *PostReference  `json:"replyTo,omitempty"`
    QuotePost   *PostReference  `json:"quotePost,omitempty"`
    
    // Archive metadata
    ArchivedAt  time.Time       `json:"archivedAt"`
    LocalMedia  []string        `json:"localMedia,omitempty"`
}
```

### Archive Metadata
```go
type ArchiveMetadata struct {
    DID              string    `json:"did"`
    Handle           string    `json:"handle"`
    FirstArchiveDate time.Time `json:"firstArchiveDate"`
    LastUpdateDate   time.Time `json:"lastUpdateDate"`
    TotalPosts       int       `json:"totalPosts"`
    TotalMedia       int       `json:"totalMedia"`
    ArchiveVersion   string    `json:"archiveVersion"`
}
```

## API Endpoints (Web UI)

```
GET  /                          # Dashboard
GET  /timeline                  # Browse posts
GET  /search                    # Search interface
POST /search/query              # Search API
GET  /posts/{uri}               # Single post view
GET  /stats                     # Statistics & charts
GET  /export                    # Export management
POST /export/generate           # Generate new export
GET  /settings                  # Configuration
POST /archive/sync              # Trigger manual sync
GET  /api/posts                 # JSON API for posts
```

## CLI Commands

```bash
# Initialize and authenticate
archiver init
archiver auth add

# Sync operations
archiver sync                    # One-time sync
archiver sync --incremental      # Fetch only new posts
archiver sync --full             # Complete re-sync

# Export operations
archiver export --format=json
archiver export --format=markdown --output=/path/to/dir
archiver export --format=html --output=/path/to/site

# Search from CLI
archiver search "query text"
archiver search --from=2024-01-01 --to=2024-12-31

# Server management
archiver serve --port=8080       # Start web server
archiver daemon --interval=1h    # Background sync daemon

# Utilities
archiver stats                   # Show archive statistics
archiver verify                  # Verify archive integrity
archiver cleanup                 # Remove orphaned files
```

## Implementation Phases

### Phase 1: Core Foundation (Week 1-2)
- [ ] Project structure setup
- [ ] OAuth integration with bskyoauth
- [ ] Basic post fetching
- [ ] JSON storage implementation
- [ ] CLI skeleton with Cobra

### Phase 2: Data Collection (Week 2-3)
- [ ] Complete post archival
- [ ] Media download functionality
- [ ] Profile data collection
- [ ] Incremental sync logic
- [ ] Error handling & retry logic

### Phase 3: Search & Storage (Week 3-4)
- [ ] SQLite indexing
- [ ] Full-text search with bleve
- [ ] Advanced filtering
- [ ] Data deduplication
- [ ] Archive versioning

### Phase 4: Export Functionality (Week 4-5)
- [ ] JSON exporter
- [ ] Markdown exporter
- [ ] HTML static site generator
- [ ] CSV exporter
- [ ] Export templates

### Phase 5: Web Interface (Week 5-7)
- [ ] Basic HTTP server
- [ ] Dashboard with statistics
- [ ] Timeline view
- [ ] Search interface
- [ ] Settings page
- [ ] Responsive design

### Phase 6: Automation & Polish (Week 7-8)
- [ ] Background scheduler
- [ ] Automatic incremental backups
- [ ] Notifications system
- [ ] Archive verification
- [ ] Documentation
- [ ] Docker support

## Configuration File Example

```yaml
# config.yaml
accounts:
  - did: "did:plc:example123"
    handle: "user.bsky.social"
    enabled: true
    
archive:
  base_path: "./archive"
  download_media: true
  media_quality: "high"  # high, medium, low
  
sync:
  interval: "24h"
  incremental: true
  max_retries: 3
  
search:
  index_type: "bleve"  # bleve or sqlite-fts
  
export:
  default_format: "json"
  
web:
  port: 8080
  host: "localhost"
  enable_auth: false  # Basic auth for web interface
  
notifications:
  email:
    enabled: false
    smtp_host: ""
    smtp_port: 587
  webhook:
    enabled: false
    url: ""
```

## Security Considerations

1. **OAuth Tokens**
   - Store refresh tokens securely
   - Encrypt sensitive data at rest
   - Use system keyring where possible

2. **Web Interface**
   - Optional basic authentication
   - CSRF protection
   - Rate limiting on API endpoints

3. **Data Privacy**
   - All data stored locally
   - No third-party services
   - User controls all data

## Future Enhancements

- **Multi-platform support**: Twitter, Mastodon archives
- **Data visualization**: Charts and graphs of posting patterns
- **Import/Export**: Migrate between instances
- **Comparison tools**: Compare archives over time
- **AI features**: Sentiment analysis, topic extraction
- **Sharing**: Generate shareable timeline snippets
- **Mobile app**: Companion mobile application

## Success Metrics

- Successfully archive 100% of user posts
- Support archives of 10,000+ posts efficiently
- Search results in <100ms for typical queries
- Generate exports in <30 seconds for typical archives
- Web interface loads in <2 seconds

## Getting Started (For Users)

```bash
# Install
go install github.com/yourusername/bluesky-archiver@latest

# Initialize
bluesky-archiver init

# Authenticate
bluesky-archiver auth add

# First sync
bluesky-archiver sync --full

# Start web interface
bluesky-archiver serve

# Visit http://localhost:8080
```

## Resources & References

- [AT Protocol Documentation](https://atproto.com/)
- [Bluesky API Docs](https://docs.bsky.app/)
- [bskyoauth Library](https://github.com/shindakun/bskyoauth)
- [Bleve Search](https://github.com/blevesearch/bleve)
- [Similar Projects for inspiration]
  - Twitter Archive Browser
  - Mastodon Archive Viewer

---

**Ready to start building? Let's begin with Phase 1!**